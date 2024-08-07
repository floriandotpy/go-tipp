package models

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"tipp.casualcoding.com/internal/scoring"
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

type ScoreboardData struct {
	Matches []int        `json:"matches"`
	Users   []UserPoints `json:"users"`
}

type UserPoints struct {
	Name        string `json:"name"`
	Points      []int  `json:"points"`
	TotalPoints []int  `json:"total_points"`
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
	// First update result_correct, goal_difference_correct, and tendency_correct
	stmt1 := `
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
		END
	WHERE m.result_a IS NOT NULL AND m.result_b IS NOT NULL;
	`

	// Execute the first statement
	result1, err := m.DB.Exec(stmt1)
	if err != nil {
		return 0, err
	}

	// Check the number of rows affected by the first statement
	rowsAffected1, err := result1.RowsAffected()
	if err != nil {
		return 0, err
	}

	// Second update to set the points based on the updated result_correct, goal_difference_correct, and tendency_correct
	stmt2 := buildUpdatePointsQuery(scoring.PhasePointsMap)

	// Execute the second statement
	result2, err := m.DB.Exec(stmt2)
	if err != nil {
		return 0, err
	}

	// Check the number of rows affected by the second statement
	rowsAffected2, err := result2.RowsAffected()
	if err != nil {
		return 0, err
	}

	// Return the max number of rows affected by both statements
	rowsAffected := int(math.Max(float64(rowsAffected1), float64(rowsAffected2)))
	return rowsAffected, nil
}

func buildUpdatePointsQuery(phasePointsMap map[string]scoring.PhasePoints) string {
	groupPoints := phasePointsMap[scoring.PhaseGroup]
	koPoints := phasePointsMap[scoring.PhaseKO]

	queryTemplate := `
    UPDATE tipps t
    JOIN matches m ON t.match_id = m.id
    SET t.points = CASE
        WHEN m.event_phase IN (1, 2, 3) THEN
            CASE
                WHEN t.result_correct = 1 THEN %d
                WHEN t.tendency_correct = 1 AND t.goal_difference_correct = 1 THEN %d
                WHEN t.tendency_correct = 1 THEN %d
                ELSE 0
            END
        WHEN m.event_phase IN (4, 5, 6, 7) THEN
            CASE
                WHEN t.result_correct = 1 THEN %d
                WHEN t.tendency_correct = 1 AND t.goal_difference_correct = 1 THEN %d
                WHEN t.tendency_correct = 1 THEN %d
                ELSE 0
            END
        ELSE 0
    END;
    `

	query := fmt.Sprintf(queryTemplate,
		groupPoints.CorrectResult,
		groupPoints.CorrectTendencyAndDiff,
		groupPoints.CorrectTendency,
		koPoints.CorrectResult,
		koPoints.CorrectTendencyAndDiff,
		koPoints.CorrectTendency,
	)

	return strings.TrimSpace(query)
}

func (m *TippModel) ComputeLiveTipps(tipps []Tipp, resultA int, resultB int, event_phase_type string) ([]Tipp, error) {

	points, ok := scoring.PhasePointsMap[event_phase_type]
	if !ok {
		return nil, errors.New("Unknown event phase type")
	}

	var liveTipps []Tipp
	for _, tipp := range tipps {
		liveTipp := tipp
		liveTipp.ResultCorrect = tipp.TippA == resultA && tipp.TippB == resultB
		liveTipp.GoalDifferenceCorrect = (resultA - resultB) == tipp.TippA-tipp.TippB
		liveTipp.TendencyCorrect = ((tipp.TippA > tipp.TippB && resultA > resultB) ||
			(tipp.TippA == tipp.TippB && resultA == resultB) ||
			(tipp.TippA < tipp.TippB && resultA < resultB))

		if liveTipp.ResultCorrect {
			liveTipp.Points = points.CorrectResult
		} else if liveTipp.TendencyCorrect && liveTipp.GoalDifferenceCorrect {
			liveTipp.Points = points.CorrectTendencyAndDiff
		} else if liveTipp.TendencyCorrect {
			liveTipp.Points = points.CorrectTendency
		}

		liveTipps = append(liveTipps, liveTipp)
	}

	// sort tipps by points descending
	sort.Slice(liveTipps, func(i, j int) bool {
		return liveTipps[i].Points > liveTipps[j].Points
	})

	return liveTipps, nil
}

func (m *TippModel) GetScoreboardData(groupIds []int) (ScoreboardData, error) {
	// Convert groupIds to a comma-separated string for the SQL query
	groupIdsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(groupIds)), ","), "[]")

	// Perform SQL query to aggregate user points
	query := `
	WITH all_matches AS (
		SELECT DISTINCT id AS match_id FROM matches WHERE result_a IS NOT NULL
	),
	user_matches AS (
		SELECT
			u.id AS user_id,
			u.name,
			m.match_id
		FROM
			users u
		CROSS JOIN
			all_matches m
		WHERE
			EXISTS (
				SELECT 1 
				FROM user_groups ug 
				WHERE ug.user_id = u.id AND ug.group_id IN (` + groupIdsStr + `)
			)
	),
	user_tipps AS (
		SELECT 
			um.user_id,
			um.match_id,
			COALESCE(SUM(t.points), 0) AS points
		FROM 
			user_matches um
		LEFT JOIN 
			tipps t ON um.user_id = t.user_id AND um.match_id = t.match_id
		GROUP BY 
			um.user_id, um.match_id
	)
	SELECT 
		ut.user_id,
		u.name,
		ut.match_id,
		ut.points,
		SUM(ut.points) OVER (PARTITION BY ut.user_id ORDER BY ut.match_id) AS total_points
	FROM 
		user_tipps ut
	JOIN 
		users u ON ut.user_id = u.id
	ORDER BY 
		ut.user_id, ut.match_id;
    `

	rows, err := m.DB.Query(query)
	if err != nil {
		return ScoreboardData{}, err
	}
	defer rows.Close()

	userPointsMap := make(map[string][]int)
	totalPointsMap := make(map[string][]int)
	var matchesSet = make(map[int]struct{})

	for rows.Next() {
		var userId int
		var name string
		var matchId int
		var points int
		var totalPoints int

		err := rows.Scan(&userId, &name, &matchId, &points, &totalPoints)
		if err != nil {
			return ScoreboardData{}, nil
		}

		// Collect matches
		matchesSet[matchId] = struct{}{}

		// Collect user points
		if _, exists := userPointsMap[name]; !exists {
			userPointsMap[name] = make([]int, 0)
		}
		userPointsMap[name] = append(userPointsMap[name], points)

		// Collect total points
		if _, exists := totalPointsMap[name]; !exists {
			totalPointsMap[name] = make([]int, 0)
		}
		totalPointsMap[name] = append(totalPointsMap[name], totalPoints)
	}

	// Convert matches set to a sorted slice
	matchNumbers := make([]int, 0, len(matchesSet))
	matches := make([]int, 0, len(matchesSet))
	var matchNumber = 1
	for matchId := range matchesSet {
		matches = append(matches, matchId)
		matchNumbers = append(matchNumbers, matchNumber)
		matchNumber += 1
	}

	// Convert map to slice of UserPoints
	users := make([]UserPoints, 0, len(userPointsMap))
	for name, points := range userPointsMap {
		users = append(users, UserPoints{Name: name, Points: points, TotalPoints: totalPointsMap[name]})
	}

	data := ScoreboardData{
		Matches: matchNumbers,
		Users:   users,
	}

	return data, nil
}
