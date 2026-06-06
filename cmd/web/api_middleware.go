package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"tipp.casualcoding.com/internal/models"
)

// --- Per-IP Rate Limiter ---

// ipRateLimiter tracks rate limiters per client IP.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int
	stop     chan struct{}
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    burst,
		stop:     make(chan struct{}),
	}
	// Background cleanup of stale entries every 3 minutes.
	go rl.cleanup(3 * time.Minute)
	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *ipRateLimiter) Stop() {
	close(rl.stop)
}

func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = &rateLimiterEntry{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *ipRateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, entry := range rl.limiters {
				if time.Since(entry.lastSeen) > 5*time.Minute {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// apiContextKey is a custom type for context keys to avoid collisions.
type apiContextKey string

const apiUserIDKey apiContextKey = "userID"

// apiRateLimit is middleware that enforces per-IP rate limiting.
// Returns 429 Too Many Requests when the limit is exceeded.
func (app *application) apiRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		limiter := app.apiLimiter.getLimiter(ip)
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			app.apiError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
