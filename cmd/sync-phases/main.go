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
	appsync "tipp.casualcoding.com/internal/sync"
)

// details structs for JSON logging
type phaseDetail struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

type matchInsertDetail struct {
	ApiMatchID int    `json:"api_match_id"`
	Teams      string `json:"teams"`
}

type matchUpdateDetail struct {
	ApiMatchID int    `json:"api_match_id"`
	Teams      string `json:"teams"`
	Change     string `json:"change"`
}

type syncDetails struct {
	PhasesCreated  []phaseDetail       `json:"phases_created,omitempty"`
	PhasesUpdated  []phaseDetail       `json:"phases_updated,omitempty"`
	MatchesInserted []matchInsertDetail `json:"matches_inserted,omitempty"`
	MatchesUpdated  []matchUpdateDetail `json:"matches_updated,omitempty"`
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
	eventPhaseModel := &models.EventPhaseModel{DB: db}
	matchModel := &models.MatchModel{DB: db}
	jobRunModel := &models.JobRunModel{DB: db}

	// Cleanup old job runs (>30 days)
	jobRunModel.DeleteOlderThan(30)

	// Helper to record a job run
	record := func(status, summary string, details interface{}) {
		finishedAt := time.Now()
		err := jobRunModel.Insert(models.JobSyncPhases, status, summary, details, startedAt, finishedAt)
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

	// Fetch all match data from the event's API
	fmt.Printf("Fetching data from: %s\n", event.ApiBaseURL)
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		fmt.Printf("Error fetching API data: %v\n", err)
		record(models.JobStatusError, fmt.Sprintf("Error fetching API data: %v", err), nil)
		return 1
	}
	fmt.Printf("Fetched %d matches from API\n", len(apiMatches))

	if len(apiMatches) == 0 {
		fmt.Println("No matches returned by API, nothing to sync.")
		record(models.JobStatusNoop, "No matches returned by API", nil)
		return 0
	}

	// Group matches by GroupOrderID (= phase number)
	grouped := appsync.GroupMatches(apiMatches)

	var details syncDetails

	for groupOrderID, groupMatches := range grouped {
		groupName := groupMatches[0].Group.GroupName

		// Build phase from group
		phase, err := appsync.PhaseFromGroup(event.ID, groupOrderID, groupName, groupMatches)
		if err != nil {
			msg := fmt.Sprintf("Error building phase for group %d (%s): %v", groupOrderID, groupName, err)
			fmt.Println(msg)
			record(models.JobStatusError, msg, nil)
			return 1
		}

		// Upsert phase — check for actual changes first
		existing, existErr := eventPhaseModel.GetByEventAndNumber(event.ID, groupOrderID)
		phaseChanged := false
		if existErr == nil {
			// Phase exists — check if anything actually differs
			if existing.Title != phase.Title || existing.PhaseType != phase.PhaseType || !existing.Start.Equal(phase.Start) || !existing.End.Equal(phase.End) {
				phaseChanged = true
			}
		}

		_, isNew, err := eventPhaseModel.Upsert(phase)
		if err != nil {
			msg := fmt.Sprintf("Error upserting phase %d (%s): %v", groupOrderID, groupName, err)
			fmt.Println(msg)
			record(models.JobStatusError, msg, nil)
			return 1
		}
		if isNew {
			fmt.Printf("  Created phase: %s (number %d)\n", groupName, groupOrderID)
			details.PhasesCreated = append(details.PhasesCreated, phaseDetail{Number: groupOrderID, Title: groupName})
		} else if phaseChanged {
			fmt.Printf("  Updated phase: %s (number %d)\n", groupName, groupOrderID)
			details.PhasesUpdated = append(details.PhasesUpdated, phaseDetail{Number: groupOrderID, Title: groupName})
		}

		// Upsert matches by API match ID
		for _, am := range groupMatches {
			parsedTime, parseErr := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
			if parseErr != nil {
				fmt.Printf("  Error parsing match time: %v\n", parseErr)
				continue
			}

			existing, err := matchModel.GetByApiMatchID(am.MatchID)
			if err != nil {
				fmt.Printf("  Error looking up api_match_id %d: %v\n", am.MatchID, err)
				continue
			}

			teams := fmt.Sprintf("%s vs %s", am.TeamA.TeamName, am.TeamB.TeamName)

			if existing.ID != 0 {
				// Match exists — update if team names or start time changed
				if existing.TeamA != am.TeamA.TeamName || existing.TeamB != am.TeamB.TeamName || !existing.Start.Equal(parsedTime) {
					var change string
					if existing.TeamA != am.TeamA.TeamName || existing.TeamB != am.TeamB.TeamName {
						change = "team names"
					}
					if !existing.Start.Equal(parsedTime) {
						if change != "" {
							change += ", "
						}
						change += "start time"
					}

					err = matchModel.UpdateMatch(existing.ID, am.TeamA.TeamName, am.TeamB.TeamName, parsedTime, phase.PhaseType, groupOrderID)
					if err != nil {
						fmt.Printf("  Error updating match %d: %v\n", existing.ID, err)
						continue
					}
					fmt.Printf("  Updated match: %s (%s)\n", teams, parsedTime.Format("2006-01-02 15:04"))
					details.MatchesUpdated = append(details.MatchesUpdated, matchUpdateDetail{
						ApiMatchID: am.MatchID,
						Teams:      teams,
						Change:     change,
					})
				}
				continue
			}

			// Insert new match
			_, err = matchModel.Insert(
				am.TeamA.TeamName, am.TeamB.TeamName,
				parsedTime, phase.PhaseType, groupOrderID, event.ID, am.MatchID,
			)
			if err != nil {
				fmt.Printf("  Error inserting match: %v\n", err)
				continue
			}
			fmt.Printf("  Inserted match: %s (%s)\n", teams, parsedTime.Format("2006-01-02 15:04"))
			details.MatchesInserted = append(details.MatchesInserted, matchInsertDetail{
				ApiMatchID: am.MatchID,
				Teams:      teams,
			})
		}
	}

	// Determine status and summary
	hasChanges := len(details.PhasesCreated) > 0 || len(details.PhasesUpdated) > 0 || len(details.MatchesInserted) > 0 || len(details.MatchesUpdated) > 0
	summary := fmt.Sprintf("%d phases created, %d updated, %d matches inserted, %d matches updated",
		len(details.PhasesCreated), len(details.PhasesUpdated),
		len(details.MatchesInserted), len(details.MatchesUpdated))
	fmt.Printf("\nSync complete: %s\n", summary)

	if hasChanges {
		record(models.JobStatusChanged, summary, details)
	} else {
		record(models.JobStatusNoop, "No changes", nil)
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
