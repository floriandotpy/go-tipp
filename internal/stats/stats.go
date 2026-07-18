// Package stats computes derived "wrapped" statistics for the end-of-tournament
// recap page. All functions are pure: they take plain input rows (matches and
// tipps) and return computed results, which keeps them cheap to unit test and
// free of database concerns. Callers are responsible for loading the rows and
// mapping them into the input types defined here.
package stats

import (
	"fmt"
	"sort"
	"time"

	"tipp.casualcoding.com/internal/scoring"
)

// MatchInfo is the minimal finished-match data needed to compute stats.
type MatchInfo struct {
	ID        int
	Start     time.Time
	ResultA   int
	ResultB   int
	PhaseType string // scoring.PhaseGroup or scoring.PhaseKO
}

// TotalGoals returns the combined goal count of the match result.
func (m MatchInfo) TotalGoals() int { return m.ResultA + m.ResultB }

// TippInfo is the minimal tipp data needed to compute stats.
type TippInfo struct {
	MatchID         int
	TippA           int
	TippB           int
	Points          int
	ResultCorrect   bool // exact result hit ("Volltreffer")
	TendencyCorrect bool // correct winner/draw tendency
}

// Scoreline returns the tipp formatted as "A:B".
func (t TippInfo) Scoreline() string { return fmt.Sprintf("%d:%d", t.TippA, t.TippB) }

// TotalGoals returns the combined goal count the user predicted.
func (t TippInfo) TotalGoals() int { return t.TippA + t.TippB }

// GoalDistBin is one bucket of the goal-distribution comparison. Predicted holds
// how many of the user's tipps fell in this bucket; Actual holds how many of the
// same matches actually finished in this bucket.
type GoalDistBin struct {
	Label     string
	Predicted int
	Actual    int
}

// Personal is the computed "Dein Turnier" recap for a single user in one event.
type Personal struct {
	TotalPoints   int
	TotalTipps    int
	MatchesTotal  int // finished matches in the event
	MatchesTipped int

	ExactHits    int     // number of exact-result guesses
	ExactHitRate float64 // 0..1, share of tipps that were exact hits

	LongestStreak int // longest run of consecutive scoring tipps (by kickoff order)

	AvgPointsPerTipp float64 // total points divided by number of tipps

	GroupAvg     float64 // avg points per tipp in the group phase
	KoAvg        float64 // avg points per tipp in the KO phase
	GroupShare   float64 // group avg as a fraction (0..1) of the max points per tipp
	KoShare      float64 // KO avg as a fraction (0..1) of the max points per tipp
	PhaseVerdict string  // human-readable verdict, e.g. "KO-Spezialist"

	BestMatchPoints   int    // most points scored on a single tipp
	FavoriteScoreline string // most frequently submitted scoreline, e.g. "2:1"

	GoalDist []GoalDistBin // actual vs predicted goal distribution over tipped matches
}

// goalBinLabels defines the fixed buckets used for goal-distribution charts.
// The final bucket is a catch-all for high-scoring matches.
var goalBinLabels = []string{"0", "1", "2", "3", "4", "5", "6+"}

// goalBinIndex maps a total goal count to its bucket index.
func goalBinIndex(totalGoals int) int {
	if totalGoals < 0 {
		return 0
	}
	if totalGoals >= len(goalBinLabels)-1 {
		return len(goalBinLabels) - 1
	}
	return totalGoals
}

