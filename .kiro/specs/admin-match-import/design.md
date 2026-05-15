# Design Document: Admin Match Import

## Architecture Overview

This feature adds a web-based match import flow to the admin panel. It extracts shared API types and fetch logic into `internal/api`, then builds GET/POST handlers that let admins preview and selectively import match data from the external football API into the database.

The architecture follows the existing patterns in the codebase:
- Handlers in `cmd/web/handlers.go`
- Routes registered in `cmd/web/routes.go` under the admin middleware chain
- Models in `internal/models/`
- Templates in `ui/html/pages/`
- Shared logic in `internal/` packages

```
┌─────────────────┐       ┌──────────────┐       ┌─────────────────┐
│  Admin Browser   │──────▶│  Web Server  │──────▶│  External API   │
│  (admin.html)    │◀──────│  (handlers)  │◀──────│  (football data)│
└─────────────────┘       └──────┬───────┘       └─────────────────┘
                                 │
                          ┌──────▼───────┐
                          │   MySQL DB   │
                          │  (matches)   │
                          └──────────────┘
```

## Components

### 1. Shared API Package (`internal/api/client.go`)

Extracts types and fetch logic from `cmd/cli/main.go` into a reusable package.

```go
package api

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type ApiMatch struct {
    MatchDateTime   string      `json:"matchDateTime"`
    TeamA           ApiTeam     `json:"team1"`
    TeamB           ApiTeam     `json:"team2"`
    MatchResults    []ApiResult `json:"matchResults"`
    MatchIsFinished bool        `json:"matchIsFinished"`
    Goals           []ApiGoal   `json:"goals"`
}

type ApiGoal struct {
    ScoreTeamA     int     `json:"scoreTeam1"`
    ScoreTeamB     int     `json:"scoreTeam2"`
    MatchMinute    int     `json:"matchMinute"`
    GoalGetterID   int     `json:"goalGetterID"`
    GoalGetterName string  `json:"goalGetterName"`
    IsPenalty      bool    `json:"isPenalty"`
    IsOwnGoal     bool    `json:"isOwnGoal"`
    IsOvertime     bool    `json:"isOvertime"`
    Comment        *string `json:"comment"`
}

type ApiTeam struct {
    TeamName string `json:"teamName"`
}

type ApiResult struct {
    ResultName  string `json:"resultName"`
    PointsTeamA int    `json:"pointsTeam1"`
    PointsTeamB int    `json:"pointsTeam2"`
}

// FetchMatchData fetches and decodes match data from the given URL.
func FetchMatchData(url string) ([]ApiMatch, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to fetch data: %s", resp.Status)
    }

    var matches []ApiMatch
    err = json.NewDecoder(resp.Body).Decode(&matches)
    if err != nil {
        return nil, err
    }

    return matches, nil
}
```

### 2. Updated CLI (`cmd/cli/main.go`)

Remove local type definitions and `fetchMatchData` function. Import from `internal/api`:

```go
import (
    "tipp.casualcoding.com/internal/api"
)

// Usage changes:
// api.FetchMatchData(url) instead of fetchMatchData(url)
// api.ApiMatch instead of ApiMatch
// api.ConvertApiGoalToGoal(apiGoal) instead of ConvertApiGoalToGoal(apiGoal)
```

The `ConvertApiGoalToGoal` function also moves to `internal/api/client.go` since it bridges API types to model types.

### 3. Import Handlers (`cmd/web/handlers.go`)

#### GET Handler: Preview

```go
func (app *application) adminImportPhaseGet(w http.ResponseWriter, r *http.Request) {
    // 1. Parse phaseID from URL path
    // 2. Fetch EventPhase by ID (404 if not found)
    // 3. Fetch parent Event by EventPhase.EventID
    // 4. Construct API URL: event.ApiBaseURL + phase.ApiPath
    // 5. Call api.FetchMatchData(url)
    // 6. For each ApiMatch, check duplicate via matchModel.GetByMetadata
    // 7. Build preview data with duplicate flags
    // 8. Render admin_phase_import.html template
}
```

#### POST Handler: Confirm Import

