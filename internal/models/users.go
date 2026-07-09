package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
	Points         int
	Tipps          int
	IsAdmin        bool
	Place          int
	IsActive       bool
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created) VALUES(?, ?, ?, UTC_TIMESTAMP())`

	result, err := m.DB.Exec(stmt, name, email, string(hashedPassword))
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return 0, ErrDuplicateEmail
			}
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_name") {
				return 0, ErrDuplicateName
			}
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *UserModel) Get(id int) (User, error) {

	stmt := `SELECT id, name, email, created, admin FROM users WHERE id = ?`
	var user User
	err := m.DB.QueryRow(stmt, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Created,
		&user.IsAdmin,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		} else {
			return User{}, err
		}
	}
	return user, nil
}

func (m *UserModel) GetByUsername(username string) (User, error) {
	stmt := `SELECT id, name, email, created, admin FROM users WHERE name = ?`
	var user User
	err := m.DB.QueryRow(stmt, username).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Created,
		&user.IsAdmin,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		} else {
			return User{}, err
		}
	}
	return user, nil
}

func (m *UserModel) Authenticate(email, password string) (int, bool, error) {

	var id int
	var hashedPassword []byte
	var isAdmin bool

	stmt := "SELECT id, hashed_password, admin FROM users WHERE email = ?"
	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword, &isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, ErrInvalidCredentials
		} else {
			return 0, false, err
		}
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, false, ErrInvalidCredentials
		} else {
			return 0, false, err
		}
	}

	return id, isAdmin, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	return false, nil
}

// All returns all users ordered by name.
func (m *UserModel) All() ([]User, error) {
	stmt := `SELECT id, name, email, created, admin FROM users ORDER BY name ASC`
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.Created, &u.IsAdmin)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdatePassword sets a new hashed password for the given user.
func (m *UserModel) UpdatePassword(id int, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	stmt := `UPDATE users SET hashed_password = ? WHERE id = ?`
	_, err = m.DB.Exec(stmt, string(hashedPassword), id)
	return err
}

func (m *UserModel) GroupLeaderboard(groupId, eventID int) ([]User, error) {
	stmt := `SELECT 
		u.id AS user_id, 
		u.name AS user_name, 
		COALESCE(SUM(CASE WHEN ma.finished = 1 THEN t.points ELSE 0 END), 0) AS total_points,
		COUNT(t.id) AS tipps_count,
		(SELECT COUNT(*) FROM tipps t2
			JOIN (
				SELECT id FROM matches
				WHERE event_id = ? AND finished = 1
				ORDER BY start DESC LIMIT 10
			) recent ON t2.match_id = recent.id
			WHERE t2.user_id = u.id
		) AS recent_tipps
	FROM users u
	JOIN user_groups ug ON u.id = ug.user_id
	LEFT JOIN tipps t ON u.id = t.user_id
	LEFT JOIN matches ma ON t.match_id = ma.id
	WHERE ug.group_id = ? AND (t.id IS NULL OR ma.event_id = ?)
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	rows, err := m.DB.Query(stmt, eventID, groupId, eventID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User
		var recentTipps int
		err = rows.Scan(&user.ID, &user.Name, &user.Points, &user.Tipps, &recentTipps)
		if err != nil {
			return nil, err
		}
		user.IsActive = recentTipps >= 1

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	sortedUsers := setPlaceValues(users)

	return sortedUsers, nil
}

func (m *UserModel) GlobalLeaderboard(eventID int) ([]User, error) {
	stmt := `SELECT 
		u.id AS user_id, 
		u.name AS user_name, 
		COALESCE(SUM(CASE WHEN ma.finished = 1 THEN t.points ELSE 0 END), 0) AS total_points, 
		COUNT(t.id) AS tipps_count,
		(SELECT COUNT(*) FROM tipps t2
			JOIN (
				SELECT id FROM matches
				WHERE event_id = ? AND finished = 1
				ORDER BY start DESC LIMIT 10
			) recent ON t2.match_id = recent.id
			WHERE t2.user_id = u.id
		) AS recent_tipps
	FROM users u
	LEFT JOIN tipps t ON u.id = t.user_id
	LEFT JOIN matches ma ON t.match_id = ma.id
	WHERE t.id IS NULL OR ma.event_id = ?
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	rows, err := m.DB.Query(stmt, eventID, eventID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User
		var recentTipps int
		err = rows.Scan(&user.ID, &user.Name, &user.Points, &user.Tipps, &recentTipps)
		if err != nil {
			return nil, err
		}
		user.IsActive = recentTipps >= 1

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	users = setPlaceValues(users)

	return users, nil
}

func (m *UserModel) GetBestInSelectedPhases(groupId, eventID int, phaseIds []int) ([]User, error) {

	stmt := `SELECT
		u.id AS user_id,
		u.name AS user_name,
		COALESCE(SUM(CASE WHEN m.finished = 1 THEN t.points ELSE 0 END), 0) AS total_points,
		COUNT(t.id) AS tipps_count
	FROM users u
	JOIN user_groups ug ON u.id = ug.user_id
	LEFT JOIN tipps t ON u.id = t.user_id
	JOIN matches m ON t.match_id = m.id
	WHERE ug.group_id = ? AND m.event_id = ? AND m.event_phase IN (` + strings.Repeat("?,", len(phaseIds)-1) + `?)
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	// Create a slice of interface{} to hold all parameters
	params := make([]interface{}, 0, len(phaseIds)+2)
	params = append(params, groupId)
	params = append(params, eventID)
	for _, phase := range phaseIds {
		params = append(params, phase)
	}

	rows, err := m.DB.Query(stmt, params...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User
		err = rows.Scan(&user.ID, &user.Name, &user.Points, &user.Tipps)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	users = setPlaceValues(users)

	return users, nil
}

func setPlaceValues(users []User) []User {
	var place = 0
	var i = 0
	var prevScore = 9999999

	var newUsers []User

	for _, user := range users {
		i += 1
		var newUser = user

		if user.Points < prevScore {
			prevScore = user.Points
			place = i
		}
		newUser.Place = place
		newUsers = append(newUsers, newUser)
	}

	return newUsers
}
