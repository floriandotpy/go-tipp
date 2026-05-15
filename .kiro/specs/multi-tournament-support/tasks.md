# Implementation Plan: Multi-Tournament Support

## Overview

Transform the go-tipp application from a single-tournament system (hardcoded Euro 2024) into a multi-tournament platform. This involves creating new database tables (`events`, `event_phases`), linking matches to events via FK, scoping all queries to the active event, adding admin CRUD for events/phases, and updating the CLI tool to read phases from the database.

## Tasks

- [x] 1. Database migration and new model files
  - [x] 1.1 Create dbmate migration for events and event_phases tables
    - Create file `db/migrations/<timestamp>_add_events_and_event_phases.sql`
    - Migration up: CREATE `events` table, CREATE `event_phases` table, INSERT Euro 2024 seed event, INSERT 7 seed phases, ALTER `matches` to add `event_id` column with FK, UPDATE existing matches to set `event_id` to Euro 2024 ID
    - Migration down: DROP FK on matches, DROP `event_id` column from matches, DROP `event_phases` table, DROP `events` table
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 11.1, 11.2, 11.3, 11.4, 11.5_

  - [x] 1.2 Create `EventModel` in `internal/models/events.go`
    - Rewrite `internal/models/events.go` — remove all hardcoded `Phases`, `NoPhase`, `DetermineEventPhase`, `GetEventPhases`, `GetEventPhaseById`
    - Define `Event` struct with fields: ID, Name, Slug, ApiBaseURL, IsActive, Created
    - Define `EventModel` struct with `DB *sql.DB`
    - Implement methods: `GetActive()`, `GetBySlug(slug string)`, `All()`, `Insert(name, slug, apiBaseURL string)`, `SetActive(eventID int)`
    - `SetActive` must use a transaction to set all events inactive then activate the target
    - _Requirements: 4.1, 4.2, 4.4, 12.1, 12.6_

  - [x] 1.3 Create `EventPhaseModel` in `internal/models/event_phases.go`
    - Define `EventPhase` struct with fields: ID, EventID, Number, Title, ApiPath, PhaseType, Start, End
    - Define `EventPhaseModel` struct with `DB *sql.DB`
    - Implement methods: `AllForEvent(eventID int)`, `GetByEventAndNumber(eventID, number int)`, `DetermineCurrentPhase(eventID int, now time.Time)`, `Insert(ep EventPhase)`
    - `DetermineCurrentPhase` returns the phase where `start <= now <= end`
    - _Requirements: 2.1, 2.4, 2.5, 9.3_

