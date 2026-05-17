# Design Document: Admin Event Sync

## Overview

This feature replaces the per-phase manual import workflow with a unified "Daten synchronisieren" button per event. The system fetches all match data from the external API in one call, auto-creates/updates phases based on group metadata, detects duplicates, and presents a preview before persisting. The manual phase creation and per-phase import flows are removed.

## Architecture

The sync feature follows the existing server-rendered MVC pattern of the application:

1. **API Layer** (`internal/api/client.go`) — Extended `ApiMatch` struct with `Group` field; reuse of `FetchMatchData`
2. **Sync Logic** (new `internal/sync/sync.go`) — Pure functions for grouping, phase type inference, phase construction, and duplicate filtering
3. **Model Layer** (`internal/models/event_phases.go`) — New `Upsert` method on `EventPhaseModel`
4. **Handler Layer** (`cmd/web/handlers.go`) — Two new handlers: GET preview + POST confirm
5. **Template Layer** — New `admin_sync_preview.html`, updated `admin.html`

```
┌─────────────┐     GET /admin/events/{id}/sync      ┌──────────────┐
│  Admin UI   │ ──────────────────────────────────────▶│  SyncPreview │
│  (button)   │                                       │   Handler    │
└─────────────┘                                       └──────┬───────┘
                                                             │
                                                             ▼
                                                    ┌────────────────┐
                                                    │ FetchMatchData │
                                                    │  (API client)  │
                                                    └────────┬───────┘
                                                             │
                                                             ▼
                                                    ┌────────────────┐
                                                    │  Sync Logic    │
                                                    │ (group, infer, │
                                                    │  build phases) │
                                                    └────────┬───────┘
                                                             │
                                                             ▼
                                                    ┌────────────────┐
                                                    │ Render Preview │
                                                    │   Template     │
                                                    └────────────────┘

┌─────────────┐    POST /admin/events/{id}/sync      ┌──────────────┐
│  Bestätigen │ ──────────────────────────────────────▶│ SyncConfirm  │
│   Button    │                                       │   Handler    │
└─────────────┘                                       └──────┬───────┘
                                                             │
                                                             ▼
                                                    ┌────────────────┐
                                                    │ Re-fetch API + │
                                                    │ Upsert Phases  │
                                                    │ Insert Matches │
                                                    └────────┬───────┘
                                                             │
                                                             ▼
                                                    ┌────────────────┐
                                                    │ Redirect /admin│
                                                    │ + flash message│
                                                    └────────────────┘
```

## Components

### 1. Extended ApiMatch Struct

**File:** `internal/api/client.go`

```go
type ApiGroup struct {
	GroupName    string `json:"groupName"`
	GroupOrderID int    `json:"groupOrderID"`
	GroupID      int    `json:"groupID"`
}

type ApiMatch struct {
	MatchDateTime   string      `json:"matchDateTime"`
	TeamA           ApiTeam     `json:"team1"`
	TeamB           ApiTeam     `json:"team2"`
	MatchResults    []ApiResult `json:"matchResults"`
	MatchIsFinished bool        `json:"matchIsFinished"`
	Goals           []ApiGoal   `json:"goals"`
	Group           ApiGroup    `json:"group"`
}
```

The existing `FetchMatchData` function is reused without modification — it already decodes the full JSON response into `[]ApiMatch`. The new `Group` field is populated automatically by `json.Decoder`.

### 2. Sync Logic Package

**File:** `internal/sync/sync.go`

This package contains pure functions with no database or HTTP dependencies, making them easy to test.

```go
package sync

import (
	"strings"
	"time"

	"tipp.casualcoding.com/internal/api"
	"tipp.casualcoding.com/internal/models"
)

// koKeywords are substrings that indicate a knockout phase.
var koKeywords = []string{"Finale", "Viertelfinale", "Halbfinale", "Achtelfinale"}

// InferPhaseType returns "phase_ko" if groupName contains any KO keyword,
// otherwise "phase_group".
func InferPhaseType(groupName string) string {
	for _, kw := range koKeywords {
		if strings.Contains(groupName, kw) {
			return "phase_ko"
		}
	}
	return "phase_group"
}

// GroupedMatches groups API matches by their GroupOrderID.
func GroupMatches(matches []api.ApiMatch) map[int][]api.ApiMatch {
	groups := make(map[int][]api.ApiMatch)
	for _, m := range matches {
		groups[m.Group.GroupOrderID] = append(groups[m.Group.GroupOrderID], m)
	}
	return groups
}

// PhaseFromGroup constructs an EventPhase from a group of API matches.
// It sets Number, Title, PhaseType, Start, End, and ApiPath (empty string).
func PhaseFromGroup(eventID int, groupOrderID int, groupName string, matches []api.ApiMatch) (models.EventPhase, error) {
	var earliest, latest time.Time
	for _, m := range matches {
		t, err := time.Parse("2006-01-02T15:04:05", m.MatchDateTime)
		if err != nil {
			return models.EventPhase{}, err
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}

	return models.EventPhase{
		EventID:   eventID,
		Number:    groupOrderID,
		Title:     groupName,
		ApiPath:   "",
		PhaseType: InferPhaseType(groupName),
		Start:     earliest,
		End:       latest,
	}, nil
}

// SyncPreviewPhase holds phase data plus its matches for the preview template.
type SyncPreviewPhase struct {
	Phase   models.EventPhase
	IsNew   bool // true = will be inserted, false = will be updated
	Matches []SyncPreviewMatch
}

// SyncPreviewMatch holds a single match for the preview template.
type SyncPreviewMatch struct {
	Date        string
	Time        string
	TeamA       string
	TeamB       string
	IsDuplicate bool
}
```

