package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"tipp.casualcoding.com/internal/models"
)

type ApiMatch struct {
	MatchDateTime   string      `json:"matchDateTime"`
	TeamA           ApiTeam     `json:"team1"`
	TeamB           ApiTeam     `json:"team2"`
	MatchResults    []ApiResult `json:"matchResults"`
	MatchIsFinished bool        `json:"matchIsFinished"`
}

type ApiTeam struct {
	TeamName string `json:"teamName"`
}

type ApiResult struct {
	ResultName  string `json:"resultName"`
	PointsTeamA int    `json:"pointsTeam1"`
	PointsTeamB int    `json:"pointsTeam2"`
}

func fetchMatchData(url string) ([]ApiMatch, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data: %s", resp.Status)
	}

	var matches []ApiMatch
	err = json.NewDecoder(resp.Body).Decode(&matches)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

func main() {
	dsn := flag.String("dsn", "user:pass@/dbname?parseTime=true", "MySQL data source name")
	flag.Parse()

	// URL to fetch data from
	url := "https://api.openligadb.de/getmatchdata/em/2024/1"

	// Fetch match data
	matches, err := fetchMatchData(url)
	if err != nil {
		log.Fatal(err)
	}

	// database connection pool
	db, err := openDB(*dsn)
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Create a MatchModel instance
	matchModel := &models.MatchModel{DB: db}
	tippModel := &models.TippModel{DB: db}

	var dbUpdated = false

	for _, apiMatch := range matches {
		// Parse match date and time
		matchTime, err := time.Parse("2006-01-02T15:04:05", apiMatch.MatchDateTime)
		if err != nil {
			log.Fatal(err)
		}

		// Extract end score from match results
		var endScoreTeamA, endScoreTeamB int
		if len(apiMatch.MatchResults) == 0 || !apiMatch.MatchIsFinished {
			fmt.Printf("Skipping match (%s vs %s) without result...\n", apiMatch.TeamA.TeamName, apiMatch.TeamB.TeamName)
			continue
		}
		for _, result := range apiMatch.MatchResults {
			if strings.ToLower(result.ResultName) == "endergebnis" {
				endScoreTeamA = result.PointsTeamA
				endScoreTeamB = result.PointsTeamB
				break
			}
		}

		// Output extracted information
		dayString := matchTime.Format("2006-01-02")
		// timeString := matchTime.Format("15:04")
		fmt.Printf("Day of the match: %s\n", dayString)
		// fmt.Printf("Time of the match: %s\n", timeString)
		fmt.Printf("Name of team 1: %s\n", apiMatch.TeamA.TeamName)
		fmt.Printf("Name of team 2: %s\n", apiMatch.TeamB.TeamName)
		fmt.Printf("End score of team 1: %d\n", endScoreTeamA)
		fmt.Printf("End score of team 2: %d\n", endScoreTeamB)

		// Call the GetByMetadata function
		dbMatch, err := matchModel.GetByMetadata(dayString, apiMatch.TeamA.TeamName, apiMatch.TeamB.TeamName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Check if a match was found
		if dbMatch.ID != 0 {
			fmt.Printf("Match found: %d\n", dbMatch.ID)

			if dbMatch.ResultA == nil && dbMatch.ResultB == nil {
				fmt.Printf("-> Update result to %d:%d\n", endScoreTeamA, endScoreTeamB)
				matchModel.SetResults(dbMatch.ID, endScoreTeamA, endScoreTeamB)
				dbUpdated = true
			} else if *dbMatch.ResultA != endScoreTeamA || *dbMatch.ResultB != endScoreTeamB {
				fmt.Printf("Warning: Score mismatch API (%d:%d) vs DB (%d:%d)\n", *dbMatch.ResultA, *dbMatch.ResultB, endScoreTeamA, endScoreTeamB)
			} else {
				fmt.Printf("Existing result won't be updated, score is %d:%d\n", *dbMatch.ResultA, *dbMatch.ResultB)
			}
		}

		fmt.Printf("\n")
	}

	fmt.Printf("\n")

	if dbUpdated {
		fmt.Printf("Trigger point update for all user tipps...\n")
		rowsAffected, err := tippModel.UpdatePoints()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Done, updated %d db entries\n", rowsAffected)
	} else {
		fmt.Printf("No database updated occured, no user points were affected\n")
	}
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
