package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"tipp.casualcoding.com/internal/models"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"

	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

// application-wide dependencies
type application struct {
	logger         *slog.Logger
	matches        *models.MatchModel
	tipps          *models.TippModel
	matchTipps     *models.MatchTippModel
	users          *models.UserModel
	groups         *models.GroupModel
	goals          *models.GoalModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	// CLI flags
	addr := flag.String("addr", ":8090", "HTTP network address")
	dsn := flag.String("dsn", "user:pass@/dbname?parseTime=true", "MySQL data source name")
	https := flag.Bool("https", false, "Enable TLS for https")
	flag.Parse()

	// logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Strip the "mysql://" prefix
	if strings.HasPrefix(*dsn, "mysql://") {
		*dsn = strings.TrimPrefix(*dsn, "mysql://")
	}

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
	sessionManager.Lifetime = 24 * 30 * time.Hour

	// setup for dependency injection across our app
	matchModel := models.MatchModel{DB: db}
	tippModel := models.TippModel{DB: db}
	matchTippModel := models.MatchTippModel{DB: db, MatchModel: &matchModel, TippModel: &tippModel}
	formDecoder := form.NewDecoder()
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
		users:          &models.UserModel{DB: db},
		groups:         &models.GroupModel{DB: db},
		goals:          &models.GoalModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	// restrict TLS to only allow elliptic curves with efficient implementations, avoids potential load on sersver
	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// start server
	logger.Info("Starting server", "addr", srv.Addr)

	if *https {
		err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	} else {
		err = srv.ListenAndServe()
	}
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
