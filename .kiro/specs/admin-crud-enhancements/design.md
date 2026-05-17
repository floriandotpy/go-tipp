# Design Document: Admin CRUD Enhancements

## Architecture Overview

This feature extends the existing admin interface with Update and Delete operations for events and phases. The architecture follows the established patterns in the codebase:

- **Handler layer** (`cmd/web/handlers.go`): New handler functions for edit/delete operations using form structs with embedded `validator.Validator`
- **Model layer** (`internal/models/`): New `Update`, `Delete`, and `Get` methods on `EventModel` and `EventPhaseModel`
- **Template layer** (`ui/html/pages/`): New edit form templates and updated admin index template with inline phases
- **Routing** (`cmd/web/routes.go`): New routes registered under the `admin` alice chain

All new routes are protected by the existing `requireAdminAuthentication` middleware and `noSurf` CSRF middleware.

## Components

### 1. Model Layer Extensions

#### EventModel — New Methods

```go
// Get returns a single event by ID, or ErrNoRecord if not found.
func (m *EventModel) Get(id int) (Event, error) {
    stmt := `SELECT id, name, slug, api_base_url, is_active, created
        FROM events WHERE id = ?`
    var event Event
    err := m.DB.QueryRow(stmt, id).Scan(
        &event.ID, &event.Name, &event.Slug,
        &event.ApiBaseURL, &event.IsActive, &event.Created,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return Event{}, ErrNoRecord
        }
        return Event{}, err
    }
    return event, nil
}

// Update modifies an existing event's name, slug, and api_base_url.
// Returns ErrNoRecord if the event doesn't exist, ErrDuplicateSlug if slug conflicts.
func (m *EventModel) Update(id int, name, slug, apiBaseURL string) error {
    stmt := `UPDATE events SET name = ?, slug = ?, api_base_url = ? WHERE id = ?`
    result, err := m.DB.Exec(stmt, name, slug, apiBaseURL, id)
    if err != nil {
        var mySQLError *mysql.MySQLError
        if errors.As(err, &mySQLError) {
            if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "events_uc_slug") {
                return ErrDuplicateSlug
            }
        }
        return err
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return ErrNoRecord
    }
    return nil
}

// Delete removes an event and all associated data (phases, matches, tipps, goals)
// within a transaction. Required because the matches FK does not cascade.
func (m *EventModel) Delete(id int) error {
    tx, err := m.DB.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Delete goals for all matches belonging to this event
    _, err = tx.Exec(`DELETE g FROM goals g
        INNER JOIN matches m ON g.match_id = m.id
        WHERE m.event_id = ?`, id)
    if err != nil {
        return err
    }

    // Delete tipps for all matches belonging to this event
    _, err = tx.Exec(`DELETE t FROM tipps t
        INNER JOIN matches m ON t.match_id = m.id
        WHERE m.event_id = ?`, id)
    if err != nil {
        return err
    }

    // Delete matches belonging to this event
    _, err = tx.Exec(`DELETE FROM matches WHERE event_id = ?`, id)
    if err != nil {
        return err
    }

    // Delete the event (phases cascade via FK)
    result, err := tx.Exec(`DELETE FROM events WHERE id = ?`, id)
    if err != nil {
        return err
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return ErrNoRecord
    }

    return tx.Commit()
}
```

#### EventPhaseModel — New Methods

```go
// Get returns a single phase by ID, or ErrNoRecord if not found.
func (m *EventPhaseModel) Get(id int) (EventPhase, error) {
    stmt := `SELECT id, event_id, number, title, api_path, phase_type, start, end
        FROM event_phases WHERE id = ?`
    var ep EventPhase
    err := m.DB.QueryRow(stmt, id).Scan(
        &ep.ID, &ep.EventID, &ep.Number, &ep.Title,
        &ep.ApiPath, &ep.PhaseType, &ep.Start, &ep.End,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return EventPhase{}, ErrNoRecord
        }
        return EventPhase{}, err
    }
    return ep, nil
}

// Update modifies an existing phase's fields.
// Returns ErrNoRecord if the phase doesn't exist.
func (m *EventPhaseModel) Update(ep EventPhase) error {
    stmt := `UPDATE event_phases
        SET number = ?, title = ?, api_path = ?, phase_type = ?, start = ?, end = ?
        WHERE id = ?`
    result, err := m.DB.Exec(stmt, ep.Number, ep.Title, ep.ApiPath, ep.PhaseType, ep.Start, ep.End, ep.ID)
    if err != nil {
        return err
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return ErrNoRecord
    }
    return nil
}

// Delete removes a phase by ID. Returns ErrNoRecord if not found.
func (m *EventPhaseModel) Delete(id int) error {
    result, err := m.DB.Exec(`DELETE FROM event_phases WHERE id = ?`, id)
    if err != nil {
        return err
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return ErrNoRecord
    }
    return nil
}
```

