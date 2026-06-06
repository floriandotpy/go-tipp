package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

const (
	JobStatusNoop    = "success_noop"
	JobStatusChanged = "success_changed"
	JobStatusError   = "error"

	JobFetchResults = "fetch-results"
	JobSyncPhases   = "sync-phases"
)

type JobRun struct {
	ID         int
	JobName    string
	Status     string
	Summary    string
	Details    json.RawMessage // nullable JSON
	StartedAt  time.Time
	FinishedAt time.Time
}

type JobRunModel struct {
	DB *sql.DB
}

// Insert records a completed job run.
func (m *JobRunModel) Insert(jobName, status, summary string, details interface{}, startedAt, finishedAt time.Time) error {
	var detailsJSON []byte
	if details != nil {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}

	stmt := `INSERT INTO job_runs (job_name, status, summary, details, started_at, finished_at)
	         VALUES (?, ?, ?, ?, ?, ?)`
	_, err := m.DB.Exec(stmt, jobName, status, summary, detailsJSON, startedAt, finishedAt)
	return err
}

// Recent returns the last n job runs for the given job name, most recent first.
func (m *JobRunModel) Recent(jobName string, limit int) ([]JobRun, error) {
	stmt := `SELECT id, job_name, status, summary, details, started_at, finished_at
	         FROM job_runs
	         WHERE job_name = ?
	         ORDER BY started_at DESC
	         LIMIT ?`

	rows, err := m.DB.Query(stmt, jobName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var r JobRun
		var details []byte
		err := rows.Scan(&r.ID, &r.JobName, &r.Status, &r.Summary, &details, &r.StartedAt, &r.FinishedAt)
		if err != nil {
			return nil, err
		}
		r.Details = details
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// DeleteOlderThan removes job runs older than the given duration.
func (m *JobRunModel) DeleteOlderThan(days int) (int, error) {
	stmt := `DELETE FROM job_runs WHERE started_at < DATE_SUB(NOW(), INTERVAL ? DAY)`
	result, err := m.DB.Exec(stmt, days)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}
