package models

import (
	"database/sql"
	"errors"
	"fmt"
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

// check if user should still be allowed to submit a tipp for this match
func (m *MatchModel) AcceptsTipps(matchId int) (bool, error) {
	match, err := m.Get(matchId)
	if err != nil {
		return false, err
	}

	// match has already begun
	now := time.Now()

	fmt.Printf("now = %s, matchStart = %s\n", now.String(), match.Start.String())
	if match.Start.Before(now) {
		return false, nil
	} else {
		return true, nil
	}
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

	// Set the timezone of the start time to local
	match.Start, err = forceLocalTimezone(match.Start)
	if err != nil {
		return Match{}, err
	}

	return match, nil
}

// GetByMetadata retrieves a match based on the provided metadata
func (m *MatchModel) GetByMetadata(day string, teamA string, teamB string) (Match, error) {
	// Parse the date
	startDate, err := time.Parse("2006-01-02", day)
	if err != nil {
		return Match{}, fmt.Errorf("invalid date format: %v", err)
	}

	// Prepare the SQL query
	query := `
		SELECT id, start, team_a, team_b, result_a, result_b, match_type
		FROM matches
		WHERE DATE(start) = ? AND team_a = ? AND team_b = ?
	`

	// Execute the query
	row := m.DB.QueryRow(query, startDate, teamA, teamB)

	// Create a variable to hold the result
	var match Match

	// Scan the result into the match variable
	err = row.Scan(&match.ID, &match.Start, &match.TeamA, &match.TeamB, &match.ResultA, &match.ResultB, &match.MatchType)
	if err != nil {
		if err == sql.ErrNoRows {
			// No matching entry found
			return Match{}, nil
		}
		// An error occurred while querying
		return Match{}, err
	}

	// Return the match
	return match, nil
}

func (m *MatchModel) SetResults(id int, resultA int, resultB int) error {
	// Prepare the SQL query
	query := `
		UPDATE matches
		SET result_a = ?, result_b = ?
		WHERE id = ?
	`

	// Execute the query
	result, err := m.DB.Exec(query, resultA, resultB, id)
	if err != nil {
		return fmt.Errorf("could not execute update query: %v", err)
	}

	// Check if the update was successful
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not retrieve affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no match found with id %d", id)
	}

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
		// Set the timezone of the start time to local
		match.Start, err = forceLocalTimezone(match.Start)
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

func forceLocalTimezone(t time.Time) (time.Time, error) {
	// Load the local timezone
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return time.Time{}, err
	}

	// Set the timezone of the time to local
	localTime := time.Date(
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc,
	)
	return localTime, nil
}
