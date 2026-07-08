package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tipp.casualcoding.com/internal/models"
)

type ApiGroup struct {
	GroupName    string `json:"groupName"`
	GroupOrderID int    `json:"groupOrderID"`
	GroupID      int    `json:"groupID"`
}

type ApiMatch struct {
	MatchID          int         `json:"matchID"`
	MatchDateTime    string      `json:"matchDateTime"`
	MatchDateTimeUTC string      `json:"matchDateTimeUTC"`
	TeamA            ApiTeam     `json:"team1"`
	TeamB            ApiTeam     `json:"team2"`
	MatchResults     []ApiResult `json:"matchResults"`
	MatchIsFinished  bool        `json:"matchIsFinished"`
	Goals            []ApiGoal   `json:"goals"`
	Group            ApiGroup    `json:"group"`
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
	ResultName   string `json:"resultName"`
	PointsTeamA  int    `json:"pointsTeam1"`
	PointsTeamB  int    `json:"pointsTeam2"`
	ResultTypeID int    `json:"resultTypeID"`
}

const (
	ResultRegularTime    = "regular_time"
	ResultAfterExtraTime = "after_extra_time"
	ResultAfterPenalties = "after_penalties"
)

// ExtractMatchResults returns canonical result slots from OpenLigaDB data.
// Regular time is the score used for tip scoring. Newer datasets expose it as
// "nach 90 Minuten" / type 3; older datasets used "Endergebnis" / type 2.
func ExtractMatchResults(apiMatch ApiMatch) map[string][2]int {
	results := make(map[string][2]int)

	for _, result := range apiMatch.MatchResults {
		score := [2]int{result.PointsTeamA, result.PointsTeamB}

		switch {
		case result.ResultTypeID == 3 || result.ResultName == "nach 90 Minuten":
			results[ResultRegularTime] = score
		case result.ResultTypeID == 4 || result.ResultName == "nach Verlängerung":
			results[ResultAfterExtraTime] = score
		case result.ResultTypeID == 5 || result.ResultName == "nach Elfmeterschießen":
			results[ResultAfterPenalties] = score
		case result.ResultTypeID == 2 || result.ResultName == "Endergebnis":
			if _, ok := results[ResultRegularTime]; !ok {
				results[ResultRegularTime] = score
			}
		}
	}

	if penalty, ok := results[ResultAfterPenalties]; ok {
		if regular, ok2 := results[ResultRegularTime]; ok2 && penalty == regular {
			delete(results, ResultAfterPenalties)
		}
	}

	return results
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
