package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
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

// hashToken computes the SHA-256 hex digest of a plaintext token.
// Since API tokens have 256 bits of entropy (not user-chosen passwords),
// a fast cryptographic hash is sufficient — bcrypt is unnecessary here.
func hashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
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
	tokenHash := hashToken(plaintext)

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
		userID, tokenHash,
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

// Validate looks up the token by its SHA-256 hash (O(1) indexed lookup).
// Returns the user ID if valid, or ErrInvalidCredentials if not found.
func (m *ApiTokenModel) Validate(plaintext string) (int, error) {
	if len(plaintext) < len(apiTokenPrefix) || plaintext[:len(apiTokenPrefix)] != apiTokenPrefix {
		return 0, ErrInvalidCredentials
	}

	tokenHash := hashToken(plaintext)

	var userID int
	err := m.DB.QueryRow(
		`SELECT user_id FROM api_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrInvalidCredentials
		}
		return 0, err
	}

	return userID, nil
}