// ComputePersonal builds the personal recap for one user. matches must contain
// every finished match in the event; tipps must contain only that user's tipps.
// Tipps that do not correspond to a match in matches are ignored, so the caller
// may pass a user's full tipp history without pre-filtering by event.
func ComputePersonal(matches []MatchInfo, tipps []TippInfo) Personal {
	matchByID := make(map[int]MatchInfo, len(matches))
	for _, m := range matches {
		matchByID[m.ID] = m
	}

	// Keep only tipps that belong to a finished match in this event.
	scoped := make([]TippInfo, 0, len(tipps))
	for _, t := range tipps {
		if _, ok := matchByID[t.MatchID]; ok {
			scoped = append(scoped, t)
		}
	}

	p := Personal{
		MatchesTotal:  len(matches),
		MatchesTipped: len(scoped),
		TotalTipps:    len(scoped),
	}

	if len(scoped) == 0 {
		p.GoalDist = emptyGoalDist()
		p.PhaseVerdict = "Keine Tipps abgegeben"
		return p
	}

	// Aggregate counters.
	scorelineCounts := make(map[string]int)
	var groupPoints, koPoints int
	var groupTipps, koTipps int

	dist := emptyGoalDist()

	for _, t := range scoped {
		m := matchByID[t.MatchID]

		p.TotalPoints += t.Points
		if t.Points > p.BestMatchPoints {
			p.BestMatchPoints = t.Points
		}
		if t.ResultCorrect {
			p.ExactHits++
		}

		scorelineCounts[fmt.Sprintf("%d:%d", t.TippA, t.TippB)]++

		switch m.PhaseType {
		case scoring.PhaseKO:
			koPoints += t.Points
			koTipps++
		default:
			groupPoints += t.Points
			groupTipps++
		}

		dist[goalBinIndex(t.TotalGoals())].Predicted++
		dist[goalBinIndex(m.TotalGoals())].Actual++
	}

	p.ExactHitRate = float64(p.ExactHits) / float64(len(scoped))
	p.GoalDist = dist
	p.FavoriteScoreline = mostCommonScoreline(scorelineCounts)
	p.LongestStreak = longestScoringStreak(scoped, matchByID)

	p.AvgPointsPerTipp = float64(p.TotalPoints) / float64(len(scoped))

	if groupTipps > 0 {
		p.GroupAvg = float64(groupPoints) / float64(groupTipps)
		p.GroupShare = p.GroupAvg / maxPointsPerTipp(scoring.PhaseGroup)
	}
	if koTipps > 0 {
		p.KoAvg = float64(koPoints) / float64(koTipps)
		p.KoShare = p.KoAvg / maxPointsPerTipp(scoring.PhaseKO)
	}
	p.PhaseVerdict = phaseVerdict(p.GroupAvg, groupTipps, p.KoAvg, koTipps)

	return p
}

func emptyGoalDist() []GoalDistBin {
	dist := make([]GoalDistBin, len(goalBinLabels))
	for i, label := range goalBinLabels {
		dist[i].Label = label
	}
	return dist
}

// mostCommonScoreline returns the scoreline with the highest count, breaking ties
// deterministically by the lexicographically smallest scoreline.
func mostCommonScoreline(counts map[string]int) string {
	best := ""
	bestCount := 0
	for scoreline, count := range counts {
		if count > bestCount || (count == bestCount && (best == "" || scoreline < best)) {
			best = scoreline
			bestCount = count
		}
	}
	return best
}

// longestScoringStreak returns the longest run of consecutive tipps (ordered by
// match kickoff, then match ID) that each scored at least one point.
func longestScoringStreak(tipps []TippInfo, matchByID map[int]MatchInfo) int {
	ordered := make([]TippInfo, len(tipps))
	copy(ordered, tipps)
	sort.SliceStable(ordered, func(i, j int) bool {
		mi, mj := matchByID[ordered[i].MatchID], matchByID[ordered[j].MatchID]
		if mi.Start.Equal(mj.Start) {
			return mi.ID < mj.ID
		}
		return mi.Start.Before(mj.Start)
	})

	longest, current := 0, 0
	for _, t := range ordered {
		if t.Points > 0 {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}

// phaseVerdict compares normalized group- vs KO-phase performance and returns a
// short verdict. Scores are normalized against the maximum points obtainable per
// tipp in each phase so the two phases are comparable despite different scales.
func phaseVerdict(groupAvg float64, groupTipps int, koAvg float64, koTipps int) string {
	groupMax := maxPointsPerTipp(scoring.PhaseGroup)
	koMax := maxPointsPerTipp(scoring.PhaseKO)

	switch {
	case groupTipps == 0 && koTipps == 0:
		return "Keine Tipps abgegeben"
	case koTipps == 0:
		return "Gruppenphasen-Tipper"
	case groupTipps == 0:
		return "KO-Tipper"
	}

	groupNorm := groupAvg / groupMax
	koNorm := koAvg / koMax

	const margin = 1.15
	switch {
	case koNorm > groupNorm*margin:
		return "KO-Spezialist"
	case groupNorm > koNorm*margin:
		return "Gruppenphasen-Held"
	default:
		return "Ausgeglichener Tipper"
	}
}

// maxPointsPerTipp returns the highest points obtainable for a single tipp in the
// given phase, used to normalize cross-phase comparisons.
func maxPointsPerTipp(phaseType string) float64 {
	pts, ok := scoring.PhasePointsMap[phaseType]
	if !ok {
		return 1
	}
	max := pts.CorrectResult
	if pts.CorrectTendencyAndDiff > max {
		max = pts.CorrectTendencyAndDiff
	}
	if pts.CorrectTendency > max {
		max = pts.CorrectTendency
	}
	if max == 0 {
		return 1
	}
	return float64(max)
}
