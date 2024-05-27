package models

import (
	"database/sql"
	"errors"
	"time"
)

type Tipp struct {
	ID      int
	MatchId int
	UserId  int
	TippA   int
	TippB   int
	Created time.Time
	Changed time.Time
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
	stmt := `SELECT id, match_id, user_id, tipp_a, tipp_b, created, changed FROM tipps WHERE id = ?`
	var t Tipp
	err := m.DB.QueryRow(stmt, id).Scan(&t.ID, &t.MatchId, &t.UserId, &t.TippA, &t.TippB, &t.Created, &t.Changed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tipp{}, ErrNoRecord
		} else {
			return Tipp{}, nil
		}
	}

	return t, nil
}

func (m *TippModel) AllForUser(userId int) ([]Tipp, error) {
	stmt := `SELECT id, match_id, user_id, tipp_a, tipp_b, created, changed FROM tipps WHERE user_id = ?`
	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tipps []Tipp
	for rows.Next() {
		var t Tipp
		err = rows.Scan(&t.ID, &t.MatchId, &t.UserId, &t.TippA, &t.TippB, &t.Created, &t.Changed)
		if err != nil {
			return nil, err
		}
		tipps = append(tipps, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tipps, nil
}
