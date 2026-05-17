package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Event struct {
	ID         int
	Name       string
	Slug       string
	ApiBaseURL string
	IsActive   bool
	Created    time.Time
}

type EventModel struct {
	DB *sql.DB
}

// GetActive returns the currently active event.
func (m *EventModel) GetActive() (Event, error) {
	stmt := `SELECT id, name, slug, api_base_url, is_active, created
		FROM events WHERE is_active = TRUE`

	var event Event
	err := m.DB.QueryRow(stmt).Scan(
		&event.ID,
		&event.Name,
		&event.Slug,
		&event.ApiBaseURL,
		&event.IsActive,
		&event.Created,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Event{}, ErrNoRecord
		}
		return Event{}, err
	}

	return event, nil
}

// GetBySlug returns the event matching the given slug.
func (m *EventModel) GetBySlug(slug string) (Event, error) {
	stmt := `SELECT id, name, slug, api_base_url, is_active, created
		FROM events WHERE slug = ?`

	var event Event
	err := m.DB.QueryRow(stmt, slug).Scan(
		&event.ID,
		&event.Name,
		&event.Slug,
		&event.ApiBaseURL,
		&event.IsActive,
		&event.Created,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Event{}, ErrNoRecord
		}
		return Event{}, err
	}

	return event, nil
}

// All returns all events ordered by creation date descending.
func (m *EventModel) All() ([]Event, error) {
	stmt := `SELECT id, name, slug, api_base_url, is_active, created
		FROM events ORDER BY created DESC`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event

	for rows.Next() {
		var event Event
		err = rows.Scan(
			&event.ID,
			&event.Name,
			&event.Slug,
			&event.ApiBaseURL,
			&event.IsActive,
			&event.Created,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// Insert creates a new event with is_active = false and returns the new event ID.
func (m *EventModel) Insert(name, slug, apiBaseURL string) (int, error) {
	stmt := `INSERT INTO events (name, slug, api_base_url, is_active, created)
		VALUES (?, ?, ?, FALSE, UTC_TIMESTAMP())`

	result, err := m.DB.Exec(stmt, name, slug, apiBaseURL)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "events_uc_slug") {
				return 0, ErrDuplicateSlug
			}
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

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

// SetActive atomically sets all events inactive then activates the target event.
func (m *EventModel) SetActive(eventID int) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE events SET is_active = FALSE`)
	if err != nil {
		return err
	}

	result, err := tx.Exec(`UPDATE events SET is_active = TRUE WHERE id = ?`, eventID)
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
