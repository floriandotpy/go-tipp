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
	eventPhaseModel := &models.EventPhaseModel{DB: db}
	matchModel := &models.MatchModel{DB: db}

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

	// Fetch all match data from the event's API
	fmt.Printf("Fetching data from: %s\n", event.ApiBaseURL)
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		fmt.Printf("Error fetching API data: %v\n", err)
		return 1
	}
	fmt.Printf("Fetched %d matches from API\n", len(apiMatches))

	if len(apiMatches) == 0 {
		fmt.Println("No matches returned by API, nothing to sync.")
		return 0
	}

	// Group matches by GroupOrderID (= phase number)
	grouped := appsync.GroupMatches(apiMatches)

	var phasesCreated, phasesUpdated, matchesInserted, matchesUpdated int

	for groupOrderID, groupMatches := range grouped {
		groupName := groupMatches[0].Group.GroupName

		// Build phase from group
		phase, err := appsync.PhaseFromGroup(event.ID, groupOrderID, groupName, groupMatches)
		if err != nil {
			fmt.Printf("Error building phase for group %d (%s): %v\n", groupOrderID, groupName, err)
			return 1
		}

		// Upsert phase
		_, isNew, err := eventPhaseModel.Upsert(phase)
		if err != nil {
			fmt.Printf("Error upserting phase %d (%s): %v\n", groupOrderID, groupName, err)
			return 1
		}
		if isNew {
			fmt.Printf("  Created phase: %s (number %d)\n", groupName, groupOrderID)
			phasesCreated++
		} else {
			phasesUpdated++
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

			if existing.ID != 0 {
				// Match exists — update if team names or start time changed
				if existing.TeamA != am.TeamA.TeamName || existing.TeamB != am.TeamB.TeamName || !existing.Start.Equal(parsedTime) {
					err = matchModel.UpdateMatch(existing.ID, am.TeamA.TeamName, am.TeamB.TeamName, parsedTime, phase.PhaseType, groupOrderID)
					if err != nil {
						fmt.Printf("  Error updating match %d: %v\n", existing.ID, err)
						continue
					}
					fmt.Printf("  Updated match: %s vs %s (%s)\n", am.TeamA.TeamName, am.TeamB.TeamName, parsedTime.Format("2006-01-02 15:04"))
					matchesUpdated++
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
			fmt.Printf("  Inserted match: %s vs %s (%s)\n", am.TeamA.TeamName, am.TeamB.TeamName, parsedTime.Format("2006-01-02 15:04"))
			matchesInserted++
		}
	}

	fmt.Printf("\nSync complete: %d phases created, %d updated, %d matches inserted, %d matches updated\n",
		phasesCreated, phasesUpdated, matchesInserted, matchesUpdated)
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
