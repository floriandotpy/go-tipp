package models

import (
	"database/sql"
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
	for _, match := range matches {
		mt := MatchTipp{
			MatchId:   match.ID,
			TeamA:     match.TeamA,
			TeamB:     match.TeamB,
			Start:     match.Start,
			MatchType: match.MatchType,
			ResultA:   match.ResultA,
			ResultB:   match.ResultB,
		}
		if userTipp, ok := tippMap[match.ID]; ok {
			mt.SetTipp(userTipp)
		}
		joined = append(joined, mt)
	}

	return joined, nil
}