### 2. Template Data Extension

The `templateData` struct gains a new field to pass phases grouped by event:

```go
type templateData struct {
    // ... existing fields ...
    EventPhasesMap map[int][]models.EventPhase // key: event ID
}
```

### 3. Handler Layer

#### Admin Index Handler (Modified)

The existing `adminIndex` handler is extended to load phases for all events:

```go
func (app *application) adminIndex(w http.ResponseWriter, r *http.Request) {
    data := app.newTemplateData(r)

    groups, err := app.groups.All()
    if err != nil {
        app.serverError(w, r, err)
        return
    }
    data.Groups = groups

    events, err := app.events.All()
    if err != nil {
        app.serverError(w, r, err)
        return
    }
    data.Events = events

    // Load phases grouped by event
    phasesMap := make(map[int][]models.EventPhase)
    for _, event := range events {
        phases, err := app.eventPhases.AllForEvent(event.ID)
        if err != nil {
            app.serverError(w, r, err)
            return
        }
        phasesMap[event.ID] = phases
    }
    data.EventPhasesMap = phasesMap

    app.render(w, r, http.StatusOK, "admin.html", data)
}
```

#### Edit Event Handlers

```go
type eventEditForm struct {
    Name                string `form:"name"`
    Slug                string `form:"slug"`
    ApiBaseURL          string `form:"api_base_url"`
    validator.Validator `form:"-"`
}

func (app *application) adminEditEvent(w http.ResponseWriter, r *http.Request) {
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

    data := app.newTemplateData(r)
    data.Form = eventEditForm{
        Name:       event.Name,
        Slug:       event.Slug,
        ApiBaseURL: event.ApiBaseURL,
    }
    data.Event = event
    app.render(w, r, http.StatusOK, "admin_event_edit.html", data)
}

func (app *application) adminEditEventPost(w http.ResponseWriter, r *http.Request) {
    eventID, err := strconv.Atoi(r.PathValue("eventID"))
    if err != nil || eventID < 1 {
        http.NotFound(w, r)
        return
    }

    var form eventEditForm
    err = app.decodePostForm(r, &form)
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(validator.NotBlank(form.Name), "name", "Darf nicht leer sein")
    form.CheckField(validator.NotBlank(form.Slug), "slug", "Darf nicht leer sein")
    form.CheckField(validator.Matches(form.Slug, slugRX), "slug", "Nur Kleinbuchstaben, Zahlen und Bindestriche erlaubt")
    form.CheckField(validator.NotBlank(form.ApiBaseURL), "api_base_url", "Darf nicht leer sein")

    if !form.Valid() {
        data := app.newTemplateData(r)
        data.Form = form
        data.Event = models.Event{ID: eventID}
        app.render(w, r, http.StatusUnprocessableEntity, "admin_event_edit.html", data)
        return
    }

    err = app.events.Update(eventID, form.Name, form.Slug, form.ApiBaseURL)
    if err != nil {
        if errors.Is(err, models.ErrDuplicateSlug) {
            form.AddFieldError("slug", "Dieser Slug ist bereits vergeben")
            data := app.newTemplateData(r)
            data.Form = form
            data.Event = models.Event{ID: eventID}
            app.render(w, r, http.StatusUnprocessableEntity, "admin_event_edit.html", data)
        } else if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Event erfolgreich aktualisiert!")
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
```

#### Delete Event Handler

```go
func (app *application) adminDeleteEventPost(w http.ResponseWriter, r *http.Request) {
    eventID, err := strconv.Atoi(r.PathValue("eventID"))
    if err != nil || eventID < 1 {
        http.NotFound(w, r)
        return
    }

    err = app.events.Delete(eventID)
    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Event erfolgreich gelöscht!")
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
```

#### Edit Phase Handlers

