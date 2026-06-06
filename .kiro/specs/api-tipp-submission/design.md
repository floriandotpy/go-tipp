# Technical Design: API Tipp Submission

## Overview

This design adds a REST API layer to the existing go-tipp web application, enabling programmatic read/write access to matches and tipps via per-user Bearer tokens. The API lives under `/api/v1/` on the same HTTP server, sharing the database and model layer but using a separate middleware chain that skips CSRF and session handling.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Server                             │
├───────────────────────────┬─────────────────────────────────┤
│   Web Routes (existing)   │      API Routes (new)           │
│   session + CSRF + HTML   │   Bearer token + JSON           │
├───────────────────────────┼─────────────────────────────────┤
│   dynamic middleware      │   apiAuth middleware            │
│   (LoadAndSave, noSurf,   │   (validateToken, resolve       │
│    resolveEvent)           │    user, resolve event)         │
├───────────────────────────┴─────────────────────────────────┤
│              Shared Model Layer (internal/models)            │
├─────────────────────────────────────────────────────────────┤
│                    MySQL Database                            │
└─────────────────────────────────────────────────────────────┘
```

The API is served by the same Go HTTP server on the same port. API requests are distinguished by their `/api/v1/` path prefix and use a dedicated middleware chain (`apiAuth`) that performs Bearer token validation instead of session-based auth. Both the web and API layers share the same model layer and database connection pool.

## Components and Interfaces

### New Files

| File | Purpose |
|------|---------|
| `db/migrations/20260603120000_add_api_tokens.sql` | Migration for api_tokens table |
| `internal/models/api_tokens.go` | ApiTokenModel with Generate/Revoke/Validate/Exists |
| `cmd/web/api_middleware.go` | Bearer token auth middleware + JSON error helpers |
| `cmd/web/api_handlers.go` | API endpoint handlers (matches, tipps) |
| `ui/html/pages/settings.html` | Token management UI page |

### Modified Files

| File | Change |
|------|--------|
| `cmd/web/main.go` | Add `apiTokens *models.ApiTokenModel` to application struct |
| `cmd/web/routes.go` | Register API routes and settings web routes |
| `cmd/web/handlers.go` | Add token management web handlers (settings, generate, revoke) |
| `cmd/web/templates.go` | Add `TokenExists` and `NewToken` to templateData |
| `ui/html/partials/nav.html` | Add "Einstellungen" link for authenticated users |
| `db/schema.sql` | Add api_tokens table definition |

### API Endpoint Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/matches` | Bearer | List all matches for the active event |
| GET | `/api/v1/tipps` | Bearer | List authenticated user's tipps for active event |
| POST | `/api/v1/tipps` | Bearer | Create or update a tipp for a match |

### Web Route Additions

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/user/settings` | Session | Show token management page |
| POST | `/user/settings/token/generate` | Session | Generate a new API token |
| POST | `/user/settings/token/revoke` | Session | Revoke existing API token |

### Component: ApiTokenModel (`internal/models/api_tokens.go`)

```go
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
func (m *ApiTokenModel) Generate(userID int) (string, error)

// Revoke deletes the API token for the given user.
// Returns ErrNoRecord if no token exists.
func (m *ApiTokenModel) Revoke(userID int) error

// Exists checks whether the given user has an active API token.
func (m *ApiTokenModel) Exists(userID int) (bool, error)

// Validate checks the plaintext token against all stored hashes.
// Returns the user ID if valid, or ErrInvalidCredentials if not.
func (m *ApiTokenModel) Validate(plaintext string) (int, error)
```

### Component: API Middleware (`cmd/web/api_middleware.go`)

```go
package main

// apiAuth validates the Bearer token and stores the user ID in context.
func (app *application) apiAuth(next http.Handler) http.Handler

// apiUserID extracts the authenticated user ID from the request context.
func apiUserID(r *http.Request) int

// apiError writes a JSON error response with the given status and message.
func (app *application) apiError(w http.ResponseWriter, status int, message string)

