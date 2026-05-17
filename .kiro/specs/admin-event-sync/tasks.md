# Implementation Plan: Admin Event Sync

## Overview

Replace the per-phase manual import workflow with a unified "Daten synchronisieren" button per event. This involves extending the API client struct, creating a new sync logic package, adding an Upsert method to EventPhaseModel, implementing two new handlers (preview + confirm), creating a new template, updating the admin UI, and removing the old phase creation and per-phase import flows.

## Tasks

- [x] 1. Extend API client and create sync logic package
  - [x] 1.1 Add `ApiGroup` struct and `Group` field to `ApiMatch` in `internal/api/client.go`
    - Add `ApiGroup` struct with `GroupName`, `GroupOrderID`, `GroupID` fields and JSON tags
    - Add `Group ApiGroup` field with `json:"group"` tag to `ApiMatch`
    - _Requirements: 2.1, 2.2_

  - [x] 1.2 Create `internal/sync/sync.go` with pure sync logic functions
    - Implement `InferPhaseType(groupName string) string` — returns `phase_ko` if groupName contains "Finale", "Viertelfinale", "Halbfinale", or "Achtelfinale", otherwise `phase_group`
    - Implement `GroupMatches(matches []api.ApiMatch) map[int][]api.ApiMatch` — groups by `GroupOrderID`
    - Implement `PhaseFromGroup(eventID, groupOrderID int, groupName string, matches []api.ApiMatch) (models.EventPhase, error)` — builds phase with Number, Title, PhaseType, Start (earliest), End (latest), ApiPath=""
    - Define `SyncPreviewPhase` and `SyncPreviewMatch` structs for template rendering
    - _Requirements: 2.3, 3.3, 3.4, 3.5, 3.6, 4.1, 4.2, 11.2_

  - [ ]* 1.3 Write property tests for `InferPhaseType`
    - **Property 3: Phase type inference from keywords**
    - **Validates: Requirements 4.1, 4.2**

  - [ ]* 1.4 Write property tests for `GroupMatches`
    - **Property 2: Distinct group identification**
    - **Validates: Requirements 2.3**

  - [ ]* 1.5 Write property tests for `PhaseFromGroup`
    - **Property 4: Phase date range equals min/max of group matches**
    - **Property 5: Phase construction field mapping**
    - **Validates: Requirements 3.3, 3.4, 3.5, 3.6, 11.2**

- [x] 2. Add EventPhaseModel.Upsert and extend templateData
  - [x] 2.1 Add `Upsert` method to `EventPhaseModel` in `internal/models/event_phases.go`
    - Check if phase exists via `GetByEventAndNumber(eventID, number)`
    - If not found: call `Insert`, return (id, true, nil)
    - If found: set ID on phase, call `Update`, return (id, false, nil)
    - Signature: `func (m *EventPhaseModel) Upsert(ep EventPhase) (int, bool, error)`
    - _Requirements: 3.1, 3.2_

  - [x] 2.2 Add `SyncPreviewPhases` field to `templateData` in `cmd/web/templates.go`
    - Add import for `sync` package
    - Add `SyncPreviewPhases []sync.SyncPreviewPhase` field to `templateData` struct
    - _Requirements: 6.1_

- [x] 3. Implement sync handlers
  - [x] 3.1 Implement `adminSyncEventGet` handler in `cmd/web/handlers.go`
    - Parse eventID from path, fetch event from DB
    - Call `api.FetchMatchData(event.ApiBaseURL)` — render error page on failure
    - Call `sync.GroupMatches(apiMatches)` to group by groupOrderID
    - For each group: build phase via `PhaseFromGroup`, check if phase is new/existing via `GetByEventAndNumber`
    - For each match: check duplicate via `GetByMetadata(day, teamA, teamB)`
    - Build `[]SyncPreviewPhase` with IsNew flag and duplicate indicators
    - Render `admin_sync_preview.html` template
    - _Requirements: 1.2, 1.3, 6.1, 6.2, 6.3, 6.4_

  - [x] 3.2 Implement `adminSyncEventPost` handler in `cmd/web/handlers.go`
    - Parse eventID, fetch event, re-fetch API data
    - Group matches, for each group: build phase, call `Upsert`
    - For each non-duplicate match: call `matches.Insert` with teamA, teamB, parsedTime, phaseType, groupOrderID, eventID
    - Track counts of phasesCreated, phasesUpdated, matchesInserted
    - Set flash message with counts, redirect to `/admin`
    - _Requirements: 3.1, 3.2, 5.1, 5.2, 5.3, 5.4, 5.5, 7.1, 7.2_

- [x] 4. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Update routes and templates
  - [x] 5.1 Update routes in `cmd/web/routes.go`
    - Add `GET /admin/events/{eventID}/sync` → `adminSyncEventGet`
    - Add `POST /admin/events/{eventID}/sync` → `adminSyncEventPost`
    - Remove `GET /admin/events/{eventID}/phases/new` route
    - Remove `POST /admin/events/{eventID}/phases/new` route
    - Remove `GET /admin/phases/{phaseID}/import` route
    - Remove `POST /admin/phases/{phaseID}/import` route
    - _Requirements: 8.2, 8.3, 9.2, 9.3_

  - [x] 5.2 Create `ui/html/pages/admin_sync_preview.html` template
    - Display event name and heading
    - For each phase: show number, title, inferred PhaseType, start, end, and whether it's new or an update
    - For each match in phase: show date, time, teamA, teamB
    - Visually indicate duplicate matches (e.g., strikethrough or muted style)
    - Include a single "Bestätigen" form/button that POSTs to `/admin/events/{eventID}/sync`
    - Show error message if `ImportError` is set
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 5.3 Update `ui/html/pages/admin.html`
    - Replace "Phase hinzufügen" button with "Daten synchronisieren" link pointing to `/admin/events/{eventID}/sync`
    - Remove "Spieldaten laden" link from phase table actions
    - _Requirements: 1.1, 8.1, 9.1_

- [x] 6. Remove obsolete handlers and templates
  - [x] 6.1 Remove `adminAddPhase`, `adminAddPhasePost`, `adminImportPhaseGet`, `adminImportPhasePost` handlers from `cmd/web/handlers.go`
    - Delete the handler function bodies for phase creation and per-phase import
    - _Requirements: 8.2, 8.3, 9.2, 9.3_

  - [x] 6.2 Delete obsolete template files
    - Delete `ui/html/pages/admin_phase_new.html`
    - Delete `ui/html/pages/admin_phase_import.html`
    - _Requirements: 8.1, 9.1_

- [x] 7. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- The phase edit template (`admin_phase_edit.html`) already has a phase_type dropdown — no changes needed (Requirement 10 is already satisfied)
- The `ApiPath` field is retained in the schema; new phases set it to empty string (Requirement 11)

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "2.1"] },
    { "id": 1, "tasks": ["1.2", "2.2"] },
    { "id": 2, "tasks": ["1.3", "1.4", "1.5", "3.1", "3.2"] },
    { "id": 3, "tasks": ["5.1", "5.2", "5.3"] },
    { "id": 4, "tasks": ["6.1", "6.2"] }
  ]
}
```
