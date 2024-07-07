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
	Goals           []ApiGoal   `json:"goals"`
}

type ApiGoal struct {
	ScoreTeamA     int     `json:"scoreTeam1"`
	ScoreTeamB     int     `json:"scoreTeam2"`
	MatchMinute    int     `json:"matchMinute"`
	GoalGetterID   int     `json:"goalGetterID"`
	GoalGetterName string  `json:"goalGetterName"`
	IsPenalty      bool    `json:"isPenalty"`
	IsOwnGoal      bool    `json:"isOwnGoal"`
	IsOvertime     bool    `json:"isOvertime"`
	Comment        *string `json:"comment"`
}

type ApiTeam struct {
	TeamName string `json:"teamName"`
}

type ApiResult struct {
	ResultName  string `json:"resultName"`
	PointsTeamA int    `json:"pointsTeam1"`
	PointsTeamB int    `json:"pointsTeam2"`
}

func ConvertApiGoalToGoal(apiGoal ApiGoal) models.Goal {
	return models.Goal{
		ScoreTeamA:     apiGoal.ScoreTeamA,
		ScoreTeamB:     apiGoal.ScoreTeamB,
		MatchMinute:    apiGoal.MatchMinute,
		GoalGetterID:   apiGoal.GoalGetterID,
		GoalGetterName: strings.TrimSpace(apiGoal.GoalGetterName),
		IsPenalty:      apiGoal.IsPenalty,
		IsOwnGoal:      apiGoal.IsOwnGoal,
		IsOvertime:     apiGoal.IsOvertime,
		Comment:        apiGoal.Comment,
	}
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
	today := time.Now().Local()
	eventPhase := models.DetermineEventPhase(today)
	if eventPhase.ApiUrl == "" {
		fmt.Printf("No API URL for event phase %s\n", eventPhase.Title)
		return
	}
	fmt.Printf("Fetching data for event phase %s\n", eventPhase.Title)
	url := eventPhase.ApiUrl

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
	goalModel := &models.GoalModel{DB: db}

	var recomputeUserScores = false

	for _, apiMatch := range matches {
		// Parse match date and time
		matchTime, err := time.Parse("2006-01-02T15:04:05", apiMatch.MatchDateTime)
		if err != nil {
			log.Fatal(err)
		}

		// Output extracted information
		dayString := matchTime.Format("2006-01-02")
		// timeString := matchTime.Format("15:04")
		fmt.Printf("Day of the match: %s\n", dayString)
		// fmt.Printf("Time of the match: %s\n", timeString)
		fmt.Printf("Name of team 1: %s\n", apiMatch.TeamA.TeamName)
		fmt.Printf("Name of team 2: %s\n", apiMatch.TeamB.TeamName)

		// Call the GetByMetadata function
		dbMatch, err := matchModel.GetByMetadata(dayString, apiMatch.TeamA.TeamName, apiMatch.TeamB.TeamName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Check if a match was found
		if dbMatch.ID == 0 {
			fmt.Printf("No match in database found, skipping (%s, %s vs. %s)\n", dayString, apiMatch.TeamA.TeamName, apiMatch.TeamB.TeamName)
			fmt.Printf("  -> YOU SHOULD ADD THIS MATCH MANUALLY!\n\n")
			continue
		}
		fmt.Printf("Match found in database: %d\n", dbMatch.ID)

		// remove all goals for this match
		// this will fix the issue that a previous goal has been revoked (e.g. due to VAR)
		// this has happened during Euro 2024 several times and I had to fix it manually in the DB
		// (this is a bit of a hack, but it works)
		fmt.Printf("Removing all goals for this match from db...\n")
		goalModel.DeleteAllForMatch(dbMatch.ID)

		// insert (or update goals)
		for _, apiGoal := range apiMatch.Goals {
			goal := ConvertApiGoalToGoal(apiGoal)
			goalId, err := goalModel.InsertOrUpdate(dbMatch.ID, goal)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			var dbOp = "added"
			if goalId == 0 {
				dbOp = "updated"
			}
			fmt.Printf("Goal %s (id %d): %d:%d (minute %d by %s)\n", dbOp, goalId, goal.ScoreTeamA, goal.ScoreTeamB, goal.MatchMinute, goal.GoalGetterName)
		}

		// read end result from api response api (also while game is still running, to get current score)
		results := make(map[string]map[string]int)
		RESULT_END := "Endergebnis"
		RESULT_AET := "nach Verlängerung"
		RESULT_APEN := "nach Elfmeterschießen"

		for _, result := range apiMatch.MatchResults {
			if result.ResultName == RESULT_END || result.ResultName == RESULT_AET || result.ResultName == RESULT_APEN {
				results[result.ResultName] = map[string]int{
					"teamA": result.PointsTeamA,
					"teamB": result.PointsTeamB,
				}
			}
		}

		// print results
		fmt.Printf("Results from API response (raw):\n")
		for key, value := range results {
			fmt.Printf("  %s: %d:%d\n", key, value["teamA"], value["teamB"])
		}

		// fix some issues with the API data
		// sometimes "nach Elfmeterschießen" is reported but it doesn't differ from the end result, so we can ignore it
		// note: for "nach Verlängerung" it's a valid case that it could be the same as the end result, so we don't ignore it
		if _, ok := results[RESULT_APEN]; ok {
			if results[RESULT_APEN]["teamA"] == results[RESULT_END]["teamA"] && results[RESULT_APEN]["teamB"] == results[RESULT_END]["teamB"] {
				fmt.Printf("Ignoring result after penalty shootout, because it's the same as the end result\n")
				delete(results, RESULT_APEN)
			}
		}

		// print results
		fmt.Printf("Results from API response (cleaned):\n")
		for key, value := range results {
			fmt.Printf("  %s: %d:%d\n", key, value["teamA"], value["teamB"])
		}

		// set end result in db
		if _, ok := results[RESULT_END]; ok {
			endScoreTeamA := results[RESULT_END]["teamA"]
			endScoreTeamB := results[RESULT_END]["teamB"]
			if dbMatch.ResultA == nil || dbMatch.ResultB == nil || *dbMatch.ResultA != endScoreTeamA || *dbMatch.ResultB != endScoreTeamB {
				fmt.Printf("-> Update result to %d:%d\n", endScoreTeamA, endScoreTeamB)
				matchModel.SetResults(dbMatch.ID, endScoreTeamA, endScoreTeamB)
			} else {
				fmt.Printf("Existing result won't be updated, score is %d:%d\n", *dbMatch.ResultA, *dbMatch.ResultB)
			}
		}

		// set result after extension in db
		if _, ok := results[RESULT_AET]; ok {
			aetScoreTeamA := results[RESULT_AET]["teamA"]
			aetScoreTeamB := results[RESULT_AET]["teamB"]
			if dbMatch.ResultAETA == nil || dbMatch.ResultAETB == nil || *dbMatch.ResultAETA != aetScoreTeamA || *dbMatch.ResultAETB != aetScoreTeamB {
				fmt.Printf("-> Update result after extension to %d:%d\n", aetScoreTeamA, aetScoreTeamB)
				matchModel.SetResultsAfterExtension(dbMatch.ID, aetScoreTeamA, aetScoreTeamB)
			} else {
				fmt.Printf("Existing result after extension won't be updated, score is %d:%d\n", *dbMatch.ResultAETA, *dbMatch.ResultAETB)
			}

		}

		// set result after penalty shootout in db
		if _, ok := results[RESULT_APEN]; ok {
			apenScoreTeamA := results[RESULT_APEN]["teamA"]
			apenScoreTeamB := results[RESULT_APEN]["teamB"]
			if dbMatch.ResultAPenA == nil || dbMatch.ResultAPenB == nil || *dbMatch.ResultAPenA != apenScoreTeamA || *dbMatch.ResultAPenB != apenScoreTeamB {
				fmt.Printf("-> Update result after penalty shootout to %d:%d\n", apenScoreTeamA, apenScoreTeamB)
				matchModel.SetResultsAfterPenalty(dbMatch.ID, apenScoreTeamA, apenScoreTeamB)
			} else {
				fmt.Printf("Existing result after penalty shootout won't be updated, score is %d:%d\n", *dbMatch.ResultAPenA, *dbMatch.ResultAPenB)
			}
		}

		// set match is finished in db
		if dbMatch.Finished != apiMatch.MatchIsFinished {
			fmt.Printf("-> Update match to finished = %t\n", apiMatch.MatchIsFinished)
			matchModel.SetMatchIsFinished(dbMatch.ID, apiMatch.MatchIsFinished)
			recomputeUserScores = true
		}

		// if !endResultFound {
		// 	fmt.Printf("Skipping match (%s vs %s) without reported end result...\n\n", apiMatch.TeamA.TeamName, apiMatch.TeamB.TeamName)
		// 	continue
		// }

		fmt.Printf("Match finished: %t\n", apiMatch.MatchIsFinished)
		if _, ok := results[RESULT_END]; ok {
			fmt.Printf("End score of team 1: %d\n", results[RESULT_END]["teamA"])
			fmt.Printf("End score of team 2: %d\n", results[RESULT_END]["teamB"])
		}

		// if dbMatch.ResultA == nil || dbMatch.ResultB == nil || *dbMatch.ResultA != endScoreTeamA || *dbMatch.ResultB != endScoreTeamB || dbMatch.Finished != apiMatch.MatchIsFinished {
		// 	fmt.Printf("-> Update result to %d:%d (finished: %t)\n", endScoreTeamA, endScoreTeamB, apiMatch.MatchIsFinished)
		// 	matchModel.SetResults(dbMatch.ID, endScoreTeamA, endScoreTeamB, apiMatch.MatchIsFinished)
		// 	recomputeUserScores = true
		// } else {
		// 	fmt.Printf("Existing result won't be updated, score is %d:%d (finished: %t)\n", *dbMatch.ResultA, *dbMatch.ResultB, dbMatch.Finished)
		// }

		fmt.Printf("\n")
	}

	if recomputeUserScores {
		fmt.Printf("Trigger points update for all user tipps...\n")
		rowsAffected, err := tippModel.UpdatePoints()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Done, updated %d db entries\n", rowsAffected)
	} else {
		fmt.Printf("No database updated occured of final scores, no user points were affected\n")
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
