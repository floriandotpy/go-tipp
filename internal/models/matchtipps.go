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
	TeamA     string
	TeamB     string
	Start     time.Time
	MatchType string
	ResultA   *int
	ResultB   *int

	AcceptsTipps bool
	Status       string

	// data from tipp
	TippA       *int
	TippB       *int
	TippCreated *time.Time
	TippChanged *time.Time
}

func (mt *MatchTipp) SetTipp(tipp Tipp) {
	mt.TippA = &tipp.TippA
	mt.TippB = &tipp.TippB
	mt.TippCreated = &tipp.Created
	mt.TippChanged = &tipp.Changed
}

type MatchTippModel struct {
	DB         *sql.DB
	MatchModel *MatchModel
	TippModel  *TippModel
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

// todo: move somewhere else
// Define string constants for match statuses
const (
	MatchFuture  = "future"
	MatchLive    = "live"
	MatchDone    = "done"
	MatchPending = "pending"
)

// matchStatus determines the status of a match
func matchStatus(start time.Time, now time.Time, resultA *int, resultB *int) string {
	if now.Before(start) {
		return MatchFuture
	} else if resultA != nil && resultB != nil {
		return MatchDone
	} else if now.Sub(start) >= 120*time.Minute && resultA == nil && resultB == nil {
		return MatchPending
	} else {
		return MatchLive
	}
}

func (m *MatchTippModel) All(userId int) ([]MatchTipp, error) {
	matches, err := m.MatchModel.All()
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
		status := matchStatus(match.Start, now, match.ResultA, match.ResultB)
		mt := MatchTipp{
			MatchId:      match.ID,
			TeamA:        match.TeamA,
			TeamB:        match.TeamB,
			Start:        match.Start,
			MatchType:    match.MatchType,
			ResultA:      match.ResultA,
			ResultB:      match.ResultB,
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