// apiValidationError writes a JSON 422 response with field-level errors.
func (app *application) apiValidationError(w http.ResponseWriter, fields map[string]string)
```

### Component: API Handlers (`cmd/web/api_handlers.go`)

```go
package main

// apiGetMatches handles GET /api/v1/matches
// Returns all matches for the active event as JSON.
func (app *application) apiGetMatches(w http.ResponseWriter, r *http.Request)

// apiGetTipps handles GET /api/v1/tipps
// Returns all tipps for the authenticated user in the active event.
func (app *application) apiGetTipps(w http.ResponseWriter, r *http.Request)

// apiPostTipp handles POST /api/v1/tipps
// Creates or updates a tipp for the authenticated user.
func (app *application) apiPostTipp(w http.ResponseWriter, r *http.Request)
```

### Route Registration

```go
// In routes(), after existing web routes:
api := alice.New(app.apiAuth)
mux.Handle("GET /api/v1/matches", api.ThenFunc(app.apiGetMatches))
mux.Handle("GET /api/v1/tipps", api.ThenFunc(app.apiGetTipps))
mux.Handle("POST /api/v1/tipps", api.ThenFunc(app.apiPostTipp))
```

These routes are wrapped by the outer `standard` chain (recoverPanic, logRequest, commonHeaders) but bypass session/CSRF handling entirely.

## Data Models

### New Table: `api_tokens`

```sql
CREATE TABLE `api_tokens` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `token_hash` char(60) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `created` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `api_tokens_uc_user` (`user_id`),
  CONSTRAINT `api_tokens_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

Design choices:
- `UNIQUE KEY` on `user_id` enforces one token per user at the database level.
- `token_hash` stores a bcrypt hash (60 chars) — the plaintext is never persisted.
- `ON DELETE CASCADE` ensures tokens are cleaned up when a user is deleted.
- No expiry column for now (tokens are valid until revoked).

### JSON Response Shapes

**GET /api/v1/matches** response:
```json
[
  {
    "id": 1,
    "team_a": "Germany",
    "team_b": "France",
    "start": "2026-06-14T18:00:00Z",
    "match_type": "group",
    "event_phase": 1,
    "finished": false,
    "accepts_tipps": true
  }
]
```

**GET /api/v1/tipps** response:
```json
[
  {
    "match_id": 1,
    "tipp_a": 2,
    "tipp_b": 1,
    "created": "2026-06-10T09:30:00Z",
    "changed": "2026-06-10T09:30:00Z"
  }
]
```

**POST /api/v1/tipps** request body:
```json
{
  "match_id": 1,
  "tipp_a": 2,
  "tipp_b": 1
}
```

**Error response** (all errors):
```json
{
  "error": "match is locked (already started or teams unknown)"
}
```

**Validation error** (422):
```json
{
  "error": "validation failed",
  "fields": {
    "tipp_a": "tipp_a must be between 0 and 99"
  }
}
```

### templateData Additions

```go
type templateData struct {
    // ... existing fields ...
    TokenExists bool   // whether the user has an active API token
    NewToken    string // plaintext token shown once after generation
}
```

## Error Handling

| Scenario | HTTP Status | Error Message |
|----------|-------------|---------------|
| Missing/malformed Authorization header | 401 | "missing or malformed Authorization header" |
| Invalid token | 401 | "invalid API token" |
| User account deleted but token remains | 401 | "invalid API token" |
| No active event | 404 | "no active event configured" |
| Match not found in active event | 404 | "match not found in active event" |
| Match locked (started or teams unknown) | 409 | "match is locked (already started or teams unknown)" |
| Malformed JSON body | 400 | "malformed JSON request body" |
| Validation failure (field values) | 422 | "validation failed" + fields object |
| Internal server error | 500 | "internal error" (no implementation details exposed) |

All error responses use `Content-Type: application/json` and have the shape `{"error": "..."}` optionally with `"fields": {...}` for 422s.

## Correctness Properties

### Property 1: Token Uniqueness
The `UNIQUE KEY api_tokens_uc_user` constraint guarantees at most one token per user at the database level. The `Generate` method wraps delete+insert in a transaction to prevent race conditions.

**Validates: Requirements 1.4**

### Property 2: Scope Isolation
The `apiUserID()` function derives user identity exclusively from the validated token context value. No API endpoint accepts a user_id parameter. All queries filter by the token-derived user ID.

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 3: Tipp Locking Consistency
The `acceptsTipps` check (`match.Start.After(time.Now()) && match.TeamA != "" && match.TeamB != ""`) reuses the same logic as the existing web handlers, ensuring the API and web UI enforce identical submission windows.

**Validates: Requirements 6.2**

### Property 4: Idempotent Upsert
`POST /api/v1/tipps` uses the existing `InsertOrUpdate` method, making repeated submissions for the same match safe and deterministic.

**Validates: Requirements 6.1**

### Property 5: Token Secrecy
Plaintext tokens are generated in memory using `crypto/rand` and returned once in the HTTP response. Only bcrypt hashes (cost 12) are persisted to the database.

**Validates: Requirements 1.1, 1.2**

### Property 6: CSRF Immunity
Bearer token auth is immune to CSRF because browsers do not automatically attach Authorization headers to cross-origin requests.

**Validates: Requirements 3.5**

## Testing Strategy

### Unit Tests (`internal/models/api_tokens_test.go`)

- `TestGenerate`: generates token, verifies plaintext has prefix, verifies hash stored in DB
- `TestGenerateRevokesExisting`: generating twice only leaves one row
- `TestValidate`: valid token returns correct user ID
- `TestValidateInvalid`: random string returns ErrInvalidCredentials
- `TestRevoke`: token is deleted
- `TestRevokeNonExistent`: returns ErrNoRecord
- `TestExists`: true when token present, false when not

### Integration Tests (`cmd/web/api_handlers_test.go`)

- `TestApiGetMatches_Authenticated`: valid token → 200, correct JSON shape
- `TestApiGetMatches_NoToken`: missing header → 401
- `TestApiGetMatches_BadToken`: invalid token → 401
- `TestApiGetTipps_Empty`: no tipps → 200, empty array
- `TestApiPostTipp_Success`: valid submission → 200, tipp saved
- `TestApiPostTipp_LockedMatch`: started match → 409
- `TestApiPostTipp_InvalidMatch`: non-existent match → 404
- `TestApiPostTipp_BadValues`: negative scores → 422 with field errors
- `TestApiPostTipp_MalformedJSON`: garbage body → 400

## Security Considerations

1. **Token storage**: Only bcrypt hashes are stored; plaintext is shown once during generation.
2. **No user_id in API**: Tokens derive identity; there is no mechanism to act on behalf of another user.
3. **CSRF bypass is safe**: Bearer tokens are not auto-attached by browsers, so CSRF is not applicable.
4. **Rate limiting**: Not included in this iteration. Could be added via middleware if abuse becomes a concern.
5. **Token rotation**: Generating a new token automatically revokes the old one, allowing users to invalidate compromised tokens.
6. **Bcrypt iteration cost**: Cost 12 (same as password hashing in the app).

## Design Note: Token Lookup

Because bcrypt hashes are salted (non-deterministic), we cannot do `WHERE token_hash = ?`. The `Validate` method iterates over all stored hashes. Since there's at most one token per user and this is a small-group app (~10-50 users), this is acceptable. For scale, a token "hint" column (first 8 hex chars, indexed, unhashed) could narrow lookup to one row.

## Traceability

| Requirement | Implemented By |
|-------------|---------------|
| Req 1: Token Generation | `ApiTokenModel.Generate()` |
| Req 2: Token Management UI | Settings page template + web handlers |
| Req 3: Bearer Auth | `apiAuth` middleware |
| Req 4: Read Matches | `apiGetMatches` handler |
| Req 5: Read Tipps | `apiGetTipps` handler |
| Req 6: Submit Tipp | `apiPostTipp` handler |
| Req 7: Scope Restriction | `apiUserID()` from context, no user_id params |
| Req 8: Error Format | `apiError()` + `apiValidationError()` helpers |
