package models

import (
	"database/sql"
)

type Goal struct {
	ScoreTeamA     int
	ScoreTeamB     int
	MatchMinute    int
	GoalGetterID   int
	GoalGetterName string
	IsPenalty      bool
	IsOwnGoal      bool
	IsOvertime     bool
	Comment        *string
}

type GoalModel struct {
	DB *sql.DB
}

func (m *GoalModel) InsertOrUpdate(matchId int, goal Goal) (int, error) {

	// Insert goal into the database, avoiding duplicates
	result, err := m.DB.Exec(`
INSERT INTO goals (
	match_id, score_team_a, score_team_b, match_minute,
	goal_getter_id, goal_getter_name, is_penalty, is_own_goal,
	is_overtime, comment
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	match_minute = VALUES(match_minute),
	goal_getter_id = VALUES(goal_getter_id),
	goal_getter_name = VALUES(goal_getter_name),
	is_penalty = VALUES(is_penalty),
	is_own_goal = VALUES(is_own_goal),
	is_overtime = VALUES(is_overtime),
	comment = VALUES(comment)`,
		matchId, goal.ScoreTeamA, goal.ScoreTeamB,
		goal.MatchMinute, goal.GoalGetterID, goal.GoalGetterName,
		goal.IsPenalty, goal.IsOwnGoal, goal.IsOvertime, goal.Comment)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil

}