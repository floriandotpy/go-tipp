# Implementation Plan: Admin CRUD Enhancements

## Overview

Extend the admin interface with Update and Delete operations for events and phases. Implementation follows the existing patterns: model methods → templateData extension → handlers → routes → templates. Each step builds incrementally on the previous one.

## Tasks

- [x] 1. Add model methods for Event and EventPhase CRUD operations
  - [x] 1.1 Add `Get`, `Update`, and `Delete` methods to `EventModel`
    - Add `Get(id int) (Event, error)` method that queries by ID and returns `ErrNoRecord` if not found
    - Add `Update(id int, name, slug, apiBaseURL string) error` method with duplicate slug detection (`ErrDuplicateSlug`) and `ErrNoRecord` on zero rows affected
    - Add `Delete(id int) error` method using a transaction to cascade-delete goals, tipps, matches, then the event itself (phases cascade via FK)
    - _Requirements: 2.2, 4.4_

  - [x] 1.2 Add `Get`, `Update`, and `Delete` methods to `EventPhaseModel`
    - Add `Get(id int) (EventPhase, error)` method that queries by ID and returns `ErrNoRecord` if not found
    - Add `Update(ep EventPhase) error` method that updates number, title, api_path, phase_type, start, end by ID; returns `ErrNoRecord` on zero rows affected
    - Add `Delete(id int) error` method that deletes by ID; returns `ErrNoRecord` on zero rows affected
    - _Requirements: 3.2, 5.4_

  - [ ]* 1.3 Write property tests for EventModel round-trip and cascade delete
    - **Property 1: Event update round-trip** — After `Update`, `Get` returns updated values
    - **Property 7: Event cascade delete removes all associated data** — After `Delete`, event/phases/matches/tipps/goals all return no records
    - **Validates: Requirements 2.2, 4.4**

  - [ ]* 1.4 Write property tests for EventPhaseModel round-trip and delete
    - **Property 5: Phase update round-trip** — After `Update`, `Get` returns updated values
    - **Property 8: Phase delete removes the phase record** — After `Delete`, `Get` returns `ErrNoRecord`
    - **Validates: Requirements 3.2, 5.4**

- [x] 2. Extend templateData and modify adminIndex handler
  - [x] 2.1 Add `EventPhasesMap map[int][]models.EventPhase` field to `templateData` struct in `cmd/web/templates.go`
    - _Requirements: 1.1_

  - [x] 2.2 Modify `adminIndex` handler to load phases for all events into `EventPhasesMap`
    - Iterate over events, call `app.eventPhases.AllForEvent(event.ID)` for each, populate the map
    - _Requirements: 1.1, 1.2, 1.4_

- [x] 3. Implement edit and delete event handlers
  - [x] 3.1 Add `eventEditForm` struct and `adminEditEvent` GET handler
    - Define `eventEditForm` with Name, Slug, ApiBaseURL fields and embedded `validator.Validator`
    - Handler parses `{eventID}` path value, calls `app.events.Get(eventID)`, renders `admin_event_edit.html` with pre-filled form
    - _Requirements: 2.1_

  - [x] 3.2 Add `adminEditEventPost` POST handler
    - Decode form, validate (NotBlank name, valid slug regex, NotBlank api_base_url), call `app.events.Update`
    - Handle `ErrDuplicateSlug` and `ErrNoRecord` errors, flash success message, redirect to `/admin`
    - _Requirements: 2.2, 2.3, 2.4, 2.5, 2.6_

  - [x] 3.3 Add `adminDeleteEventPost` POST handler
    - Parse `{eventID}` path value, call `app.events.Delete(eventID)`, handle `ErrNoRecord`, flash success, redirect
    - _Requirements: 4.2, 4.4, 4.5, 4.6_

  - [ ]* 3.4 Write property tests for event form validation
    - **Property 2: Event validation rejects blank required fields**
    - **Property 3: Event validation rejects invalid slug format**
    - **Property 4: Event validation rejects duplicate slugs**
    - **Validates: Requirements 2.3, 2.4, 2.5, 2.6**

