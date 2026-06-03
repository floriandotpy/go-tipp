package main

import (
	"encoding/json"
	"errors"
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

// apiPostTipp handles POST /api/v1/tipps
func (app *application) apiPostTipp(w http.ResponseWriter, r *http.Request) {
	// Decode JSON body.
	var input struct {
		MatchID *int `json:"match_id"`
		TippA   *int `json:"tipp_a"`
		TippB   *int `json:"tipp_b"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		app.apiError(w, http.StatusBadRequest, "malformed JSON request body")
		return
	}

	// Validate fields.
	fieldErrors := make(map[string]string)

	if input.MatchID == nil || *input.MatchID <= 0 {
		fieldErrors["match_id"] = "match_id is required and must be greater than 0"
	}

	if input.TippA == nil {
		fieldErrors["tipp_a"] = "tipp_a is required"
	} else if *input.TippA < 0 || *input.TippA > 99 {
		fieldErrors["tipp_a"] = "tipp_a must be between 0 and 99"
	}

	if input.TippB == nil {
		fieldErrors["tipp_b"] = "tipp_b is required"
	} else if *input.TippB < 0 || *input.TippB > 99 {
		fieldErrors["tipp_b"] = "tipp_b must be between 0 and 99"
	}

	if len(fieldErrors) > 0 {
		app.apiValidationError(w, fieldErrors)
		return
	}

	matchID := *input.MatchID
	tippA := *input.TippA
	tippB := *input.TippB

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

	// Fetch match and verify it belongs to the active event.
	match, err := app.matches.Get(matchID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.apiError(w, http.StatusNotFound, "match not found in active event")
			return
		}
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if match.EventID != event.ID {
		app.apiError(w, http.StatusNotFound, "match not found in active event")
		return
	}

	// Check if match accepts tipps.
	acceptsTipps := match.Start.After(time.Now()) && match.TeamA != "" && match.TeamB != ""
	if !acceptsTipps {
		app.apiError(w, http.StatusConflict, "match is locked (already started or teams unknown)")
		return
	}

	// Get user ID from context.
	userID := apiUserID(r)

	// Save the tipp.
	err = app.tipps.InsertOrUpdate(matchID, userID, tippA, tippB)
	if err != nil {
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Fetch the saved tipp to return it.
	savedTipp, err := app.tipps.GetByMatchAndUser(matchID, userID)
	if err != nil {
		app.apiError(w, http.StatusInternalServerError, "internal error")
		return
	}

	response := apiTipp{
		MatchID: savedTipp.MatchId,
		TippA:   savedTipp.TippA,
		TippB:   savedTipp.TippB,
		Created: savedTipp.Created.UTC().Format(time.RFC3339),
		Changed: savedTipp.Changed.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
