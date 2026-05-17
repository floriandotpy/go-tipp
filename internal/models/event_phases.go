package models

import (
	"database/sql"
	"errors"
	"time"
)

type EventPhase struct {
	ID        int
	EventID   int
	Number    int
	Title     string
	PhaseType string // "phase_group" or "phase_ko"
	Start     time.Time
	End       time.Time
}

type EventPhaseModel struct {
	DB *sql.DB
}

// AllForEvent returns all phases for an event ordered by number ascending.
func (m *EventPhaseModel) AllForEvent(eventID int) ([]EventPhase, error) {
	stmt := `SELECT id, event_id, number, title, phase_type, start, end
             FROM event_phases
             WHERE event_id = ?
             ORDER BY number ASC`

	rows, err := m.DB.Query(stmt, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phases []EventPhase

	for rows.Next() {
		var ep EventPhase
		err = rows.Scan(&ep.ID, &ep.EventID, &ep.Number, &ep.Title, &ep.PhaseType, &ep.Start, &ep.End)
		if err != nil {
			return nil, err
		}
		phases = append(phases, ep)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return phases, nil
}

// GetByEventAndNumber returns a single phase for the given event and number,
// or ErrNoRecord if not found.
func (m *EventPhaseModel) GetByEventAndNumber(eventID, number int) (EventPhase, error) {
	stmt := `SELECT id, event_id, number, title, phase_type, start, end
             FROM event_phases
             WHERE event_id = ? AND number = ?`

	var ep EventPhase
	err := m.DB.QueryRow(stmt, eventID, number).Scan(
		&ep.ID, &ep.EventID, &ep.Number, &ep.Title, &ep.PhaseType, &ep.Start, &ep.End)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventPhase{}, ErrNoRecord
		}
		return EventPhase{}, err
	}

	return ep, nil
}

// DetermineCurrentPhase returns the phase where start <= now <= end for the given event.
// Returns ErrNoRecord if no phase matches the current time.
func (m *EventPhaseModel) DetermineCurrentPhase(eventID int, now time.Time) (EventPhase, error) {
	stmt := `SELECT id, event_id, number, title, phase_type, start, end
             FROM event_phases
             WHERE event_id = ? AND start <= ? AND end >= ?`

	var ep EventPhase
	err := m.DB.QueryRow(stmt, eventID, now, now).Scan(
		&ep.ID, &ep.EventID, &ep.Number, &ep.Title, &ep.PhaseType, &ep.Start, &ep.End)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventPhase{}, ErrNoRecord
		}
		return EventPhase{}, err
	}

	return ep, nil
}

// Get returns a single phase by ID, or ErrNoRecord if not found.
func (m *EventPhaseModel) Get(id int) (EventPhase, error) {
	stmt := `SELECT id, event_id, number, title, phase_type, start, end
		FROM event_phases WHERE id = ?`
	var ep EventPhase
	err := m.DB.QueryRow(stmt, id).Scan(
		&ep.ID, &ep.EventID, &ep.Number, &ep.Title,
		&ep.PhaseType, &ep.Start, &ep.End,
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
func (m *EventPhaseModel) Update(ep EventPhase) error {
	stmt := `UPDATE event_phases
		SET number = ?, title = ?, phase_type = ?, start = ?, end = ?
		WHERE id = ?`
	_, err := m.DB.Exec(stmt, ep.Number, ep.Title, ep.PhaseType, ep.Start, ep.End, ep.ID)
	return err
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

// Upsert inserts a new phase or updates an existing one based on event_id and number.
// Returns the phase ID and whether it was newly created (true) or updated (false).
func (m *EventPhaseModel) Upsert(ep EventPhase) (int, bool, error) {
	existing, err := m.GetByEventAndNumber(ep.EventID, ep.Number)
	if err != nil && !errors.Is(err, ErrNoRecord) {
		return 0, false, err
	}

	if errors.Is(err, ErrNoRecord) {
		id, err := m.Insert(ep)
		if err != nil {
			return 0, false, err
		}
		return id, true, nil
	}

	ep.ID = existing.ID
	err = m.Update(ep)
	if err != nil {
		return 0, false, err
	}
	return existing.ID, false, nil
}

// Insert inserts a new event phase and returns the new phase ID.
func (m *EventPhaseModel) Insert(ep EventPhase) (int, error) {
	stmt := `INSERT INTO event_phases (event_id, number, title, phase_type, start, end)
             VALUES (?, ?, ?, ?, ?, ?)`

	result, err := m.DB.Exec(stmt, ep.EventID, ep.Number, ep.Title, ep.PhaseType, ep.Start, ep.End)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}