- [x] 4. Implement edit and delete phase handlers
  - [x] 4.1 Add `phaseEditForm` struct and `adminEditPhase` GET handler
    - Define `phaseEditForm` with Number, Title, ApiPath, PhaseType, Start, End fields and embedded `validator.Validator`
    - Handler parses `{phaseID}` path value, calls `app.eventPhases.Get(phaseID)`, renders `admin_phase_edit.html` with pre-filled form (dates formatted as `2006-01-02T15:04`)
    - _Requirements: 3.1_

  - [x] 4.2 Add `adminEditPhasePost` POST handler
    - Decode form, validate (number ≥ 1, NotBlank title/api_path, PermittedValue phase_type, parse and validate start/end timestamps, start < end)
    - Look up existing phase for event_id, call `app.eventPhases.Update(ep)`, handle errors, flash success, redirect
    - _Requirements: 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_

  - [x] 4.3 Add `adminDeletePhasePost` POST handler
    - Parse `{phaseID}` path value, call `app.eventPhases.Delete(phaseID)`, handle `ErrNoRecord`, flash success, redirect
    - _Requirements: 5.2, 5.4, 5.5, 5.6_

  - [ ]* 4.4 Write property tests for phase form validation
    - **Property 6: Phase validation rejects invalid inputs** — number < 1, blank title/api_path, invalid phase_type, end ≤ start
    - **Validates: Requirements 3.3, 3.4, 3.5, 3.6, 3.7**

- [x] 5. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Register new routes and create templates
  - [x] 6.1 Register 6 new routes in `cmd/web/routes.go` under the `admin` alice chain
    - `GET /admin/events/{eventID}/edit` → `adminEditEvent`
    - `POST /admin/events/{eventID}/edit` → `adminEditEventPost`
    - `POST /admin/events/{eventID}/delete` → `adminDeleteEventPost`
    - `GET /admin/phases/{phaseID}/edit` → `adminEditPhase`
    - `POST /admin/phases/{phaseID}/edit` → `adminEditPhasePost`
    - `POST /admin/phases/{phaseID}/delete` → `adminDeletePhasePost`
    - _Requirements: 6.3, 6.4_

  - [x] 6.2 Create `ui/html/pages/admin_event_edit.html` template
    - Same form layout as `admin_event_new.html` but with action `/admin/events/{{.Event.ID}}/edit`, pre-filled values, and submit button text "Event aktualisieren"
    - Include CSRF token hidden field
    - _Requirements: 2.1, 6.1_

  - [x] 6.3 Create `ui/html/pages/admin_phase_edit.html` template
    - Same form layout as `admin_phase_new.html` but with action `/admin/phases/{{.Event.ID}}/edit` (using phase ID from form context), pre-filled values, and submit button text "Phase aktualisieren"
    - Include CSRF token hidden field
    - _Requirements: 3.1, 6.1_

  - [x] 6.4 Modify `ui/html/pages/admin.html` to show inline phases and action buttons
    - Display phases nested beneath each event row using `{{index $.EventPhasesMap .ID}}`
    - Add "Bearbeiten" link and "Löschen" button (with confirm dialog) for each event
    - Add "Bearbeiten" link and "Löschen" button (with confirm dialog) for each phase
    - Show "Keine Phasen konfiguriert." message when no phases exist for an event
    - _Requirements: 1.1, 1.2, 1.3, 2.7, 3.8, 4.1, 4.7, 5.1, 5.7_

- [x] 7. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- The design uses Go code directly, so all implementation uses Go
- All new routes are protected by the existing `requireAdminAuthentication` middleware and `noSurf` CSRF middleware — no additional security setup needed
- The `ErrDuplicateSlug` error is already defined in the models package (used by `EventModel.Insert`)

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2"] },
    { "id": 1, "tasks": ["1.3", "1.4", "2.1"] },
    { "id": 2, "tasks": ["2.2", "3.1", "4.1"] },
    { "id": 3, "tasks": ["3.2", "3.3", "3.4", "4.2", "4.3"] },
    { "id": 4, "tasks": ["4.4", "6.1"] },
    { "id": 5, "tasks": ["6.2", "6.3", "6.4"] }
  ]
}
```
