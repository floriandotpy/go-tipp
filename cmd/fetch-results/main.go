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
	ID          int    `json:"id"`
	Teams       string `json:"teams"`
	GoalsSynced int    `json:"goals_synced"`
	ResultSet   string `json:"result_set,omitempty"`
	Finished    bool   `json:"finished"`
}

type fetchDetails struct {
	Matches          []matchDetail `json:"matches"`
	PointsRecomputed bool          `json:"points_recomputed"`
}

func main() {
	dsn := flag.String("dsn", "", "MySQL data source name")
	all := flag.Bool("all", false, "reconcile all started/finished matches of the active event, bypassing the live/recently-finished gate (useful for backfilling result corrections)")
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

	os.Exit(run(db, *all))
}

// run processes match results for the active event. When reconcileAll is true,
// the live/recently-finished early-exit gate is skipped so that every
// started/finished match is re-checked and corrected — used to backfill result
// fixes (e.g. after a result-parsing bug) outside the normal 24h window.
func run(db *sql.DB, reconcileAll bool) int {
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

	// Early exit: check if any match is live or finished within the last 24h.
	// Skipped when reconcileAll is set, so all matches get re-checked.
	if reconcileAll {
		fmt.Println("Reconcile-all mode: bypassing live/recently-finished gate.")
	} else {
		hasLive, err := matchModel.HasLiveMatch(event.ID)
		if err != nil {
			fmt.Printf("Error checking for live matches: %v\n", err)
			record(models.JobStatusError, fmt.Sprintf("Error checking for live matches: %v", err), nil)
			return 1
		}
		hasRecentlyFinished, err := matchModel.HasRecentlyFinishedMatch(event.ID)
		if err != nil {
			fmt.Printf("Error checking for recently finished matches: %v\n", err)
			record(models.JobStatusError, fmt.Sprintf("Error checking for recently finished matches: %v", err), nil)
			return 1
		}
		if !hasLive && !hasRecentlyFinished {
			fmt.Println("No live or recently finished matches, nothing to do.")
			record(models.JobStatusNoop, "No live or recently finished matches", nil)
			return 0
		}
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
	now := time.Now().UTC()
	var relevant []api.ApiMatch
	for _, am := range apiMatches {
		// Use UTC time from API for accurate comparison (matchDateTime is local/CEST)
		matchTime, parseErr := time.Parse(time.RFC3339, am.MatchDateTimeUTC)
		if parseErr != nil {
			// Fallback: try parsing local time as Europe/Berlin
			matchTime, parseErr = time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
			if parseErr != nil {
				continue
			}
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

		// Skip if already marked finished in our DB and API agrees,
		// BUT only if the result also matches (API may correct scores after marking finished).
		if dbMatch.Finished && apiMatch.MatchIsFinished {
			apiResults := api.ExtractMatchResults(apiMatch)
			if regular, ok := apiResults[api.ResultRegularTime]; ok {
				if dbMatch.ResultA != nil && dbMatch.ResultB != nil &&
					*dbMatch.ResultA == regular[0] && *dbMatch.ResultB == regular[1] {
					continue
				}
				// Result mismatch or missing — fall through to update it
				if dbMatch.ResultA != nil && dbMatch.ResultB != nil {
					fmt.Printf("Result correction needed: %s vs %s (db=%d:%d, api=%d:%d)\n",
						dbMatch.TeamA, dbMatch.TeamB, *dbMatch.ResultA, *dbMatch.ResultB, regular[0], regular[1])
				} else {
					fmt.Printf("Result missing in DB: %s vs %s (api=%d:%d)\n",
						dbMatch.TeamA, dbMatch.TeamB, regular[0], regular[1])
				}
			} else {
				// No regular-time result in API but already finished — skip
				continue
			}
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
		results := api.ExtractMatchResults(apiMatch)

		// Determine result string for detail logging
		var resultStr string
		var resultChanged bool

		// Set regular-time result used for scoring.
		if regular, ok := results[api.ResultRegularTime]; ok {
			if dbMatch.ResultA == nil || dbMatch.ResultB == nil || *dbMatch.ResultA != regular[0] || *dbMatch.ResultB != regular[1] {
				fmt.Printf("  Setting regular-time result: %d:%d\n", regular[0], regular[1])
				matchModel.SetResults(dbMatch.ID, regular[0], regular[1])
				resultChanged = true
			}
			resultStr = fmt.Sprintf("%d:%d", regular[0], regular[1])
		}

		// Set result after extra time
		if aet, ok := results[api.ResultAfterExtraTime]; ok {
			if dbMatch.ResultAETA == nil || dbMatch.ResultAETB == nil || *dbMatch.ResultAETA != aet[0] || *dbMatch.ResultAETB != aet[1] {
				fmt.Printf("  Setting AET result: %d:%d\n", aet[0], aet[1])
				matchModel.SetResultsAfterExtension(dbMatch.ID, aet[0], aet[1])
				resultChanged = true
			}
			resultStr = fmt.Sprintf("%d:%d AET", aet[0], aet[1])
		}

		// Set result after penalty shootout
		if apen, ok := results[api.ResultAfterPenalties]; ok {
			if dbMatch.ResultAPenA == nil || dbMatch.ResultAPenB == nil || *dbMatch.ResultAPenA != apen[0] || *dbMatch.ResultAPenB != apen[1] {
				fmt.Printf("  Setting penalty result: %d:%d\n", apen[0], apen[1])
				matchModel.SetResultsAfterPenalty(dbMatch.ID, apen[0], apen[1])
				resultChanged = true
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

		// Also recompute scores if a result was corrected on an already-finished match
		if resultChanged && dbMatch.Finished {
			recomputeUserScores = true
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
