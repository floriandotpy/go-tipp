# Implementation Plan: API Tipp Submission

## Overview

This plan implements a REST API for the go-tipp application, adding per-user Bearer token authentication and JSON endpoints for reading matches and submitting tipps. The work is broken into 8 tasks following a bottom-up approach: database first, then model layer, middleware, handlers, routes, UI, and finally end-to-end verification.

## Tasks

- [x] 1. Create database migration for api_tokens table
  - Create migration file `db/migrations/20260603120000_add_api_tokens.sql` with the `api_tokens` table DDL (id, user_id, token_hash char(60), created datetime, PRIMARY KEY, UNIQUE KEY on user_id, FK to users with ON DELETE CASCADE)
  - Update `db/schema.sql` to include the new `api_tokens` table definition
  - **Requirements:** 1.2, 1.4

- [x] 2. Implement ApiTokenModel
  - Create `internal/models/api_tokens.go` with the `ApiToken` struct, `ApiTokenModel` struct, and `apiTokenPrefix` constant (`"gtipp_"`)
  - Implement `Generate(userID int) (string, error)` — generates 32 random bytes with `crypto/rand`, prefixes with `gtipp_`, hashes with bcrypt cost 12, deletes existing token and inserts new hash in a transaction, returns plaintext
  - Implement `Revoke(userID int) error` — deletes the token row for the user, returns `ErrNoRecord` if none exists
  - Implement `Exists(userID int) (bool, error)` — returns true if a token row exists for the user
  - Implement `Validate(plaintext string) (int, error)` — iterates all stored token hashes, compares with bcrypt, returns user ID on match or `ErrInvalidCredentials` if none match
  - **Requirements:** 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 3.4

- [x] 3. Implement API middleware and error helpers
  - Create `cmd/web/api_middleware.go` with context key type `apiContextKey` and constant `apiUserIDKey`
  - Implement `apiAuth` middleware — extracts Bearer token from Authorization header, calls `Validate`, verifies user exists via `users.Get`, stores user ID in context; returns 401 JSON on failure
  - Implement `apiUserID(r *http.Request) int` helper to extract user ID from context
  - Implement `apiError(w, status, message)` — sets Content-Type to application/json, writes status, encodes `{"error": message}`
  - Implement `apiValidationError(w, fields)` — writes 422 with `{"error": "validation failed", "fields": {...}}`
  - **Requirements:** 3.1, 3.2, 3.3, 3.5, 3.6, 8.1, 8.2, 8.3, 8.4, 8.5

- [x] 4. Implement API handlers for matches and tipps
  - Create `cmd/web/api_handlers.go` with JSON response structs (`apiMatch`, `apiTipp`)
  - Implement `apiGetMatches` — gets active event, fetches all matches, computes `accepts_tipps` per match, includes result fields for finished matches, returns JSON array ordered by start time
  - Implement `apiGetTipps` — gets active event, fetches all matches for event to build ID set, fetches user's tipps, filters to active event matches, returns JSON array
  - Implement `apiPostTipp` — decodes JSON body, validates fields (match_id required, tipp_a/tipp_b in 0-99), verifies match exists in active event, checks `acceptsTipps`, calls `InsertOrUpdate`, returns saved tipp as JSON
  - **Requirements:** 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.2, 5.3, 5.4, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 7.1, 7.2

- [x] 5. Register API routes and update application struct
  - Add `apiTokens *models.ApiTokenModel` field to the `application` struct in `cmd/web/main.go`
  - Initialize `apiTokens` in `main()` with `&models.ApiTokenModel{DB: db}`
  - Add API routes in `cmd/web/routes.go`: create `api` middleware chain with `alice.New(app.apiAuth)`, register `GET /api/v1/matches`, `GET /api/v1/tipps`, `POST /api/v1/tipps`
  - Verify the app compiles and API routes respond (401 without token)
  - **Requirements:** 3.5, 4.1, 5.1, 6.1

- [x] 6. Implement token management web handlers
  - Add `TokenExists bool` and `NewToken string` fields to `templateData` in `cmd/web/templates.go`
  - Implement `userSettings` handler in `cmd/web/handlers.go` — checks `apiTokens.Exists`, renders settings page
  - Implement `userGenerateToken` handler — calls `apiTokens.Generate`, renders settings page with plaintext token
  - Implement `userRevokeToken` handler — calls `apiTokens.Revoke`, sets flash message, redirects to settings
  - Register web routes in `cmd/web/routes.go` under the `protected` middleware chain: `GET /user/settings`, `POST /user/settings/token/generate`, `POST /user/settings/token/revoke`
  - **Requirements:** 1.5, 1.6, 1.7, 1.8, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6

- [x] 7. Create settings page template and navigation link
  - Create `ui/html/pages/settings.html` — displays API access status (enabled/disabled), "Generate Token" button (POST form with CSRF token), token display area (read-only input + copy button + warning that token won't be shown again), "Revoke Token" button (POST form, shown when token exists)
  - Add "Einstellungen" navigation link in `ui/html/partials/nav.html` for authenticated users, linking to `/user/settings`
  - Verify the template is auto-discovered by the existing glob pattern in `newTemplateCache()`
  - **Requirements:** 2.1, 2.2, 2.3, 2.4, 2.5

- [x] 8. End-to-end verification
  - Run the database migration against a local MySQL instance
  - Start the server and verify web token management flow: generate token, see plaintext, navigate away, come back (token not shown), revoke token
  - Test API endpoints with curl: `GET /api/v1/matches` with valid Bearer token returns matches JSON, `GET /api/v1/tipps` returns user's tipps, `POST /api/v1/tipps` creates a tipp for an open match
  - Verify error cases: missing auth → 401, invalid token → 401, locked match → 409, bad values → 422, missing match → 404
  - Verify the existing web UI still works (no regressions from route/middleware changes)
  - **Requirements:** 1.1, 2.2, 3.1, 3.2, 3.3, 4.1, 5.1, 6.1, 6.2, 6.3, 6.4, 7.1, 7.2, 8.1, 8.2, 8.3

## Task Dependency Graph

```json
{
  "waves": [
    ["1"],
    ["2"],
    ["3", "6"],
    ["4", "7"],
    ["5"],
    ["8"]
  ]
}
```

Tasks 1→2 form the foundation. After task 2, the API middleware (3) and web handlers (6) can proceed in parallel. Handlers (4) and template (7) follow. Route wiring (5) integrates everything, and task 8 verifies the full feature end-to-end.

## Notes

- The migration file timestamp `20260603120000` follows the existing naming convention in `db/migrations/`.
- The `Validate` method iterates all token hashes (bcrypt is non-deterministic). This is acceptable for the expected user count (10-50 users). A token hint column can be added later for scale.
- No new Go dependencies are needed — `crypto/rand`, `encoding/hex`, and `golang.org/x/crypto/bcrypt` are already available in the project.
