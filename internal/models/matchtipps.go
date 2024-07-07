package models

import (
	"database/sql"
	"errors"
	"time"
)

// a MatchTipp instance represents the join between a match and the corresponding user guess (=tipps)
type MatchTipp struct {

	// ids
	MatchId int
	TippId  *int

	// data from match
	TeamA       string
	TeamB       string
	Start       time.Time
	MatchType   string
	ResultA     *int
	ResultB     *int
	ResultAETA  *int
	ResultAETB  *int
	ResultAPenA *int
	ResultAPenB *int

	AcceptsTipps bool
	Status       string

	// data from tipp
	TippA       *int
	TippB       *int
	TippCreated *time.Time
	TippChanged *time.Time
	Points      int
}

// Define string constants for match statuses
const (
	MatchFuture  = "future"
	MatchLive    = "live"
	MatchDone    = "done"
	MatchPending = "pending"
)

func (mt *MatchTipp) SetTipp(tipp Tipp) {
	mt.TippA = &tipp.TippA
	mt.TippB = &tipp.TippB
	mt.TippCreated = &tipp.Created
	mt.TippChanged = &tipp.Changed
	mt.Points = tipp.Points
}

type MatchTippModel struct {
	DB         *sql.DB
	MatchModel *MatchModel
	TippModel  *TippModel
}

func (m *MatchTippModel) MatchStatus(match Match, now time.Time) string {
	if now.Before(match.Start) {
		return MatchFuture
	} else if match.Finished {
		return MatchDone
	} else if !match.Finished {
		return MatchLive
	} else {
		return MatchPending
	}
}

func (m *MatchTippModel) AcceptsTipps(matchId int) (bool, error) {
	if m.MatchModel == nil {
		return false, errors.New("MatchModel is nil")
	}

	accepts, err := m.MatchModel.AcceptsTipps(matchId)
	if err != nil {
		return false, err
	}

	return accepts, nil
}

func (m *MatchTippModel) AllByDaterange(userId int, after time.Time, before time.Time) ([]MatchTipp, error) {
	matches, err := m.MatchModel.AllByDaterange(after, before)
	if err != nil {
		return nil, err
	}

	tipps, err := m.TippModel.AllForUser(userId)
	if err != nil {
		return nil, err
	}

	// create a map for the Tipp list
	tippMap := make(map[int]Tipp)
	for _, tipp := range tipps {
		tippMap[tipp.MatchId] = tipp
	}

	// Perform the join
	var joined []MatchTipp
	now := time.Now()
	for _, match := range matches {
		acceptsTipps, err := m.AcceptsTipps(match.ID)
		if err != nil {
			return nil, err
		}
		status := m.MatchStatus(match, now)
		mt := MatchTipp{
			MatchId:      match.ID,
			TeamA:        match.TeamA,
			TeamB:        match.TeamB,
			Start:        match.Start,
			MatchType:    match.MatchType,
			ResultA:      match.ResultA,
			ResultB:      match.ResultB,
			ResultAETA:   match.ResultAETA,
			ResultAETB:   match.ResultAETB,
			ResultAPenA:  match.ResultAPenA,
			ResultAPenB:  match.ResultAPenB,
			AcceptsTipps: acceptsTipps,
			Status:       status,
		}
		if userTipp, ok := tippMap[match.ID]; ok {
			mt.SetTipp(userTipp)
		}
		joined = append(joined, mt)
	}

	return joined, nil
}