```go
type phaseEditForm struct {
    Number              int    `form:"number"`
    Title               string `form:"title"`
    ApiPath             string `form:"api_path"`
    PhaseType           string `form:"phase_type"`
    Start               string `form:"start"`
    End                 string `form:"end"`
    validator.Validator `form:"-"`
}

func (app *application) adminEditPhase(w http.ResponseWriter, r *http.Request) {
    phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
    if err != nil || phaseID < 1 {
        http.NotFound(w, r)
        return
    }

    phase, err := app.eventPhases.Get(phaseID)
    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    const timeLayout = "2006-01-02T15:04"
    data := app.newTemplateData(r)
    data.Form = phaseEditForm{
        Number:    phase.Number,
        Title:     phase.Title,
        ApiPath:   phase.ApiPath,
        PhaseType: phase.PhaseType,
        Start:     phase.Start.Format(timeLayout),
        End:       phase.End.Format(timeLayout),
    }
    data.Event = models.Event{ID: phase.EventID}
    app.render(w, r, http.StatusOK, "admin_phase_edit.html", data)
}

func (app *application) adminEditPhasePost(w http.ResponseWriter, r *http.Request) {
    phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
    if err != nil || phaseID < 1 {
        http.NotFound(w, r)
        return
    }

    var form phaseEditForm
    err = app.decodePostForm(r, &form)
    if err != nil {
        app.clientError(w, http.StatusBadRequest)
        return
    }

    form.CheckField(form.Number >= 1, "number", "Muss mindestens 1 sein")
    form.CheckField(validator.NotBlank(form.Title), "title", "Darf nicht leer sein")
    form.CheckField(validator.NotBlank(form.ApiPath), "api_path", "Darf nicht leer sein")
    form.CheckField(validator.PermittedValue(form.PhaseType, "phase_group", "phase_ko"), "phase_type", "Muss 'phase_group' oder 'phase_ko' sein")

    const timeLayout = "2006-01-02T15:04"
    var startTime, endTime time.Time

    form.CheckField(validator.NotBlank(form.Start), "start", "Darf nicht leer sein")
    form.CheckField(validator.NotBlank(form.End), "end", "Darf nicht leer sein")

    if validator.NotBlank(form.Start) {
        startTime, err = time.Parse(timeLayout, form.Start)
        if err != nil {
            form.AddFieldError("start", "Ungültiges Datumsformat")
        }
    }
    if validator.NotBlank(form.End) {
        endTime, err = time.Parse(timeLayout, form.End)
        if err != nil {
            form.AddFieldError("end", "Ungültiges Datumsformat")
        }
    }
    if !startTime.IsZero() && !endTime.IsZero() {
        form.CheckField(startTime.Before(endTime), "end", "Ende muss nach dem Start liegen")
    }

    if !form.Valid() {
        // Look up the phase to get the event ID for the template
        phase, _ := app.eventPhases.Get(phaseID)
        data := app.newTemplateData(r)
        data.Form = form
        data.Event = models.Event{ID: phase.EventID}
        app.render(w, r, http.StatusUnprocessableEntity, "admin_phase_edit.html", data)
        return
    }

    // Look up existing phase to get event_id
    existingPhase, err := app.eventPhases.Get(phaseID)
    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    ep := models.EventPhase{
        ID:        phaseID,
        EventID:   existingPhase.EventID,
        Number:    form.Number,
        Title:     form.Title,
        ApiPath:   form.ApiPath,
        PhaseType: form.PhaseType,
        Start:     startTime,
        End:       endTime,
    }

    err = app.eventPhases.Update(ep)
    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Phase erfolgreich aktualisiert!")
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
```

#### Delete Phase Handler

```go
func (app *application) adminDeletePhasePost(w http.ResponseWriter, r *http.Request) {
    phaseID, err := strconv.Atoi(r.PathValue("phaseID"))
    if err != nil || phaseID < 1 {
        http.NotFound(w, r)
        return
    }

    err = app.eventPhases.Delete(phaseID)
    if err != nil {
        if errors.Is(err, models.ErrNoRecord) {
            http.NotFound(w, r)
        } else {
            app.serverError(w, r, err)
        }
        return
    }

    app.sessionManager.Put(r.Context(), "flash", "Phase erfolgreich gelöscht!")
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
```

### 4. Routes

New routes added to the `admin` chain in `routes.go`:

```go
// Edit event
mux.Handle("GET /admin/events/{eventID}/edit", admin.ThenFunc(app.adminEditEvent))
mux.Handle("POST /admin/events/{eventID}/edit", admin.ThenFunc(app.adminEditEventPost))

// Delete event
mux.Handle("POST /admin/events/{eventID}/delete", admin.ThenFunc(app.adminDeleteEventPost))

// Edit phase
mux.Handle("GET /admin/phases/{phaseID}/edit", admin.ThenFunc(app.adminEditPhase))
mux.Handle("POST /admin/phases/{phaseID}/edit", admin.ThenFunc(app.adminEditPhasePost))

// Delete phase
mux.Handle("POST /admin/phases/{phaseID}/delete", admin.ThenFunc(app.adminDeletePhasePost))
```

### 5. Templates

#### admin_event_edit.html

Reuses the same form layout as `admin_event_new.html` but with pre-filled values and a POST action pointing to the edit endpoint.

#### admin_phase_edit.html

Reuses the same form layout as `admin_phase_new.html` but with pre-filled values and a POST action pointing to the edit endpoint.

