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

	stmt := `
        SELECT u.id, u.name, u.email, u.created, u.admin,
               COALESCE(SUM(t.points), 0) AS total_points,
               COUNT(t.id) AS total_tipps
        FROM users u
        LEFT JOIN tipps t ON u.id = t.user_id
        WHERE u.id = ?
        GROUP BY u.id
    `
	var user User
	err := m.DB.QueryRow(stmt, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Created,
		&user.IsAdmin,
		&user.Points,
		&user.Tipps,
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
	stmt := `
        SELECT u.id, u.name, u.email, u.created, u.admin,
               COALESCE(SUM(t.points), 0) AS total_points,
               COUNT(t.id) AS total_tipps
        FROM users u
        LEFT JOIN tipps t ON u.id = t.user_id
        WHERE u.name = ?
        GROUP BY u.id
    `
	var user User
	err := m.DB.QueryRow(stmt, username).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Created,
		&user.IsAdmin,
		&user.Points,
		&user.Tipps,
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

func (m *UserModel) GroupLeaderboard(groupId int) ([]User, error) {
	stmt := `SELECT 
		u.id AS user_id, 
		u.name AS user_name, 
		COALESCE(SUM(t.points), 0) AS total_points,
		COUNT(t.id) AS tipps_count
	FROM users u
	JOIN user_groups ug ON u.id = ug.user_id
	LEFT JOIN tipps t ON u.id = t.user_id
	WHERE ug.group_id = ?
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	rows, err := m.DB.Query(stmt, groupId)
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

	sortedUsers := setPlaceValues(users)

	return sortedUsers, nil
}

func (m *UserModel) GlobalLeaderboard() ([]User, error) {
	stmt := `SELECT 
		u.id AS user_id, 
		u.name AS user_name, 
		COALESCE(SUM(t.points), 0) AS total_points, 
		COUNT(t.id) AS tipps_count
	FROM users u
	LEFT JOIN tipps t ON u.id = t.user_id
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	rows, err := m.DB.Query(stmt)
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

func (m *UserModel) GetBestInSelectedPhases(groupId int, phaseIds []int) ([]User, error) {

	stmt := `SELECT
		u.id AS user_id,
		u.name AS user_name,
		COALESCE(SUM(t.points), 0) AS total_points,
		COUNT(t.id) AS tipps_count
	FROM users u
	JOIN user_groups ug ON u.id = ug.user_id
	LEFT JOIN tipps t ON u.id = t.user_id
	JOIN matches m ON t.match_id = m.id
	WHERE ug.group_id = ? AND m.event_phase IN (` + strings.Repeat("?,", len(phaseIds)-1) + `?)
	GROUP BY u.id, u.name
	ORDER BY total_points DESC, tipps_count DESC, user_id ASC;`

	// Create a slice of interface{} to hold all parameters
	params := make([]interface{}, 0, len(phaseIds)+1)
	params = append(params, groupId)
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
