package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// FetchMatchData fetches and decodes match data from the given URL.
func FetchMatchData(url string) ([]ApiMatch, error) {
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

// ConvertApiGoalToGoal converts an API goal response to the internal Goal model.
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
