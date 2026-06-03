package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tipp.casualcoding.com/internal/models"
)

// apiContextKey is a custom type for context keys to avoid collisions.
type apiContextKey string

const apiUserIDKey apiContextKey = "userID"

// apiAuth validates the Bearer token from the Authorization header and stores
// the authenticated user ID in the request context. Returns 401 JSON on failure.
func (app *application) apiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, "Bearer ") {
			app.apiError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := app.apiTokens.Validate(token)
		if err != nil {
			if errors.Is(err, models.ErrInvalidCredentials) {
				app.apiError(w, http.StatusUnauthorized, "invalid API token")
				return
			}
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Verify user still exists
		_, err = app.users.Get(userID)
		if err != nil {
			if errors.Is(err, models.ErrUserNotFound) {
				app.apiError(w, http.StatusUnauthorized, "invalid API token")
				return
			}
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}

		ctx := context.WithValue(r.Context(), apiUserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// apiUserID extracts the authenticated user ID from the request context.
func apiUserID(r *http.Request) int {
	userID, ok := r.Context().Value(apiUserIDKey).(int)
	if !ok {
		return 0
	}
	return userID
}

// apiError writes a JSON error response with the given status code and message.
func (app *application) apiError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// apiValidationError writes a JSON 422 response with field-level validation errors.
func (app *application) apiValidationError(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  "validation failed",
		"fields": fields,
	})
}