- [x] 2. Checkpoint - Ensure migration and models compile
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Modify existing models for event-scoping
  - [x] 3.1 Update `MatchModel` to accept `eventID` parameter
    - Add `EventID int` field to `Match` struct
    - Modify `All()` → `All(eventID int)` — add `WHERE event_id = ?`
    - Modify `AllByDaterange()` → `AllByDaterange(eventID int, after, before time.Time)` — add `WHERE event_id = ?`
    - Modify `AllMatchesFinished()` → `AllMatchesFinished(eventID int)` — add `WHERE event_id = ?`
    - Update `Get()`, `GetByMetadata()` to also SELECT and scan `event_id`
    - _Requirements: 3.3, 3.5, 5.1, 5.4_

  - [x] 3.2 Update `TippModel.UpdatePoints` to join `event_phases`
    - Modify `UpdatePoints()` → `UpdatePoints(eventID int)`
    - Rewrite `buildUpdatePointsQuery` to JOIN `matches m` → `event_phases ep` ON `m.event_id = ep.event_id AND m.event_phase = ep.number`
    - Use `ep.phase_type` to determine scoring tier instead of hardcoded phase number ranges
    - Add `WHERE m.event_id = ?` to scope to active event
    - Handle matches with no matching `event_phases` row by scoring 0 points
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [x] 3.3 Update `TippModel.GetScoreboardData` to accept `eventID`
    - Modify signature: `GetScoreboardData(groupIds []int, eventID int)`
    - Add JOIN to `matches` and filter `WHERE m.event_id = ?` in the CTE
    - _Requirements: 6.2_

  - [x] 3.4 Update `UserModel` leaderboard methods to accept `eventID`
    - Modify `GroupLeaderboard(groupID int)` → `GroupLeaderboard(groupID, eventID int)` — JOIN tipps → matches, filter by `m.event_id = ?`
    - Modify `GlobalLeaderboard()` → `GlobalLeaderboard(eventID int)` — JOIN tipps → matches, filter by `m.event_id = ?`
    - Modify `GetBestInSelectedPhases(groupID int, phaseIds []int)` → `GetBestInSelectedPhases(groupID, eventID int, phaseIds []int)` — add `m.event_id = ?`
    - _Requirements: 6.1, 6.3, 6.4_

  - [x] 3.5 Update `MatchTippModel.AllByDaterange` to accept `eventID`
    - Pass `eventID` through to `MatchModel.AllByDaterange`
    - _Requirements: 5.1, 5.4_

  - [x] 3.6 Update `InferEventPhaseType` to query the database
    - Change `InferEventPhaseType` to accept a `db *sql.DB` (or move to `EventPhaseModel`) and query `event_phases` by `event_id` and `number` to return `phase_type`
    - Remove the hardcoded switch statement
    - _Requirements: 8.2_

- [x] 4. Checkpoint - Ensure models compile and existing logic is consistent
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Web application middleware and context
  - [x] 5.1 Add `events` and `eventPhases` fields to `application` struct
    - In `cmd/web/main.go`: add `events *models.EventModel` and `eventPhases *models.EventPhaseModel` to the `application` struct
    - Instantiate both models in `main()` and inject into `app`
    - Add startup check: call `events.GetActive()` — if error, log and `os.Exit(1)`
    - _Requirements: 4.3, 5.5_

  - [x] 5.2 Implement `resolveEvent` middleware in `cmd/web/middleware.go`
    - Read `?event=<slug>` query parameter from request URL
    - If present: call `app.events.GetBySlug(slug)` — if not found, return 404
    - If absent: call `app.events.GetActive()` — if error, return 500
    - Store resolved `Event` in request context using a context key
    - _Requirements: 4.4, 10.1, 10.2, 10.5_

  - [x] 5.3 Add context helper functions in `cmd/web/helpers.go`
    - Add `eventFromContext(r *http.Request) models.Event` helper to extract event from context
    - Add `isActiveEvent(r *http.Request) bool` helper to check if resolved event is the active one
    - Update `newTemplateData` to include the resolved event and `IsActiveEvent` flag
    - _Requirements: 10.3_

  - [x] 5.4 Wire `resolveEvent` middleware into route chains in `cmd/web/routes.go`
    - Add `resolveEvent` to the `dynamic` middleware chain (after session load, before handlers)
    - _Requirements: 4.4, 10.2_