```go
func (app *application) adminImportPhasePost(w http.ResponseWriter, r *http.Request) {
    // 1. Parse phaseID from URL path
    // 2. Fetch EventPhase and Event (same as GET)
    // 3. Re-fetch API data (or receive serialized data from form)
    // 4. Parse selected match indices from form
    // 5. For each selected index, call matchModel.Insert(...)
    // 6. Redirect to /admin with flash message
}
```

### 4. MatchModel.Insert Implementation

The existing stub in `internal/models/matches.go` needs a real implementation:

```go
func (m *MatchModel) Insert(teamA string, teamB string, start time.Time, matchType string, eventPhase int, eventID int) (int, error) {
    stmt := `INSERT INTO matches (team_a, team_b, start, match_type, finished, event_phase, event_id)
             VALUES (?, ?, ?, ?, FALSE, ?, ?)`

    result, err := m.DB.Exec(stmt, teamA, teamB, start, matchType, eventPhase, eventID)
    if err != nil {
        return 0, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }

    return int(id), nil
}
```

Note: The signature changes from the current stub `(teamA, teamB string, start time.Time, matchType string)` to include `eventPhase int, eventID int` parameters.

### 5. Route Registration (`cmd/web/routes.go`)

```go
// Import phase matches
mux.Handle("GET /admin/phases/{phaseID}/import", admin.ThenFunc(app.adminImportPhaseGet))
mux.Handle("POST /admin/phases/{phaseID}/import", admin.ThenFunc(app.adminImportPhasePost))
```

### 6. Template: `admin_phase_import.html`

Renders the preview table with checkboxes, duplicate badges, and a submit form.

### 7. Admin Page Button (`admin.html`)

Adds a "Spieldaten laden" link button in the phase actions column.

## Interfaces

### API Client Interface

```go
// internal/api/client.go
package api

// FetchMatchData fetches match data from the external API.
// Returns a slice of ApiMatch or an error if the request fails.
func FetchMatchData(url string) ([]ApiMatch, error)

// ConvertApiGoalToGoal converts an API goal response to the internal Goal model.
func ConvertApiGoalToGoal(apiGoal ApiGoal) models.Goal
```

### MatchModel.Insert (updated signature)

```go
// Insert creates a new match record and returns the new match ID.
func (m *MatchModel) Insert(teamA string, teamB string, start time.Time, matchType string, eventPhase int, eventID int) (int, error)
```

### Handler Signatures

```go
func (app *application) adminImportPhaseGet(w http.ResponseWriter, r *http.Request)
func (app *application) adminImportPhasePost(w http.ResponseWriter, r *http.Request)
```

### Template Data

The import page uses a dedicated struct for template data (embedded in `templateData` or passed via a field):

```go
type ImportPreviewMatch struct {
    Index     int       // position in the API response slice
    Date      string    // formatted date string
    Time      string    // formatted time string
    TeamA     string
    TeamB     string
    PhaseNum  int       // EventPhase.Number
    IsDuplicate bool   // true if GetByMetadata found a match
}
```

The template data includes:
- `Phase models.EventPhase` — the current phase being imported
- `Event models.Event` — the parent event
- `Matches []ImportPreviewMatch` — the preview rows
- `Error string` — optional error message from API fetch failure

## Data Flow

### GET /admin/phases/{phaseID}/import

1. Parse `phaseID` from path → `int`
2. `eventPhases.Get(phaseID)` → `EventPhase`
3. `events.Get(phase.EventID)` → `Event`
4. Construct URL: `event.ApiBaseURL + phase.ApiPath`
5. `api.FetchMatchData(url)` → `[]ApiMatch`
6. For each `ApiMatch`:
   - Parse `MatchDateTime` → `time.Time`
   - Format date as `"2006-01-02"`
   - Call `matches.GetByMetadata(day, teamA, teamB)` → check if `ID != 0`
   - Build `ImportPreviewMatch` with `IsDuplicate` flag
7. Render template with preview data

### POST /admin/phases/{phaseID}/import

1. Parse `phaseID` from path → `int`
2. `eventPhases.Get(phaseID)` → `EventPhase`
3. `events.Get(phase.EventID)` → `Event`
4. Construct URL and re-fetch: `api.FetchMatchData(url)` → `[]ApiMatch`
5. Parse form: selected indices from checkboxes (`selected_matches[]`)
6. For each selected index:
   - Parse `ApiMatch.MatchDateTime` → `time.Time`
   - Call `matches.Insert(teamA, teamB, start, phase.PhaseType, phase.Number, event.ID)`
