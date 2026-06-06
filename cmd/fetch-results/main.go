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

// details structs for JSON logging
type matchDetail struct {
	ID         int    `json:"id"`
	Teams      string `json:"teams"`
	GoalsSynced int   `json:"goals_synced"`
	ResultSet  string `json:"result_set,omitempty"`
	Finished   bool   `json:"finished"`
}

type fetchDetails struct {
	Matches          []matchDetail `json:"matches"`
	PointsRecomputed bool          `json:"points_recomputed"`
}

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
	startedAt := time.Now()

	eventModel := &models.EventModel{DB: db}
	matchModel := &models.MatchModel{DB: db}
	goalModel := &models.GoalModel{DB: db}
	tippModel := &models.TippModel{DB: db}
	jobRunModel := &models.JobRunModel{DB: db}

	// Cleanup old job runs (>30 days)
	jobRunModel.DeleteOlderThan(30)

	// Helper to record a job run and exit
	record := func(status, summary string, details interface{}) {
		finishedAt := time.Now()
		err := jobRunModel.Insert(models.JobFetchResults, status, summary, details, startedAt, finishedAt)
		if err != nil {
			fmt.Printf("Warning: failed to record job run: %v\n", err)
		}
	}

	// Get active event
	event, err := eventModel.GetActive()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			fmt.Println("No active event, nothing to do.")
			record(models.JobStatusNoop, "No active event", nil)
			return 0
		}
		fmt.Printf("Error fetching active event: %v\n", err)
		record(models.JobStatusError, fmt.Sprintf("Error fetching active event: %v", err), nil)
		return 1
	}
	fmt.Printf("Active event: %s (ID: %d)\n", event.Name, event.ID)

	// Early exit: check if any match is currently in progress
	hasLive, err := matchModel.HasLiveMatch(event.ID)
	if err != nil {
		fmt.Printf("Error checking for live matches: %v\n", err)
		record(models.JobStatusError, fmt.Sprintf("Error checking for live matches: %v", err), nil)
		return 1
	}
	if !hasLive {
		fmt.Println("No live matches, nothing to do.")
		record(models.JobStatusNoop, "No live matches", nil)
		return 0
	}

	// Fetch all match data from the event's API
	fmt.Printf("Fetching data from: %s\n", event.ApiBaseURL)
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		fmt.Printf("Error fetching API data: %v\n", err)
		record(models.JobStatusError, fmt.Sprintf("Error fetching API data: %v", err), nil)
		return 1
	}

	// Filter to only matches that have started
	now := time.Now()
	var relevant []api.ApiMatch
	for _, am := range apiMatches {
		matchTime, parseErr := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
		if parseErr != nil {
			continue
		}
		if matchTime.Before(now) || am.MatchIsFinished {
			relevant = append(relevant, am)
		}
	}
	fmt.Printf("Found %d relevant matches (started or finished) out of %d total\n", len(relevant), len(apiMatches))

	var recomputeUserScores bool
	var matchDetails []matchDetail

	for _, apiMatch := range relevant {
		// Look up the match in our DB by API match ID
		dbMatch, err := matchModel.GetByApiMatchID(apiMatch.MatchID)
		if err != nil {
			fmt.Printf("Error looking up api_match_id %d: %v\n", apiMatch.MatchID, err)
			continue
		}
		if dbMatch.ID == 0 {
			continue
		}

		// Skip if already marked finished in our DB and API agrees
		if dbMatch.Finished && apiMatch.MatchIsFinished {
			continue
		}

		fmt.Printf("Processing: %s vs %s (db_id=%d, api_id=%d)\n",
			dbMatch.TeamA, dbMatch.TeamB, dbMatch.ID, apiMatch.MatchID)

		// Sync goals: delete all then re-insert
		goalModel.DeleteAllForMatch(dbMatch.ID)
		for _, apiGoal := range apiMatch.Goals {
			goal := api.ConvertApiGoalToGoal(apiGoal)
			_, err := goalModel.InsertOrUpdate(dbMatch.ID, goal)
			if err != nil {
				fmt.Printf("  Error inserting goal: %v\n", err)
			}
		}

		// Parse results from API response
		results := parseResults(apiMatch)

		// Determine result string for detail logging
		var resultStr string

		// Set end result
		if end, ok := results["Endergebnis"]; ok {
			if dbMatch.ResultA == nil || dbMatch.ResultB == nil || *dbMatch.ResultA != end[0] || *dbMatch.ResultB != end[1] {
				fmt.Printf("  Setting result: %d:%d\n", end[0], end[1])
				matchModel.SetResults(dbMatch.ID, end[0], end[1])
			}
			resultStr = fmt.Sprintf("%d:%d", end[0], end[1])
		}

		// Set result after extra time
		if aet, ok := results["nach Verlängerung"]; ok {
			if dbMatch.ResultAETA == nil || dbMatch.ResultAETB == nil || *dbMatch.ResultAETA != aet[0] || *dbMatch.ResultAETB != aet[1] {
				fmt.Printf("  Setting AET result: %d:%d\n", aet[0], aet[1])
				matchModel.SetResultsAfterExtension(dbMatch.ID, aet[0], aet[1])
			}
			resultStr = fmt.Sprintf("%d:%d AET", aet[0], aet[1])
		}

		// Set result after penalty shootout
		if apen, ok := results["nach Elfmeterschießen"]; ok {
			if dbMatch.ResultAPenA == nil || dbMatch.ResultAPenB == nil || *dbMatch.ResultAPenA != apen[0] || *dbMatch.ResultAPenB != apen[1] {
				fmt.Printf("  Setting penalty result: %d:%d\n", apen[0], apen[1])
				matchModel.SetResultsAfterPenalty(dbMatch.ID, apen[0], apen[1])
			}
			resultStr = fmt.Sprintf("%d:%d PEN", apen[0], apen[1])
		}

		// Update finished flag
		newlyFinished := false
		if dbMatch.Finished != apiMatch.MatchIsFinished {
			fmt.Printf("  Marking match as finished=%t\n", apiMatch.MatchIsFinished)
			matchModel.SetMatchIsFinished(dbMatch.ID, apiMatch.MatchIsFinished)
			recomputeUserScores = true
			newlyFinished = apiMatch.MatchIsFinished
		}

		matchDetails = append(matchDetails, matchDetail{
			ID:          dbMatch.ID,
			Teams:       fmt.Sprintf("%s vs %s", dbMatch.TeamA, dbMatch.TeamB),
			GoalsSynced: len(apiMatch.Goals),
			ResultSet:   resultStr,
			Finished:    newlyFinished,
		})
	}

	// Recompute user scores if any match was newly marked finished
	if recomputeUserScores {
		fmt.Println("Recomputing user scores...")
		rowsAffected, err := tippModel.UpdatePoints(event.ID)
		if err != nil {
			fmt.Printf("Error updating points: %v\n", err)
			record(models.JobStatusError, fmt.Sprintf("Error updating points: %v", err), nil)
			return 1
		}
		fmt.Printf("Updated %d tipp entries\n", rowsAffected)
	} else {
		fmt.Println("No matches newly finished, scores unchanged.")
	}

	// Record job run
	if len(matchDetails) == 0 {
		record(models.JobStatusNoop, "No matches required updates", nil)
	} else {
		totalGoals := 0
		finishedCount := 0
		for _, md := range matchDetails {
			totalGoals += md.GoalsSynced
			if md.Finished {
				finishedCount++
			}
		}
		summary := fmt.Sprintf("%d matches processed, %d goals synced", len(matchDetails), totalGoals)
		if finishedCount > 0 {
			summary += fmt.Sprintf(", %d newly finished", finishedCount)
		}
		record(models.JobStatusChanged, summary, fetchDetails{
			Matches:          matchDetails,
			PointsRecomputed: recomputeUserScores,
		})
	}

	return 0
}

// parseResults extracts the relevant result types from the API match response.
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
