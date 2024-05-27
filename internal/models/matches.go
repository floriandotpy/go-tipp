package models

import (
	"database/sql"
	"errors"
	"time"
)

type Match struct {
	ID        int
	TeamA     string
	TeamB     string
	Start     time.Time
	MatchType string

	// goal numbers
	ResultA *int
	ResultB *int
}

type MatchModel struct {
	DB *sql.DB
}

func (m *MatchModel) Insert(teamA string, teamB string, start time.Time, matchType string) (int, error) {
	return 0, nil
}

func (m *MatchModel) Get(id int) (Match, error) {
	stmt := `SELECT id, start, team_a, team_b, result_a, result_b, match_type FROM matches WHERE id = ?`
	var match Match
	err := m.DB.QueryRow(stmt, id).Scan(&match.ID, &match.Start, &match.TeamA, &match.TeamB, &match.ResultA, &match.ResultB, &match.MatchType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Match{}, ErrNoRecord
		} else {
			return Match{}, nil
		}
	}
	return match, nil
}

func (m *MatchModel) SetResults(id int, resultA int, resultB int) error {
	return nil
}

// will return upcoming matches (i.e. which haven't been started yet)
func (m *MatchModel) Upcoming() ([]Match, error) {
	return nil, nil
}

// will return all matches
func (m *MatchModel) All() ([]Match, error) {
	stmt := `SELECT id, start, team_a, team_b, result_a, result_b, match_type FROM matches ORDER BY start ASC`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var matches []Match

	for rows.Next() {
		var match Match
		err = rows.Scan(&match.ID, &match.Start, &match.TeamA, &match.TeamB, &match.ResultA, &match.ResultB, &match.MatchType)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
