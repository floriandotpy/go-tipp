package models

import (
	"database/sql"
	"time"
)

type Tipp struct {
	id      int
	matchId int
	userId  int
	tippA   int
	tippB   int
	created time.Time
	changed time.Time
}

type TippModel struct {
	DB *sql.DB
}

func (m *TippModel) Insert(matchId int, userId int, tippA int, tippB int) (int, error) {

	stmt := `INSERT INTO tipps (match_id, user_id, tipp_a, tipp_b, created, changed)
	VALUES(?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP())`

	result, err := m.DB.Exec(stmt, matchId, userId, tippA, tippB)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *TippModel) Get(id int) (Tipp, error) {
	return Tipp{}, nil
}

func (m *TippModel) AllForUser(userId int) ([]Tipp, error) {
	return nil, nil
}