- [x] 6. Checkpoint - Ensure middleware compiles and resolves events
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Update handlers to use event context
  - [x] 7.1 Update `matchesHandler` to use event-scoped phases and matches
    - Get event from context
    - Replace `models.GetEventPhases()` with `app.eventPhases.AllForEvent(event.ID)`
    - Replace `models.DetermineEventPhase(time.Now())` with `app.eventPhases.DetermineCurrentPhase(event.ID, time.Now())`
    - Replace `models.GetEventPhaseById(phaseId)` with `app.eventPhases.GetByEventAndNumber(event.ID, phaseId)`
    - Pass `event.ID` to `app.matchTipps.AllByDaterange`
    - _Requirements: 5.1, 5.4_

  - [x] 7.2 Update `matchDetailsHandler` to verify match belongs to active event
    - After fetching match, check `match.EventID == event.ID` — if not, return 404
    - Replace `InferEventPhaseType(&match)` with DB-backed lookup via `app.eventPhases.GetByEventAndNumber(event.ID, match.EventPhase)`
    - _Requirements: 5.2, 5.3_

  - [x] 7.3 Update `tippUpdateMultipleHandler` to enforce active-event check
    - Get event from context, verify `isActiveEvent(r)`
    - For each match in the bulk submission, fetch match and verify `match.EventID == event.ID` — skip if not
    - _Requirements: 7.1, 7.2, 7.3_

  - [x] 7.4 Update `leaderboardHandler` to pass `eventID`
    - Get event from context
    - Pass `event.ID` to `app.users.GroupLeaderboard` and `app.users.GlobalLeaderboard`
    - _Requirements: 6.1, 6.3_

  - [x] 7.5 Update `scoresJsonHandler` to pass `eventID`
    - Get event from context
    - Pass `event.ID` to `app.tipps.GetScoreboardData`
    - _Requirements: 6.2_

  - [x] 7.6 Update `wrappedHandler` to pass `eventID`
    - Get event from context
    - Pass `event.ID` to `app.users.GroupLeaderboard` and `app.users.GetBestInSelectedPhases`
    - _Requirements: 6.1_

  - [x] 7.7 Update `userDetailsHandler` to pass `eventID` to `All` queries
    - Get event from context
    - Pass `event.ID` to `app.matches.All`
    - _Requirements: 5.1_

  - [x] 7.8 Update `adminUpdatePoints` to pass `eventID`
    - Get event from context (active event)
    - Pass `event.ID` to `app.tipps.UpdatePoints`
    - _Requirements: 8.1_

  - [x] 7.9 Update `eventIsFinished` helper to accept `eventID`
    - Modify to pass `eventID` to `app.matches.AllMatchesFinished`
    - Update `newTemplateData` to pass the resolved event ID
    - _Requirements: 5.1_

- [x] 8. Checkpoint - Ensure all handlers compile and use event context
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Admin event management handlers
  - [x] 9.1 Create admin event CRUD handlers in `cmd/web/handlers.go`
    - Implement `adminCreateEvent` (GET) — render event creation form
    - Implement `adminCreateEventPost` (POST) — validate name, slug, api_base_url; insert via `app.events.Insert`; handle duplicate slug error
    - Implement `adminSetActiveEventPost` (POST) — call `app.events.SetActive(eventID)`; flash success message
    - _Requirements: 12.1, 12.2, 12.3, 12.6, 4.5_

  - [x] 9.2 Create admin phase CRUD handlers in `cmd/web/handlers.go`
    - Implement `adminAddPhase` (GET) — render phase creation form for a given event
    - Implement `adminAddPhasePost` (POST) — validate number, title, api_path, phase_type, start < end; insert via `app.eventPhases.Insert`
    - _Requirements: 12.4, 12.5_

  - [x] 9.3 Register admin routes in `cmd/web/routes.go`
    - Add routes: `GET /admin/events/new`, `POST /admin/events/new`, `POST /admin/events/setactive`, `GET /admin/events/{eventID}/phases/new`, `POST /admin/events/{eventID}/phases/new`
    - _Requirements: 12.1, 12.4, 12.6_

  - [x] 9.4 Create admin HTML templates for event and phase forms
    - Create `ui/html/pages/admin_event_new.html` with form fields for name, slug, api_base_url
    - Create `ui/html/pages/admin_phase_new.html` with form fields for number, title, api_path, phase_type, start, end
    - Update `ui/html/pages/admin.html` to list events and link to create/activate actions
    - _Requirements: 12.1, 12.4_

