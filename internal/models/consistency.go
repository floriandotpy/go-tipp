package models

import (
	"database/sql"
)

// ConsistencyModel runs read-only data integrity checks against the database.
// It is used by the admin data-consistency monitoring page.
type ConsistencyModel struct {
	DB *sql.DB
}

// ConsistencyCheck holds the outcome of a single consistency check.
type ConsistencyCheck struct {
	Key         string     // stable identifier
	Title       string     // human readable title (German)
	Description string     // what this check looks for
	Count       int        // total number of offending rows
	Columns     []string   // column headers for the sample rows
	Rows        [][]string // capped sample of offending rows
	Truncated   bool       // true if Count > len(Rows)
	Err         string     // populated if the check failed to run
}

// OK reports whether the check found no inconsistencies.
func (c ConsistencyCheck) OK() bool {
	return c.Err == "" && c.Count == 0
}

// sampleLimit caps how many example rows we load per check.
const sampleLimit = 50

// checkDef defines a single consistency check as a pair of SQL statements:
// countSQL returns a single integer (total offending rows), sampleSQL returns
// a capped set of example rows for display.
type checkDef struct {
	key         string
	title       string
	description string
	countSQL    string
	sampleSQL   string
}

var consistencyChecks = []checkDef{
	{
		key:         "matches_without_phase",
		title:       "Spiele ohne zugeordnete Phase",
		description: "Spiele, deren (event_id, event_phase) auf keine Zeile in event_phases zeigt. Solche Spiele werden bei der Punktevergabe übersprungen.",
		countSQL: `SELECT COUNT(*)
			FROM matches m
			LEFT JOIN event_phases ep ON m.event_id = ep.event_id AND m.event_phase = ep.number
			WHERE ep.id IS NULL`,
		sampleSQL: `SELECT m.id, m.event_id, m.event_phase, m.team_a, m.team_b,
			DATE_FORMAT(m.start, '%Y-%m-%d %H:%i') AS start
			FROM matches m
			LEFT JOIN event_phases ep ON m.event_id = ep.event_id AND m.event_phase = ep.number
			WHERE ep.id IS NULL
			ORDER BY m.start
			LIMIT ?`,
	},
	{
		key:         "duplicate_matches",
		title:       "Doppelte Spiele",
		description: "Gruppen von Spielen mit identischen Team-Namen am selben Kalendertag. Deutet auf einen doppelten Import hin.",
		countSQL: `SELECT COUNT(*) FROM (
				SELECT 1
				FROM matches
				GROUP BY team_a, team_b, DATE(start)
				HAVING COUNT(*) > 1
			) d`,
		sampleSQL: `SELECT team_a, team_b, DATE(start) AS match_date,
			COUNT(*) AS anzahl, GROUP_CONCAT(id ORDER BY id) AS match_ids
			FROM matches
			GROUP BY team_a, team_b, DATE(start)
			HAVING COUNT(*) > 1
			ORDER BY match_date
			LIMIT ?`,
	},
	{
		key:         "duplicate_api_match_id",
		title:       "Doppelte API-Match-IDs",
		description: "Mehrere Spiele teilen sich dieselbe api_match_id. Die Ergebnis-Synchronisierung kann dann das falsche Spiel aktualisieren.",
		countSQL: `SELECT COUNT(*) FROM (
				SELECT 1
				FROM matches
				WHERE api_match_id IS NOT NULL
				GROUP BY api_match_id
				HAVING COUNT(*) > 1
			) d`,
		sampleSQL: `SELECT api_match_id, COUNT(*) AS anzahl, GROUP_CONCAT(id ORDER BY id) AS match_ids
			FROM matches
			WHERE api_match_id IS NOT NULL
			GROUP BY api_match_id
			HAVING COUNT(*) > 1
			ORDER BY api_match_id
			LIMIT ?`,
	},
	{
		key:         "points_on_unfinished",
		title:       "Punkte auf nicht beendeten Spielen",
		description: "Tipps mit Punkten oder Korrekt-Markierungen, obwohl das Spiel nicht als beendet markiert ist (finished = 0). Typisch für Live-Zwischenstände.",
		countSQL: `SELECT COUNT(*)
			FROM tipps t
			JOIN matches m ON t.match_id = m.id
			WHERE m.finished = 0
			  AND (t.points <> 0 OR t.result_correct <> 0
			       OR t.tendency_correct <> 0 OR t.goal_difference_correct <> 0)`,
		sampleSQL: `SELECT t.id AS tipp_id, t.user_id, t.match_id,
			CONCAT(m.team_a, ' - ', m.team_b) AS spiel, t.points
			FROM tipps t
			JOIN matches m ON t.match_id = m.id
			WHERE m.finished = 0
			  AND (t.points <> 0 OR t.result_correct <> 0
			       OR t.tendency_correct <> 0 OR t.goal_difference_correct <> 0)
			ORDER BY t.match_id, t.user_id
			LIMIT ?`,
	},
	{
		key:         "finished_without_result",
		title:       "Beendete Spiele ohne Ergebnis",
		description: "Spiele, die als beendet markiert sind, aber kein Ergebnis (result_a/result_b) haben. Für diese können keine Punkte vergeben werden.",
		countSQL: `SELECT COUNT(*)
			FROM matches
			WHERE finished = 1 AND (result_a IS NULL OR result_b IS NULL)`,
		sampleSQL: `SELECT id, event_id, team_a, team_b,
			DATE_FORMAT(start, '%Y-%m-%d %H:%i') AS start
			FROM matches
			WHERE finished = 1 AND (result_a IS NULL OR result_b IS NULL)
			ORDER BY start
			LIMIT ?`,
	},
	{
		key:         "invalid_tipp_points",
		title:       "Logisch inkonsistente Tipp-Punkte",
		description: "Tipps, deren gespeicherte Korrekt-Markierungen sich widersprechen oder deren Punkte nicht zu den Markierungen passen (z. B. Punkte ohne jede Korrekt-Markierung, oder result_correct=1 ohne tendency_correct).",
		countSQL: `SELECT COUNT(*)
			FROM tipps t
			WHERE t.points < 0
			   OR (t.points > 0 AND t.result_correct = 0 AND t.tendency_correct = 0 AND t.goal_difference_correct = 0)
			   OR (t.result_correct = 1 AND (t.tendency_correct = 0 OR t.goal_difference_correct = 0))
			   OR (t.goal_difference_correct = 1 AND t.tendency_correct = 0)`,
		sampleSQL: `SELECT t.id AS tipp_id, t.user_id, t.match_id, t.points,
			t.result_correct, t.tendency_correct, t.goal_difference_correct
			FROM tipps t
			WHERE t.points < 0
			   OR (t.points > 0 AND t.result_correct = 0 AND t.tendency_correct = 0 AND t.goal_difference_correct = 0)
			   OR (t.result_correct = 1 AND (t.tendency_correct = 0 OR t.goal_difference_correct = 0))
			   OR (t.goal_difference_correct = 1 AND t.tendency_correct = 0)
			ORDER BY t.id
			LIMIT ?`,
	},
}

