package models

import (
	"database/sql"
	"math"
)

type Goal struct {
	ID             int
	MatchId        int
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

func (m *GoalModel) AllForMatch(matchId int) ([]Goal, error) {
	stmt := `SELECT
	id,
    match_id,
    score_team_a,
    score_team_b,
    match_minute,
    goal_getter_id,
    goal_getter_name,
    is_penalty,
    is_own_goal,
    is_overtime,
    comment TEXT
	FROM goals WHERE match_id = ?
	ORDER BY (IFNULL(score_team_a,0)+IFNULL(score_team_b,0)) ASC
	`
	rows, err := m.DB.Query(stmt, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []Goal
	for rows.Next() {
		var g Goal
		err = rows.Scan(&g.ID, &g.MatchId, &g.ScoreTeamA, &g.ScoreTeamB,
			&g.MatchMinute, &g.GoalGetterID, &g.GoalGetterName, &g.IsPenalty, &g.IsOwnGoal, &g.IsOvertime, &g.Comment)

		if err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return goals, nil
}

func (m *GoalModel) LiveScore(goals []Goal) (int, int) {
	var resultA = 0
	var resultB = 0
	for _, goal := range goals {
		resultA = int(math.Max(float64(resultA), float64(goal.ScoreTeamA)))
		resultB = int(math.Max(float64(resultB), float64(goal.ScoreTeamB)))
	}
	return resultA, resultB
}