### 3. EventPhaseModel.Upsert Method

**File:** `internal/models/event_phases.go`

```go
// Upsert inserts a new phase or updates an existing one based on event_id and number.
// Returns the phase ID and whether it was newly created (true) or updated (false).
func (m *EventPhaseModel) Upsert(ep EventPhase) (int, bool, error) {
	existing, err := m.GetByEventAndNumber(ep.EventID, ep.Number)
	if err != nil && !errors.Is(err, ErrNoRecord) {
		return 0, false, err
	}

	if errors.Is(err, ErrNoRecord) {
		// Insert new phase
		id, err := m.Insert(ep)
		if err != nil {
			return 0, false, err
		}
		return id, true, nil
	}

	// Update existing phase
	ep.ID = existing.ID
	err = m.Update(ep)
	if err != nil {
		return 0, false, err
	}
	return existing.ID, false, nil
}
```

### 4. Sync Handlers

**File:** `cmd/web/handlers.go`

#### GET Handler (Preview)

```go
func (app *application) adminSyncEventGet(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	event, err := app.events.Get(eventID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	// Fetch all match data from the event's API
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil {
		data := app.newTemplateData(r)
		data.Event = event
		data.ImportError = fmt.Sprintf("Fehler beim Abrufen der API-Daten: %v", err)
		app.render(w, r, http.StatusOK, "admin_sync_preview.html", data)
		return
	}

	// Group matches and build preview
	grouped := sync.GroupMatches(apiMatches)
	var previewPhases []sync.SyncPreviewPhase

	// ... build preview phases with duplicate detection ...

	data := app.newTemplateData(r)
	data.Event = event
	data.SyncPreviewPhases = previewPhases
	app.render(w, r, http.StatusOK, "admin_sync_preview.html", data)
}
```

#### POST Handler (Confirm)

```go
func (app *application) adminSyncEventPost(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.Atoi(r.PathValue("eventID"))
	if err != nil || eventID < 1 {
		http.NotFound(w, r)
		return
	}

	event, err := app.events.Get(eventID)
	if err != nil { /* ... */ }

	// Re-fetch API data
	apiMatches, err := api.FetchMatchData(event.ApiBaseURL)
	if err != nil { /* ... */ }

	// Group and process
	grouped := sync.GroupMatches(apiMatches)
	var phasesCreated, phasesUpdated, matchesInserted int

	for groupOrderID, groupMatches := range grouped {
		groupName := groupMatches[0].Group.GroupName
		phase, err := sync.PhaseFromGroup(eventID, groupOrderID, groupName, groupMatches)
		if err != nil { /* ... */ }

		_, isNew, err := app.eventPhases.Upsert(phase)
		if err != nil { /* ... */ }
		if isNew {
			phasesCreated++
		} else {
			phasesUpdated++
		}

		// Insert non-duplicate matches
		for _, am := range groupMatches {
			parsedTime, _ := time.Parse("2006-01-02T15:04:05", am.MatchDateTime)
			day := parsedTime.Format("2006-01-02")
			existing, _ := app.matches.GetByMetadata(day, am.TeamA.TeamName, am.TeamB.TeamName)
			if existing.ID != 0 {
				continue // duplicate, skip
			}
			_, err = app.matches.Insert(
				am.TeamA.TeamName, am.TeamB.TeamName,
				parsedTime, phase.PhaseType, groupOrderID, eventID,
			)
			if err != nil { /* ... */ }
			matchesInserted++
		}
	}

	msg := fmt.Sprintf("Sync abgeschlossen: %d Phasen erstellt, %d aktualisiert, %d Spiele importiert.",
		phasesCreated, phasesUpdated, matchesInserted)
	app.sessionManager.Put(r.Context(), "flash", msg)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
```

### 5. Route Changes

**File:** `cmd/web/routes.go`