// RunChecks executes all consistency checks and returns their results in a
// stable order. Individual check failures are captured in the check's Err field
// rather than aborting the whole run.
func (m *ConsistencyModel) RunChecks() []ConsistencyCheck {
	results := make([]ConsistencyCheck, 0, len(consistencyChecks))
	for _, def := range consistencyChecks {
		results = append(results, m.runCheck(def))
	}
	return results
}

func (m *ConsistencyModel) runCheck(def checkDef) ConsistencyCheck {
	check := ConsistencyCheck{
		Key:         def.key,
		Title:       def.title,
		Description: def.description,
	}

	var count int
	if err := m.DB.QueryRow(def.countSQL).Scan(&count); err != nil {
		check.Err = err.Error()
		return check
	}
	check.Count = count

	if count == 0 {
		return check
	}

	columns, rows, err := m.querySample(def.sampleSQL)
	if err != nil {
		check.Err = err.Error()
		return check
	}
	check.Columns = columns
	check.Rows = rows
	check.Truncated = count > len(rows)
	return check
}

// querySample runs a sample query with the row limit and returns the column
// names plus the rows rendered as strings (NULL values become "NULL").
func (m *ConsistencyModel) querySample(query string) ([]string, [][]string, error) {
	rows, err := m.DB.Query(query, sampleLimit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var out [][]string
	for rows.Next() {
		raw := make([]sql.RawBytes, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range raw {
			scanArgs[i] = &raw[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, nil, err
		}

		record := make([]string, len(columns))
		for i, rb := range raw {
			if rb == nil {
				record[i] = "NULL"
			} else {
				record[i] = string(rb)
			}
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return columns, out, nil
}
