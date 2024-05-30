package main

import (
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"tipp.casualcoding.com/internal/models"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"

	_ "github.com/go-sql-driver/mysql"
)

// application-wide dependencies
type application struct {
	logger         *slog.Logger
	matches        *models.MatchModel
	tipps          *models.TippModel
	matchTipps     *models.MatchTippModel
	templateCache  map[string]*template.Template
	sessionManager *scs.SessionManager
}

func main() {
	// CLI flags
	addr := flag.String("addr", ":8090", "HTTP network address")
	dsn := flag.String("dsn", "user:pass@/dbname?parseTime=true", "MySQL data source name")
	flag.Parse()

	// logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// database connection pool
	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// session management
	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	// setup for dependency injection across our app
	matchModel := models.MatchModel{DB: db}
	tippModel := models.TippModel{DB: db}
	matchTippModel := models.MatchTippModel{DB: db, MatchModel: &matchModel, TippModel: &tippModel}
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	app := &application{
		logger:         logger,
		matches:        &matchModel,
		tipps:          &tippModel,
		matchTipps:     &matchTippModel,
		templateCache:  templateCache,
		sessionManager: sessionManager,
	}

	srv := &http.Server{
		Addr:     *addr,
		Handler:  app.routes(),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// start server
	logger.Info("Starting server", "addr", srv.Addr)

	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