#### admin.html (Modified)

Extended to:
- Show phases nested beneath each event row using `{{index $.EventPhasesMap .ID}}`
- Add "Bearbeiten" and "Löschen" buttons for each event
- Add "Bearbeiten" and "Löschen" buttons for each phase
- Delete buttons wrapped in forms with `onclick="return confirm('Wirklich löschen?')"` for confirmation

## Data Flow

### Edit Event Flow

```
GET /admin/events/{eventID}/edit
  → adminEditEvent handler
  → EventModel.Get(eventID)
  → render admin_event_edit.html with pre-filled form

POST /admin/events/{eventID}/edit
  → adminEditEventPost handler
  → decodePostForm → validate
  → EventModel.Update(id, name, slug, apiBaseURL)
  → flash "Event erfolgreich aktualisiert!"
  → redirect /admin
```

### Delete Event Flow

```
POST /admin/events/{eventID}/delete
  → adminDeleteEventPost handler
  → EventModel.Delete(eventID)
    → BEGIN TRANSACTION
    → DELETE goals (via JOIN on matches.event_id)
    → DELETE tipps (via JOIN on matches.event_id)
    → DELETE matches WHERE event_id = ?
    → DELETE events WHERE id = ? (phases cascade via FK)
    → COMMIT
  → flash "Event erfolgreich gelöscht!"
  → redirect /admin
```

### Edit Phase Flow

```
GET /admin/phases/{phaseID}/edit
  → adminEditPhase handler
  → EventPhaseModel.Get(phaseID)
  → render admin_phase_edit.html with pre-filled form

POST /admin/phases/{phaseID}/edit
  → adminEditPhasePost handler
  → decodePostForm → validate
  → EventPhaseModel.Update(ep)
  → flash "Phase erfolgreich aktualisiert!"
  → redirect /admin
```

### Delete Phase Flow

```
POST /admin/phases/{phaseID}/delete
  → adminDeletePhasePost handler
  → EventPhaseModel.Delete(phaseID)
  → flash "Phase erfolgreich gelöscht!"
  → redirect /admin
```

## Error Handling

| Scenario | Response |
|----------|----------|
| Event/Phase not found (GET) | 404 Not Found |
| Event/Phase not found (POST delete) | 404 Not Found |
| Invalid form data | 422 with form re-rendered and field errors |
| Duplicate slug on update | 422 with "Dieser Slug ist bereits vergeben" error |
| Database error | 500 Internal Server Error |
| Invalid path parameter (non-numeric ID) | 404 Not Found |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Event update round-trip

*For any* existing event and any valid update values (non-blank name, valid slug, non-blank API base URL), after calling `Update(id, name, slug, apiBaseURL)`, calling `Get(id)` should return an event with the updated name, slug, and API base URL.

**Validates: Requirements 2.2**

### Property 2: Event validation rejects blank required fields

*For any* string that is empty or composed entirely of whitespace, submitting it as the event name or API base URL in the edit form should result in a validation failure and the form should not be considered valid.

**Validates: Requirements 2.3, 2.6**

### Property 3: Event validation rejects invalid slug format

*For any* string that does not match the pattern `^[a-z0-9]+(?:-[a-z0-9]+)*$`, submitting it as the slug in the edit event form should result in a validation failure on the slug field.

**Validates: Requirements 2.4**

### Property 4: Event validation rejects duplicate slugs

*For any* two distinct events, if the second event's slug is updated to match the first event's slug, the update operation should return a duplicate slug error.

**Validates: Requirements 2.5**

### Property 5: Phase update round-trip

*For any* existing phase and any valid update values (number ≥ 1, non-blank title, non-blank API path, valid phase type, start < end), after calling `Update(ep)`, calling `Get(ep.ID)` should return a phase with the updated field values.

**Validates: Requirements 3.2**

### Property 6: Phase validation rejects invalid inputs

*For any* integer less than 1 submitted as the phase number, the form validation should reject it. *For any* string that is empty or whitespace submitted as title or API path, validation should reject it. *For any* string that is not "phase_group" or "phase_ko" submitted as phase type, validation should reject it. *For any* pair of timestamps where end ≤ start, validation should reject the end field.

**Validates: Requirements 3.3, 3.4, 3.5, 3.6, 3.7**

### Property 7: Event cascade delete removes all associated data

*For any* event with associated phases, matches, tipps, and goals, after calling `Delete(eventID)`, querying for the event, its phases, its matches, their tipps, and their goals should all return no records.

**Validates: Requirements 4.4**

### Property 8: Phase delete removes the phase record

*For any* existing phase, after calling `Delete(phaseID)`, calling `Get(phaseID)` should return `ErrNoRecord`.

**Validates: Requirements 5.4**
