package models

import (
	"database/sql"
	"time"
)

type Match struct {
	TeamA string
	TeamB string
	Start time.Time
	Title string

	// goal numbers
	// ResultA *int
	// ResultB *int
}

type MatchModel struct {
	DB *sql.DB
}

func (m *MatchModel) Insert(teamA string, teamB string, start time.Time, title string) (int, error) {
	return 0, nil
}

func (m *MatchModel) Get(id int) (Match, error) {
	return Match{}, nil
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
	return nil, nil
}
