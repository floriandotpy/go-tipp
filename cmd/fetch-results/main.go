package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"tipp.casualcoding.com/internal/api"
	"tipp.casualcoding.com/internal/models"
)

func main() {
	dsn := flag.String("dsn", "", "MySQL data source name")
	flag.Parse()

	if *dsn == "" {
		*dsn = os.Getenv("DATABASE_URL_GO")
	}
	if *dsn == "" {
		log.Fatal("dsn flag or DATABASE_URL_GO env var is required")
	}

	db, err := openDB(*dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	os.Exit(run(db))
}

func run(db *sql.DB) int {
	eventModel := &models.EventModel{DB: db}
	matchModel := &models.MatchModel{DB: db}
	goalModel := &models.GoalModel{DB: db}
	tippModel := &models.TippModel{DB: db}

	// Get active event
	event, err := eventModel.GetActive()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			fmt.Println("No active event, nothing to do.")
			return 0
		}
		fmt.Printf("Error fetching active event: %v\n", err)
		return 1
	}
	fmt.Printf("Active event: %s (ID: %d)\n", event.Name, event.ID)

	// Early exit: check if any match is currently in progress
	hasLive, err := matchModel.HasLiveMatch(event.ID)
	if err != nil {
		fmt.Printf("Error checking for live matches: %v\n", err)
		return 1
	}
	if !hasLive {
		fmt.Println("No live matches, nothing to do.")
		return 0
	}

	// Fetch all match data from the event's API
	fmt.Printf("Fetching data from: %s\n", event.ApiBaseURL)
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		fmt.Printf("Error fetching API data: %v\n", err)
		return 1
	}

	// Filter to only matches that have started and are not yet finished in the API response.
	// This avoids processing future matches or already-finished ones unnecessarily.
	now := time.Now()
	var relevant []api.ApiMatch
	for _, am := range apiMatches {
		matchTime, parseErr := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
		if parseErr != nil {
			continue
		}
		// Only process matches that have started (or started today within a reasonable window)
		if matchTime.Before(now) || am.MatchIsFinished {
			relevant = append(relevant, am)
		}
	}
	fmt.Printf("Found %d relevant matches (started or finished) out of %d total\n", len(relevant), len(apiMatches))

	var recomputeUserScores bool

	for _, apiMatch := range relevant {
		// Look up the match in our DB by API match ID
		dbMatch, err := matchModel.GetByApiMatchID(apiMatch.MatchID)
		if err != nil {
			fmt.Printf("Error looking up api_match_id %d: %v\n", apiMatch.MatchID, err)
			continue
		}
		if dbMatch.ID == 0 {
			// Not in our database (could be from a phase we haven't synced yet) — skip
			continue
		}

		// Skip if already marked finished in our DB and API agrees
		if dbMatch.Finished && apiMatch.MatchIsFinished {
			continue
		}

		fmt.Printf("Processing: %s vs %s (db_id=%d, api_id=%d)\n",
			dbMatch.TeamA, dbMatch.TeamB, dbMatch.ID, apiMatch.MatchID)

		// Sync goals: delete all then re-insert (handles VAR reversals)
		goalModel.DeleteAllForMatch(dbMatch.ID)
		for _, apiGoal := range apiMatch.Goals {
			goal := api.ConvertApiGoalToGoal(apiGoal)
			_, err := goalModel.InsertOrUpdate(dbMatch.ID, goal)
			if err != nil {
				fmt.Printf("  Error inserting goal: %v\n", err)
			}
		}
		if len(apiMatch.Goals) > 0 {
			lastGoal := apiMatch.Goals[len(apiMatch.Goals)-1]
			fmt.Printf("  Goals synced: %d total, current score %d:%d\n",
				len(apiMatch.Goals), lastGoal.ScoreTeamA, lastGoal.ScoreTeamB)
		}

		// Parse results from API response
		results := parseResults(apiMatch)

		// Set end result
		if end, ok := results["Endergebnis"]; ok {
			if dbMatch.ResultA == nil || dbMatch.ResultB == nil || *dbMatch.ResultA != end[0] || *dbMatch.ResultB != end[1] {
				fmt.Printf("  Setting result: %d:%d\n", end[0], end[1])
				matchModel.SetResults(dbMatch.ID, end[0], end[1])
			}
		}

		// Set result after extra time
		if aet, ok := results["nach Verlängerung"]; ok {
			if dbMatch.ResultAETA == nil || dbMatch.ResultAETB == nil || *dbMatch.ResultAETA != aet[0] || *dbMatch.ResultAETB != aet[1] {
				fmt.Printf("  Setting AET result: %d:%d\n", aet[0], aet[1])
				matchModel.SetResultsAfterExtension(dbMatch.ID, aet[0], aet[1])
			}
		}

		// Set result after penalty shootout
		if apen, ok := results["nach Elfmeterschießen"]; ok {
			if dbMatch.ResultAPenA == nil || dbMatch.ResultAPenB == nil || *dbMatch.ResultAPenA != apen[0] || *dbMatch.ResultAPenB != apen[1] {
				fmt.Printf("  Setting penalty result: %d:%d\n", apen[0], apen[1])
				matchModel.SetResultsAfterPenalty(dbMatch.ID, apen[0], apen[1])
			}
		}

		// Update finished flag
		if dbMatch.Finished != apiMatch.MatchIsFinished {
			fmt.Printf("  Marking match as finished=%t\n", apiMatch.MatchIsFinished)
			matchModel.SetMatchIsFinished(dbMatch.ID, apiMatch.MatchIsFinished)
			recomputeUserScores = true
		}
	}

	// Recompute user scores if any match was newly marked finished
	if recomputeUserScores {
		fmt.Println("Recomputing user scores...")
		rowsAffected, err := tippModel.UpdatePoints(event.ID)
		if err != nil {
			fmt.Printf("Error updating points: %v\n", err)
			return 1
		}
		fmt.Printf("Updated %d tipp entries\n", rowsAffected)
	} else {
		fmt.Println("No matches newly finished, scores unchanged.")
	}

	return 0
}

// parseResults extracts the relevant result types from the API match response,
// applying the same cleanup logic as the old CLI.
func parseResults(apiMatch api.ApiMatch) map[string][2]int {
	results := make(map[string][2]int)
	relevantNames := []string{"Endergebnis", "nach Verlängerung", "nach Elfmeterschießen"}

	for _, result := range apiMatch.MatchResults {
		for _, name := range relevantNames {
			if result.ResultName == name {
				results[name] = [2]int{result.PointsTeamA, result.PointsTeamB}
			}
		}
	}

	// Cleanup: ignore penalty result if it's the same as the end result
	if apen, ok := results["nach Elfmeterschießen"]; ok {
		if end, ok2 := results["Endergebnis"]; ok2 {
			if apen[0] == end[0] && apen[1] == end[1] {
				delete(results, "nach Elfmeterschießen")
			}
		}
	}

	return results
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
