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

	TendencyCorrect       bool
	GoalDifferenceCorrect bool
	ResultCorrect         bool
	Points                int

	// optional (derived properties, not set on all queries)
	UserName string
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

func (m *TippModel) Update(matchId int, userId int, tippA int, tippB int) error {
	stmt := `UPDATE tipps
	SET tipp_a = ?, tipp_b = ?, changed = UTC_TIMESTAMP()
	WHERE match_id = ? AND user_id = ?`

	_, err := m.DB.Exec(stmt, tippA, tippB, matchId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (m *TippModel) Exists(matchId int, userId int) (bool, error) {
	stmt := `SELECT COUNT(*) FROM tipps WHERE match_id = ? AND user_id = ?`

	var count int
	err := m.DB.QueryRow(stmt, matchId, userId).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *TippModel) Delete(matchId int, userId int) error {
	stmt := `DELETE FROM tipps WHERE match_id = ? AND user_id = ?`

	_, err := m.DB.Exec(stmt, matchId, userId)
	if err != nil {
		return err
	}

	return nil
}

func (m *TippModel) InsertOrUpdate(matchId int, userId int, tippA int, tippB int) error {
	tippExists, err := m.Exists(matchId, userId)
	if err != nil {
		return err
	}

	if tippExists {
		err = m.Update(matchId, userId, tippA, tippB)
		if err != nil {
			return err
		}
	} else {
		_, err = m.Insert(matchId, userId, tippA, tippB)
		if err != nil {
			return err
		}
	}
	return nil
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
	stmt := `SELECT id, match_id, user_id, tipp_a, tipp_b, created, changed, 
	result_correct, tendency_correct, goal_difference_correct, points FROM tipps WHERE user_id = ?`
	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tipps []Tipp
	for rows.Next() {
		var t Tipp
		err = rows.Scan(&t.ID, &t.MatchId, &t.UserId, &t.TippA, &t.TippB, &t.Created, &t.Changed,
			&t.ResultCorrect, &t.TendencyCorrect, &t.GoalDifferenceCorrect, &t.Points)
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

func (m *TippModel) AllForMatch(matchId int) ([]Tipp, error) {
	stmt := `SELECT t.id, t.match_id, t.user_id, t.tipp_a, t.tipp_b, t.created, t.changed, u.name as user_name, t.tendency_correct, t.goal_difference_correct, t.result_correct, t.points FROM tipps t
	JOIN  
	users u
	ON 
	t.user_id = u.id
	WHERE match_id = ? ORDER BY t.points DESC, user_name ASC`
	rows, err := m.DB.Query(stmt, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tipps []Tipp
	for rows.Next() {
		var t Tipp
		err = rows.Scan(&t.ID, &t.MatchId, &t.UserId, &t.TippA, &t.TippB, &t.Created, &t.Changed, &t.UserName, &t.TendencyCorrect, &t.GoalDifferenceCorrect, &t.ResultCorrect, &t.Points)
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

// will compute and set the points in the database for all tipps that relate to completed matches
// Returns the number of rows affected
func (m *TippModel) UpdatePoints() (int, error) {
	stmt := `
	UPDATE tipps t
	JOIN matches m ON t.match_id = m.id
	SET 
		t.result_correct = CASE 
			WHEN t.tipp_a = m.result_a AND t.tipp_b = m.result_b THEN 1 
			ELSE 0 
		END,
		t.goal_difference_correct = CASE 
			WHEN (t.tipp_a - t.tipp_b) = (m.result_a - m.result_b) THEN 1 
			ELSE 0 
		END,
		t.tendency_correct = CASE 
			WHEN (t.tipp_a > t.tipp_b AND m.result_a > m.result_b) 
				 OR (t.tipp_a = t.tipp_b AND m.result_a = m.result_b) 
				 OR (t.tipp_a < t.tipp_b AND m.result_a < m.result_b) THEN 1 
			ELSE 0 
		END,
		t.points = CASE
			WHEN t.result_correct = 1 THEN 5
			WHEN t.tendency_correct = 1 AND t.goal_difference_correct = 1 THEN 3
			WHEN t.tendency_correct = 1 THEN 1
			ELSE 0
		END
	WHERE m.result_a IS NOT NULL AND m.result_b IS NOT NULL;
	`

	result, err := m.DB.Exec(stmt)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil

}