7. Set flash: `fmt.Sprintf("%d Spiele erfolgreich importiert!", count)`
8. Redirect to `/admin`

## Error Handling

| Scenario | Response |
|----------|----------|
| Invalid phaseID (not a number or < 1) | HTTP 404 |
| EventPhase not found in DB | HTTP 404 |
| Event not found for phase | HTTP 500 (server error) |
| API fetch fails (network error) | Render template with error message |
| API returns non-200 | Render template with error message |
| Match insertion fails | HTTP 500 with error message |
| No matches selected for import | Redirect with "0 Spiele importiert" flash |

## Template Design

### `admin_phase_import.html`

```html
{{define "title"}}Import: {{.Phase.Title}}{{end}}
{{define "main"}}
<div class="container">
  <h1>Spieldaten importieren</h1>
  <p>Phase: <strong>{{.Phase.Title}}</strong> ({{.Event.Name}})</p>

  {{if .Error}}
  <div class="error-message">{{.Error}}</div>
  {{else}}
  <form method="POST" action="/admin/phases/{{.Phase.ID}}/import">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />
    <table class="pure-table pure-table-bordered">
      <thead>
        <tr>
          <th><input type="checkbox" id="select-all"></th>
          <th>Datum</th>
          <th>Uhrzeit</th>
          <th>Team A</th>
          <th>Team B</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {{range .Matches}}
        <tr>
          <td>
            <input type="checkbox" name="selected_matches" 
                   value="{{.Index}}" {{if not .IsDuplicate}}checked{{end}} />
          </td>
          <td>{{.Date}}</td>
          <td>{{.Time}}</td>
          <td>{{.TeamA}}</td>
          <td>{{.TeamB}}</td>
          <td>
            {{if .IsDuplicate}}
            <span class="badge badge-warning">bereits vorhanden</span>
            {{else}}
            <span class="badge badge-new">neu</span>
            {{end}}
          </td>
        </tr>
        {{end}}
      </tbody>
    </table>
    <button type="submit" class="pure-button pure-button-primary">
      Ausgewählte importieren
    </button>
  </form>
  {{end}}

  <p><a href="/admin" class="pure-button">Zurück</a></p>
</div>
{{end}}
```

### `admin.html` Addition

In the phase actions column, add:

```html
<a href="/admin/phases/{{.ID}}/import" class="pure-button">Spieldaten laden</a>
```

## Styling

Uses Pure CSS classes consistent with the rest of the admin UI:
- `pure-table pure-table-bordered` for the preview table
- `pure-button` and `pure-button-primary` for buttons
- Custom `.badge` classes for status indicators (added to `style.css`)

```css
.badge {
    display: inline-block;
    padding: 0.2em 0.6em;
    font-size: 0.85em;
    border-radius: 3px;
}
.badge-warning {
    background-color: #f0ad4e;
    color: #fff;
}
.badge-new {
    background-color: #5cb85c;
    color: #fff;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Preview row count equals fetched match count

*For any* list of matches returned by the external API, the rendered preview table SHALL contain exactly one row per match — no matches are dropped or duplicated during rendering.

**Validates: Requirements 2.5**

### Property 2: Duplicate status determines checkbox and badge state

*For any* match in the preview table, if `GetByMetadata` returns a match with `ID != 0` (duplicate), then the checkbox SHALL be unchecked and the "bereits vorhanden" badge SHALL be displayed; otherwise the checkbox SHALL be checked and the "neu" badge SHALL be displayed.

**Validates: Requirements 3.2, 3.3, 3.4**

### Property 3: Inserted match fields match API data and phase metadata

*For any* selected ApiMatch and its associated EventPhase/Event, the database insert SHALL receive: `team_a = ApiMatch.TeamA.TeamName`, `team_b = ApiMatch.TeamB.TeamName`, `start = parsed(ApiMatch.MatchDateTime)`, `match_type = EventPhase.PhaseType`, `event_phase = EventPhase.Number`, `event_id = Event.ID`.

**Validates: Requirements 5.2**

### Property 4: Success message count equals number of inserted matches

*For any* set of N matches selected for import where all insertions succeed, the flash message SHALL contain the number N.

**Validates: Requirements 5.3**
