package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const apiTokenPrefix = "gtipp_"

type ApiToken struct {
	ID      int
	UserID  int
	Created time.Time
}

type ApiTokenModel struct {
	DB *sql.DB
}

// Generate creates a new API token for the user, revoking any existing one.
// Returns the plaintext token (shown once to the user).
func (m *ApiTokenModel) Generate(userID int) (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	plaintext := apiTokenPrefix + hex.EncodeToString(randomBytes)

	hashedToken, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return "", err
	}

	tx, err := m.DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM api_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(
		`INSERT INTO api_tokens (user_id, token_hash, created) VALUES (?, ?, UTC_TIMESTAMP())`,
		userID, string(hashedToken),
	)
	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

// Revoke deletes the API token for the given user.
// Returns ErrNoRecord if no token exists.
func (m *ApiTokenModel) Revoke(userID int) error {
	result, err := m.DB.Exec(`DELETE FROM api_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNoRecord
	}

	return nil
}

// Exists checks whether the given user has an active API token.
func (m *ApiTokenModel) Exists(userID int) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM api_tokens WHERE user_id = ?)`, userID).Scan(&exists)
	return exists, err
}

// Validate checks the plaintext token against all stored hashes.
// Returns the user ID if valid, or ErrInvalidCredentials if not.
func (m *ApiTokenModel) Validate(plaintext string) (int, error) {
	rows, err := m.DB.Query(`SELECT user_id, token_hash FROM api_tokens`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int
		var tokenHash string

		err = rows.Scan(&userID, &tokenHash)
		if err != nil {
			return 0, err
		}

		err = bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(plaintext))
		if err == nil {
			return userID, nil
		}
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	return 0, ErrInvalidCredentials
}