- [x] 10. Checkpoint - Ensure admin handlers compile and routes are registered
  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. CLI tool update
  - [x] 11.1 Update `cmd/cli/main.go` to use database-driven phases
    - Remove import/usage of `models.DetermineEventPhase`
    - Instantiate `EventModel` and `EventPhaseModel` with the DB connection
    - Call `eventModel.GetActive()` — if error, print message and `os.Exit(1)`
    - Call `eventPhaseModel.DetermineCurrentPhase(event.ID, time.Now())` — if error, print message and `os.Exit(1)`
    - Construct URL as `event.ApiBaseURL + phase.ApiPath`
    - Pass `event.ID` to `tippModel.UpdatePoints`
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

- [x] 12. Read-only mode for non-active events
  - [x] 12.1 Update templates to hide tipp inputs for non-active events
    - In `matches.html`: conditionally hide tipp input fields when `IsActiveEvent` is false
    - In `match_details.html`: conditionally hide tipp-related UI when viewing non-active event
    - _Requirements: 10.3, 10.4_

- [x] 13. Final checkpoint - Full compilation and integration
  - Ensure all tests pass, ask the user if questions arise.

- [ ]* 14. Property-based tests
  - [ ]* 14.1 Write property test for exactly one active event invariant
    - **Property 1: Exactly One Active Event Invariant**
    - **Validates: Requirements 4.1, 4.2, 12.6**

  - [ ]* 14.2 Write property test for event-scoped match filtering
    - **Property 2: Event-Scoped Match Filtering**
    - **Validates: Requirements 3.5, 5.1, 5.4**

  - [ ]* 14.3 Write property test for event-scoped point aggregation
    - **Property 3: Event-Scoped Point Aggregation**
    - **Validates: Requirements 6.1, 6.2, 6.3**

  - [ ]* 14.4 Write property test for tipp rejection for non-active event match
    - **Property 4: Tipp Rejection for Non-Active Event Match**
    - **Validates: Requirements 7.1, 10.4**

  - [ ]* 14.5 Write property test for bulk tipp partial processing
    - **Property 5: Bulk Tipp Partial Processing**
    - **Validates: Requirements 7.3**

  - [ ]* 14.6 Write property test for phase-type-driven scoring
    - **Property 6: Phase-Type-Driven Scoring**
    - **Validates: Requirements 8.1, 8.2, 8.4**

  - [ ]* 14.7 Write property test for phase selection by timestamp
    - **Property 7: Phase Selection by Timestamp**
    - **Validates: Requirements 9.3**

  - [ ]* 14.8 Write property test for valid event and phase data accepted
    - **Property 8: Valid Event and Phase Data Accepted**
    - **Validates: Requirements 12.1, 12.4**

  - [ ]* 14.9 Write property test for invalid event and phase data rejected
    - **Property 9: Invalid Event and Phase Data Rejected**
    - **Validates: Requirements 12.2, 12.5, 2.4, 2.6**

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document using the `rapid` library
- The migration must be a single dbmate file with both `-- migrate:up` and `-- migrate:down` sections
- All existing Euro 2024 data is preserved via backfill in the migration
- The `scoring` package (`internal/scoring/config.go`) remains unchanged — `PhasePointsMap` is still used, but looked up by `phase_type` from the DB rather than hardcoded phase number ranges

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.3"] },
    { "id": 2, "tasks": ["3.1", "3.6"] },
    { "id": 3, "tasks": ["3.2", "3.3", "3.4", "3.5"] },
    { "id": 4, "tasks": ["5.1"] },
    { "id": 5, "tasks": ["5.2", "5.3"] },
    { "id": 6, "tasks": ["5.4"] },
    { "id": 7, "tasks": ["7.1", "7.2", "7.3", "7.4", "7.5", "7.6", "7.7", "7.8", "7.9"] },
    { "id": 8, "tasks": ["9.1", "9.2", "11.1"] },
    { "id": 9, "tasks": ["9.3", "9.4"] },
    { "id": 10, "tasks": ["12.1"] },
    { "id": 11, "tasks": ["14.1", "14.2", "14.3", "14.4", "14.5", "14.6", "14.7", "14.8", "14.9"] }
  ]
}
```