```go
// ADD:
mux.Handle("GET /admin/events/{eventID}/sync", admin.ThenFunc(app.adminSyncEventGet))
mux.Handle("POST /admin/events/{eventID}/sync", admin.ThenFunc(app.adminSyncEventPost))

// REMOVE:
// mux.Handle("GET /admin/events/{eventID}/phases/new", admin.ThenFunc(app.adminAddPhase))
// mux.Handle("POST /admin/events/{eventID}/phases/new", admin.ThenFunc(app.adminAddPhasePost))
// mux.Handle("GET /admin/phases/{phaseID}/import", admin.ThenFunc(app.adminImportPhaseGet))
// mux.Handle("POST /admin/phases/{phaseID}/import", admin.ThenFunc(app.adminImportPhasePost))
```

### 6. Template Changes

#### New: `admin_sync_preview.html`

Displays all phases with their matches, indicates duplicates, and provides a single "Bestätigen" button.

#### Updated: `admin.html`

- Replace `<a href="/admin/events/{{.ID}}/phases/new" class="pure-button">Phase hinzufügen</a>` with:
  ```html
  <a href="/admin/events/{{.ID}}/sync" class="pure-button">Daten synchronisieren</a>
  ```
- Remove the "Spieldaten laden" link from the phase table actions.

#### Updated: `admin_phase_edit.html`

The phase type dropdown is already implemented as a `<select>` element with `phase_group` and `phase_ko` options. No changes needed — this requirement is already satisfied.

### 7. Template Data Extensions

**File:** `cmd/web/templates.go`

```go
type templateData struct {
	// ... existing fields ...
	SyncPreviewPhases []sync.SyncPreviewPhase
}
```

## Data Flow

### Sync Preview (GET)

1. Admin clicks "Daten synchronisieren" for event X
2. Handler fetches event from DB by ID
3. Handler calls `api.FetchMatchData(event.ApiBaseURL)` — single HTTP GET
4. Handler calls `sync.GroupMatches(apiMatches)` — groups by `groupOrderID`
5. For each group:
   - Calls `sync.PhaseFromGroup(...)` to build phase metadata
   - Checks `eventPhases.GetByEventAndNumber(...)` to determine if phase is new or existing
   - For each match in group: calls `matches.GetByMetadata(...)` to detect duplicates
6. Handler renders `admin_sync_preview.html` with all preview data

### Sync Confirm (POST)

1. Admin clicks "Bestätigen" on the preview page
2. Handler re-fetches API data (ensures freshness)
3. Handler groups matches and for each group:
   - Builds phase via `sync.PhaseFromGroup(...)`
   - Calls `eventPhases.Upsert(phase)` — inserts or updates
   - For each non-duplicate match: calls `matches.Insert(...)`
4. Handler sets flash message with counts and redirects to `/admin`

## Error Handling

| Scenario | Behavior |
|----------|----------|
| API returns non-200 | Render preview page with `ImportError` message |
| API returns invalid JSON | Render preview page with `ImportError` message |
| Match datetime parse failure | Return 500 server error |
| Database error during upsert | Return 500 server error |
| Event not found | Return 404 |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Group JSON round-trip

*For any* valid JSON object containing a `group` field with `groupName` (string), `groupOrderID` (int), and `groupID` (int), marshalling it into an `ApiMatch` struct and then reading the `Group` field SHALL produce the original values.

**Validates: Requirements 2.1, 2.2**

### Property 2: Distinct group identification

*For any* list of `ApiMatch` values, `GroupMatches` SHALL return a map where each key is a unique `groupOrderID` and each value contains exactly the matches with that `groupOrderID`.

**Validates: Requirements 2.3**

### Property 3: Phase type inference from keywords

*For any* string `groupName`, `InferPhaseType(groupName)` SHALL return `"phase_ko"` if and only if `groupName` contains at least one of "Finale", "Viertelfinale", "Halbfinale", or "Achtelfinale"; otherwise it SHALL return `"phase_group"`.

**Validates: Requirements 4.1, 4.2**

### Property 4: Phase date range equals min/max of group matches

*For any* non-empty list of `ApiMatch` values belonging to the same group, `PhaseFromGroup` SHALL set `Start` to the earliest `matchDateTime` and `End` to the latest `matchDateTime` among all matches in that group.

**Validates: Requirements 3.5, 3.6**

### Property 5: Phase construction field mapping

*For any* group with `groupOrderID` G and `groupName` N, `PhaseFromGroup(eventID, G, N, matches)` SHALL produce an `EventPhase` where `Number == G`, `Title == N`, `PhaseType == InferPhaseType(N)`, and `ApiPath == ""`.

**Validates: Requirements 3.3, 3.4, 11.2**

### Property 6: Sync idempotence for duplicate matches

*For any* set of API matches and existing database matches, running the sync confirmation twice with the same API response SHALL not create duplicate matches — the second run inserts zero new matches because all are detected as duplicates via day+teamA+teamB.

**Validates: Requirements 5.1, 5.2, 5.3**
