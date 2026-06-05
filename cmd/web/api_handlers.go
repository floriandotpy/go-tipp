package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"tipp.casualcoding.com/internal/models"
)

// --- JSON response structs ---

type apiMatch struct {
	ID           int    `json:"id"`
	TeamA        string `json:"team_a"`
	TeamB        string `json:"team_b"`
	Start        string `json:"start"`
	MatchType    string `json:"match_type"`
	EventPhase   int    `json:"event_phase"`
	Finished     bool   `json:"finished"`
	AcceptsTipps bool   `json:"accepts_tipps"`
	ResultA      *int   `json:"result_a,omitempty"`
	ResultB      *int   `json:"result_b,omitempty"`
	ResultAetA   *int   `json:"result_aet_a,omitempty"`
	ResultAetB   *int   `json:"result_aet_b,omitempty"`
	ResultApenA  *int   `json:"result_apen_a,omitempty"`
	ResultApenB  *int   `json:"result_apen_b,omitempty"`
}

type apiTipp struct {
	MatchID int    `json:"match_id"`
	TippA   int    `json:"tipp_a"`
	TippB   int    `json:"tipp_b"`
	Created string `json:"created"`
	Changed string `json:"changed"`
}

// --- Handlers ---

// apiGetMatches handles GET /api/v1/matches
func (app *application) apiGetMatches(w http.ResponseWriter, r *http.Request) {
	event, err := app.events.GetActive()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.apiError(w, http.StatusNotFound, "no active event configured")
			return
		}
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	matches, err := app.matches.AllWithResults(event.ID)
	if err != nil {
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	now := time.Now()
	result := make([]apiMatch, 0, len(matches))

	for _, m := range matches {
		acceptsTipps := m.Start.After(now) && m.TeamA != "" && m.TeamB != ""

		am := apiMatch{
			ID:           m.ID,
			TeamA:        m.TeamA,
			TeamB:        m.TeamB,
			Start:        m.Start.UTC().Format(time.RFC3339),
			MatchType:    m.MatchType,
			EventPhase:   m.EventPhase,
			Finished:     m.Finished,
			AcceptsTipps: acceptsTipps,
		}

		if m.Finished {
			am.ResultA = m.ResultA
			am.ResultB = m.ResultB
			am.ResultAetA = m.ResultAETA
			am.ResultAetB = m.ResultAETB
			am.ResultApenA = m.ResultAPenA
			am.ResultApenB = m.ResultAPenB
		}

		result = append(result, am)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// apiGetTipps handles GET /api/v1/tipps
func (app *application) apiGetTipps(w http.ResponseWriter, r *http.Request) {
	event, err := app.events.GetActive()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.apiError(w, http.StatusNotFound, "no active event configured")
			return
		}
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := apiUserID(r)

	// Get all matches for the event to build a set of valid match IDs.
	matches, err := app.matches.All(event.ID)
	if err != nil {
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	matchIDs := make(map[int]struct{}, len(matches))
	for _, m := range matches {
		matchIDs[m.ID] = struct{}{}
	}

	// Fetch all tipps for the user.
	tipps, err := app.tipps.AllForUser(userID)
	if err != nil {
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Filter tipps to only those belonging to the active event.
	result := make([]apiTipp, 0)
	for _, t := range tipps {
		if _, ok := matchIDs[t.MatchId]; ok {
			result = append(result, apiTipp{
				MatchID: t.MatchId,
				TippA:   t.TippA,
				TippB:   t.TippB,
				Created: t.Created.UTC().Format(time.RFC3339),
				Changed: t.Changed.UTC().Format(time.RFC3339),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

const maxBatchSize = 200

// apiBatchTippResult is the per-item result in the response.
type apiBatchTippResult struct {
	MatchID int    `json:"match_id"`
	TippA   int    `json:"tipp_a"`
	TippB   int    `json:"tipp_b"`
	Created string `json:"created"`
	Changed string `json:"changed"`
	Status  string `json:"status"` // "created" or "updated"
}

// apiPostTipps handles POST /api/v1/tipps
// Accepts an array of tipps and processes them all-or-nothing.
// For a single tipp, send an array with one element.
func (app *application) apiPostTipps(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input struct {
		Tipps []struct {
			MatchID *int `json:"match_id"`
			TippA   *int `json:"tipp_a"`
			TippB   *int `json:"tipp_b"`
		} `json:"tipps"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		app.apiError(w, http.StatusBadRequest, "malformed JSON request body")
		return
	}

	if len(input.Tipps) == 0 {
		app.apiError(w, http.StatusBadRequest, "tipps array is empty")
		return
	}

	if len(input.Tipps) > maxBatchSize {
		app.apiError(w, http.StatusBadRequest, fmt.Sprintf("too many tipps (max %d per request)", maxBatchSize))
		return
	}

	// Validate all items upfront.
	type validatedTipp struct {
		matchID int
		tippA   int
		tippB   int
	}

	validated := make([]validatedTipp, 0, len(input.Tipps))
	for i, t := range input.Tipps {
		fieldErrors := make(map[string]string)
		prefix := fmt.Sprintf("tipps[%d].", i)

		if t.MatchID == nil || *t.MatchID <= 0 {
			fieldErrors[prefix+"match_id"] = "match_id is required and must be greater than 0"
		}
		if t.TippA == nil {
			fieldErrors[prefix+"tipp_a"] = "tipp_a is required"
		} else if *t.TippA < 0 || *t.TippA > 99 {
			fieldErrors[prefix+"tipp_a"] = "tipp_a must be between 0 and 99"
		}
		if t.TippB == nil {
			fieldErrors[prefix+"tipp_b"] = "tipp_b is required"
		} else if *t.TippB < 0 || *t.TippB > 99 {
			fieldErrors[prefix+"tipp_b"] = "tipp_b must be between 0 and 99"
		}

		if len(fieldErrors) > 0 {
			app.apiValidationError(w, fieldErrors)
			return
		}

		validated = append(validated, validatedTipp{
			matchID: *t.MatchID,
			tippA:   *t.TippA,
			tippB:   *t.TippB,
		})
	}

	// Get active event.
	event, err := app.events.GetActive()
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.apiError(w, http.StatusNotFound, "no active event configured")
			return
		}
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	userID := apiUserID(r)
	now := time.Now()

	// Pre-check all matches before saving.
	type matchCheck struct {
		match      models.Match
		tippExists bool
	}

	checks := make([]matchCheck, 0, len(validated))
	for _, vt := range validated {
		match, err := app.matches.Get(vt.matchID)
		if err != nil {
			if errors.Is(err, models.ErrNoRecord) {
				app.apiError(w, http.StatusNotFound, fmt.Sprintf("match %d not found in active event", vt.matchID))
				return
			}
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if match.EventID != event.ID {
			app.apiError(w, http.StatusNotFound, fmt.Sprintf("match %d not found in active event", vt.matchID))
			return
		}

		acceptsTipps := match.Start.After(now) && match.TeamA != "" && match.TeamB != ""
		if !acceptsTipps {
			app.apiError(w, http.StatusConflict, fmt.Sprintf("match %d is locked (already started or teams unknown)", vt.matchID))
			return
		}

		exists, err := app.tipps.Exists(vt.matchID, userID)
		if err != nil {
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}

		checks = append(checks, matchCheck{match: match, tippExists: exists})
	}

	// All checks passed — save all tipps.
	for _, vt := range validated {
		err := app.tipps.InsertOrUpdate(vt.matchID, userID, vt.tippA, vt.tippB)
		if err != nil {
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}

	// Build response.
	results := make([]apiBatchTippResult, 0, len(validated))
	for i, vt := range validated {
		savedTipp, err := app.tipps.GetByMatchAndUser(vt.matchID, userID)
		if err != nil {
			app.apiError(w, http.StatusInternalServerError, "internal error")
			return
		}

		status := "created"
		if checks[i].tippExists {
			status = "updated"
		}

		results = append(results, apiBatchTippResult{
			MatchID: savedTipp.MatchId,
			TippA:   savedTipp.TippA,
			TippB:   savedTipp.TippB,
			Created: savedTipp.Created.UTC().Format(time.RFC3339),
			Changed: savedTipp.Changed.UTC().Format(time.RFC3339),
			Status:  status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}
