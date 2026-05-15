# Implementation Plan: Admin Match Import

## Overview

This plan implements a web-based match import flow for the admin panel. It extracts shared API types into `internal/api`, updates the CLI to use the shared package, implements the `MatchModel.Insert` method, adds GET/POST handlers for previewing and importing matches, creates the import template, and wires everything together with route registration and UI updates.

## Tasks

- [x] 1. Extract shared API package
  - [x] 1.1 Create `internal/api/client.go` with exported types and FetchMatchData function
    - Move `ApiMatch`, `ApiTeam`, `ApiResult`, `ApiGoal` structs from `cmd/cli/main.go` to `internal/api/client.go`
    - Move `fetchMatchData` as exported `FetchMatchData(url string) ([]ApiMatch, error)`
    - Move `ConvertApiGoalToGoal` as exported function
    - Package name: `api`
    - _Requirements: 6.1, 6.2, 6.3_

  - [x] 1.2 Update `cmd/cli/main.go` to import from `internal/api`
    - Remove local type definitions (`ApiMatch`, `ApiTeam`, `ApiResult`, `ApiGoal`)
    - Remove local `fetchMatchData` and `ConvertApiGoalToGoal` functions
    - Import `tipp.casualcoding.com/internal/api`
    - Replace all usages: `api.FetchMatchData(url)`, `api.ApiMatch`, `api.ConvertApiGoalToGoal(apiGoal)`
    - _Requirements: 6.4_

- [x] 2. Implement MatchModel.Insert with real DB logic
  - [x] 2.1 Update `MatchModel.Insert` in `internal/models/matches.go`
    - Change signature to `Insert(teamA, teamB string, start time.Time, matchType string, eventPhase int, eventID int) (int, error)`
    - Implement with `INSERT INTO matches (team_a, team_b, start, match_type, finished, event_phase, event_id) VALUES (?, ?, ?, ?, FALSE, ?, ?)`
    - Return the `LastInsertId` as int
    - _Requirements: 5.2_

- [x] 3. Checkpoint - Verify shared package compiles
  - Ensure `go build ./...` passes, ask the user if questions arise.

- [x] 4. Add import handlers
  - [x] 4.1 Add `ImportPreviewMatch` struct and GET handler `adminImportPhaseGet` in `cmd/web/handlers.go`
    - Define `ImportPreviewMatch` struct with fields: `Index int`, `Date string`, `Time string`, `TeamA string`, `TeamB string`, `PhaseNum int`, `IsDuplicate bool`
    - Parse `phaseID` from URL path, return 404 if invalid
    - Fetch `EventPhase` by ID (404 if not found)
    - Fetch parent `Event` by `phase.EventID`
    - Construct API URL: `event.ApiBaseURL + phase.ApiPath`
    - Call `api.FetchMatchData(url)` — render error in template if it fails
    - For each `ApiMatch`, parse `MatchDateTime`, check duplicate via `matchModel.GetByMetadata`
    - Build `[]ImportPreviewMatch` with `IsDuplicate` flag
    - Add `ImportPreviewMatches []ImportPreviewMatch` field to `templateData` struct in `templates.go`
    - Render `admin_phase_import.html` template with phase, event, matches, and optional error
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3, 3.4_

  - [x] 4.2 Add POST handler `adminImportPhasePost` in `cmd/web/handlers.go`
    - Parse `phaseID` from URL path, return 404 if invalid
    - Fetch `EventPhase` and `Event` (same as GET)
    - Re-fetch API data via `api.FetchMatchData(url)`
    - Parse form values: `selected_matches` (slice of index strings)
    - For each selected index, parse `ApiMatch.MatchDateTime` and call `matchModel.Insert(teamA, teamB, start, phase.PhaseType, phase.Number, event.ID)`
    - On success: set flash message `fmt.Sprintf("%d Spiele erfolgreich importiert!", count)` and redirect to `/admin`
    - On error: return HTTP 500
    - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 5. Register import routes
  - [x] 5.1 Add route registrations in `cmd/web/routes.go`
    - Add `GET /admin/phases/{phaseID}/import` with admin middleware chain
    - Add `POST /admin/phases/{phaseID}/import` with admin middleware chain
    - _Requirements: 7.1, 7.2, 7.3_

- [x] 6. Checkpoint - Verify handlers compile
  - Ensure `go build ./...` passes, ask the user if questions arise.

- [x] 7. Create admin import template and UI updates
  - [x] 7.1 Create `ui/html/pages/admin_phase_import.html`
    - Use `{{define "title"}}Import: {{.Phase.Title}}{{end}}`
    - Display phase title and event name
    - Show error message if API fetch failed
    - Render a `pure-table` with columns: checkbox, Datum, Uhrzeit, Team A, Team B, Status
    - Checkbox `name="selected_matches"` with `value="{{.Index}}"`, checked if not duplicate
    - Show `.badge-warning` "bereits vorhanden" for duplicates, `.badge-new` "neu" for new matches
    - Include "select all" checkbox in header
    - Submit button "Ausgewählte importieren" and "Zurück" link
    - Include CSRF token hidden field
    - _Requirements: 2.5, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3_

  - [x] 7.2 Add "Spieldaten laden" button to `ui/html/pages/admin.html`
    - Add an `<a href="/admin/phases/{{.ID}}/import" class="pure-button">Spieldaten laden</a>` link in the phase actions column (next to "Bearbeiten" and "Löschen")
    - _Requirements: 1.1, 1.2_

  - [x] 7.3 Add badge CSS to `ui/static/css/style.css`
    - Add `.badge` base class with inline-block, padding, font-size, border-radius
    - Add `.badge-warning` with orange background (#f0ad4e) and white text
    - Add `.badge-new` with green background (#5cb85c) and white text
    - _Requirements: 3.2_

- [x] 8. Final checkpoint
  - Ensure `go build ./...` passes and all templates render correctly, ask the user if questions arise.

## Notes

- The design uses Go, so all tasks use Go for implementation
- The `templateData` struct needs a new field for import preview data; the `Phase` and `Event` fields already exist
- The existing `MatchModel.Insert` stub signature changes (adds `eventPhase` and `eventID` params) — no other callers exist currently
- `ConvertApiGoalToGoal` in `internal/api` will need to import `tipp.casualcoding.com/internal/models` for the `models.Goal` type
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["4.1", "7.3"] },
    { "id": 3, "tasks": ["4.2", "7.1", "7.2"] },
    { "id": 4, "tasks": ["5.1"] }
  ]
}
```
