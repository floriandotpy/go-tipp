package stats

import (
	"fmt"
	"sort"

	"tipp.casualcoding.com/internal/scoring"
)

// Scoreline returns the match result formatted as "A:B".
func (m MatchInfo) Scoreline() string { return fmt.Sprintf("%d:%d", m.ResultA, m.ResultB) }

// eligibilityRatio is the minimum share of an event's matches a user must have
// tipped to be considered for group awards. This keeps drive-by participants
// from "winning" a title on a handful of tipps.
const eligibilityRatio = 0.5

// wildGoalThreshold is the combined goal count at which a predicted scoreline is
// considered a bold ("wildcard") tipp.
const wildGoalThreshold = 5

// UserTipps bundles a group member with their tipps for group computations.
type UserTipps struct {
	UserID   int
	UserName string
	Tipps    []TippInfo
}

// Award is a title handed to a single group member, with the metric that earned
// it and a one-line explanation.
type Award struct {
	Key         string
	Title       string
	Winner      string
	Detail      string
	Explanation string
}

// GroupWrapped is the computed "Deine Tipprunde" recap for one group in one event.
type GroupWrapped struct {
	GroupID   int
	GroupName string

	PlayerCount   int
	EligibleCount int
	TotalTipps    int

	AvgPointsPerPlayer float64

	MostCommonTipp        string
	MostCommonTippCount   int
	MostCommonActual      string
	MostCommonActualCount int

	// GoalDist holds the goal-per-match distribution as percentages (0-100):
	// Predicted is the share of all group tipps in each bucket, Actual the share
	// of matches. Percentages are used because the two series have very different
	// totals (many tipps vs few matches).
	GoalDist []GoalDistBin

	Awards []Award
}

// userAgg holds the per-user figures reused across several awards.
type userAgg struct {
	name         string
	tipped       int
	points       int
	exactHits    int
	tendencyOnly int // correct tendency but not exact result
	wildTipps    int // predicted total goals >= threshold
	distinct     int // number of distinct scorelines used
	favoriteFreq int // count of the most-repeated scoreline
	predGoalsSum int // sum of predicted total goals (for optimist average)
	herdMatches  int // tipps matching the per-match majority scoreline
	eligible     bool
}

// ComputeGroupWrapped builds the group recap. matches must contain every finished
// match in the event; users must contain each group member with their tipps
// (a member's out-of-event tipps are ignored).
func ComputeGroupWrapped(groupID int, groupName string, matches []MatchInfo, users []UserTipps) GroupWrapped {
	matchByID := make(map[int]MatchInfo, len(matches))
	for _, m := range matches {
		matchByID[m.ID] = m
	}
	matchesTotal := len(matches)

	g := GroupWrapped{
		GroupID:     groupID,
		GroupName:   groupName,
		PlayerCount: len(users),
		GoalDist:    emptyGoalDist(),
	}

	// Per-match majority scoreline across the whole group (the "crowd").
	majorityByMatch := computeMajorityScorelines(matchByID, users)

	// Aggregate per user and collect group-wide tallies.
	predGoalBuckets := make([]int, len(goalBinLabels))
	tippScorelineCounts := make(map[string]int)
	aggs := make([]userAgg, 0, len(users))
	var totalPoints int

	for _, u := range users {
		a := userAgg{name: u.UserName}
		scorelineCounts := make(map[string]int)

		for _, t := range u.Tipps {
			if _, ok := matchByID[t.MatchID]; !ok {
				continue
			}
			a.tipped++
			a.points += t.Points
			totalPoints += t.Points
			if t.ResultCorrect {
				a.exactHits++
			} else if t.TendencyCorrect {
				a.tendencyOnly++
			}
			if t.TotalGoals() >= wildGoalThreshold {
				a.wildTipps++
			}
			a.predGoalsSum += t.TotalGoals()

			sl := t.Scoreline()
			scorelineCounts[sl]++
			tippScorelineCounts[sl]++
			predGoalBuckets[goalBinIndex(t.TotalGoals())]++

			if maj, ok := majorityByMatch[t.MatchID]; ok && maj == sl {
				a.herdMatches++
			}
		}

		a.distinct = len(scorelineCounts)
		for _, c := range scorelineCounts {
			if c > a.favoriteFreq {
				a.favoriteFreq = c
			}
		}
		a.eligible = matchesTotal > 0 && float64(a.tipped) >= eligibilityRatio*float64(matchesTotal)

		g.TotalTipps += a.tipped
		if a.eligible {
			g.EligibleCount++
		}
		aggs = append(aggs, a)
	}

	if g.PlayerCount > 0 {
		g.AvgPointsPerPlayer = float64(totalPoints) / float64(g.PlayerCount)
	}

	// Most common predicted vs actual scoreline.
	g.MostCommonTipp, g.MostCommonTippCount = topScoreline(tippScorelineCounts)
	actualCounts := make(map[string]int)
	for _, m := range matches {
		actualCounts[m.Scoreline()]++
	}
	g.MostCommonActual, g.MostCommonActualCount = topScoreline(actualCounts)

	// Goal distribution as percentages of their respective totals.
	actualBuckets := make([]int, len(goalBinLabels))
	for _, m := range matches {
		actualBuckets[goalBinIndex(m.TotalGoals())]++
	}
	g.GoalDist = goalDistPercentages(predGoalBuckets, actualBuckets)

	// Comeback award needs chronological match order, so it's computed here and
	// placed first as the marquee title.
	g.Awards = make([]Award, 0, 8)
	if comeback := ComputeComeback(matches, users, matchesTotal); comeback.Winner != "" {
		g.Awards = append(g.Awards, comeback)
	}
	g.Awards = append(g.Awards, computeAwards(aggs, matchesTotal)...)

	return g
}

// computeMajorityScorelines returns the most-tipped scoreline per match across
// all members (ties broken by the lexicographically smallest scoreline).
func computeMajorityScorelines(matchByID map[int]MatchInfo, users []UserTipps) map[int]string {
	perMatch := make(map[int]map[string]int)
	for _, u := range users {
		for _, t := range u.Tipps {
			if _, ok := matchByID[t.MatchID]; !ok {
				continue
			}
			if perMatch[t.MatchID] == nil {
				perMatch[t.MatchID] = make(map[string]int)
			}
			perMatch[t.MatchID][t.Scoreline()]++
		}
	}
	majority := make(map[int]string, len(perMatch))
	for matchID, counts := range perMatch {
		sl, _ := topScoreline(counts)
		majority[matchID] = sl
	}
	return majority
}

// topScoreline returns the highest-count scoreline and its count, breaking ties
// by the lexicographically smallest scoreline for determinism.
func topScoreline(counts map[string]int) (string, int) {
	best := ""
	bestCount := 0
	for sl, c := range counts {
		if c > bestCount || (c == bestCount && (best == "" || sl < best)) {
			best = sl
			bestCount = c
		}
	}
	return best, bestCount
}

// goalDistPercentages converts raw predicted/actual bucket counts into rounded
// percentages of their respective totals.
func goalDistPercentages(predicted, actual []int) []GoalDistBin {
	predTotal, actualTotal := 0, 0
	for i := range predicted {
		predTotal += predicted[i]
		actualTotal += actual[i]
	}
	dist := emptyGoalDist()
	for i := range dist {
		if predTotal > 0 {
			dist[i].Predicted = int(roundPercent(predicted[i], predTotal))
		}
		if actualTotal > 0 {
			dist[i].Actual = int(roundPercent(actual[i], actualTotal))
		}
	}
	return dist
}

func roundPercent(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

// computeAwards derives all group titles from the per-user aggregates. Only
// eligible users can win; awards with no eligible candidate are omitted.
func computeAwards(aggs []userAgg, matchesTotal int) []Award {
	var awards []Award

	add := func(a Award) {
		if a.Winner != "" {
			awards = append(awards, a)
		}
	}

	add(awardVolltrefferKoenig(aggs))
	add(awardOptimist(aggs))
	add(awardStoiker(aggs))
	add(awardWildcard(aggs))
	add(awardHerdentier(aggs))
	add(awardEinzelgaenger(aggs))
	add(awardPechvogel(aggs))

	return awards
}

// pickBest iterates eligible aggregates and returns the one maximizing score,
// using tiebreak (higher wins) to break ties. Returns nil if no eligible user
// has score > 0 (or, when allowZero, no eligible user at all).
func pickBest(aggs []userAgg, allowZero bool, score func(userAgg) float64, tiebreak func(userAgg) float64) *userAgg {
	var best *userAgg
	var bestScore, bestTie float64
	for i := range aggs {
		a := aggs[i]
		if !a.eligible {
			continue
		}
		s := score(a)
		if !allowZero && s <= 0 {
			continue
		}
		tb := 0.0
		if tiebreak != nil {
			tb = tiebreak(a)
		}
		if best == nil || s > bestScore || (s == bestScore && tb > bestTie) {
			b := aggs[i]
			best = &b
			bestScore = s
			bestTie = tb
		}
	}
	return best
}

func awardVolltrefferKoenig(aggs []userAgg) Award {
	w := pickBest(aggs, false, func(a userAgg) float64 { return float64(a.exactHits) }, nil)
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "volltreffer",
		Title:       "Volltreffer-König",
		Winner:      w.name,
		Detail:      fmt.Sprintf("%s exakt getroffen", plural(w.exactHits, "Ergebnis", "Ergebnisse")),
		Explanation: "Am häufigsten das genaue Endergebnis getippt.",
	}
}

func awardOptimist(aggs []userAgg) Award {
	w := pickBest(aggs, false,
		func(a userAgg) float64 { return avgGoals(a) },
		func(a userAgg) float64 { return float64(a.tipped) })
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "optimist",
		Title:       "Der Optimist",
		Winner:      w.name,
		Detail:      fmt.Sprintf("⌀ %s Tore pro Tipp", oneDecimal(avgGoals(*w))),
		Explanation: "Hat im Schnitt die meisten Tore pro Spiel getippt.",
	}
}

func awardStoiker(aggs []userAgg) Award {
	// Fewest distinct scorelines wins; tie broken by a higher favorite frequency.
	var best *userAgg
	for i := range aggs {
		a := aggs[i]
		if !a.eligible || a.tipped == 0 {
			continue
		}
		if best == nil || a.distinct < best.distinct ||
			(a.distinct == best.distinct && a.favoriteFreq > best.favoriteFreq) {
			b := aggs[i]
			best = &b
		}
	}
	if best == nil {
		return Award{}
	}
	return Award{
		Key:         "stoiker",
		Title:       "Stoischer Tipper",
		Winner:      best.name,
		Detail:      fmt.Sprintf("nur %s", plural(best.distinct, "verschiedenes Ergebnis", "verschiedene Ergebnisse")),
		Explanation: "Tippt am liebsten immer wieder dasselbe.",
	}
}

func awardWildcard(aggs []userAgg) Award {
	w := pickBest(aggs, false,
		func(a userAgg) float64 { return float64(a.wildTipps) },
		func(a userAgg) float64 { return float64(a.distinct) })
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "wildcard",
		Title:       "Wildcard-Tipper",
		Winner:      w.name,
		Detail:      fmt.Sprintf("%s mit 5+ Toren", plural(w.wildTipps, "Tipp", "Tipps")),
		Explanation: "Scheut auch vor Torfestival-Tipps nicht zurück.",
	}
}

func awardHerdentier(aggs []userAgg) Award {
	w := pickBest(aggs, false,
		func(a userAgg) float64 { return herdRate(a) },
		func(a userAgg) float64 { return float64(a.tipped) })
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "herdentier",
		Title:       "Herdentier",
		Winner:      w.name,
		Detail:      fmt.Sprintf("%d%% mit der Mehrheit", int(roundHalf(herdRate(*w)*100))),
		Explanation: "Tippt am häufigsten wie die Mehrheit der Gruppe.",
	}
}

func awardEinzelgaenger(aggs []userAgg) Award {
	// Most contrarian: lowest share of tipps matching the group majority.
	w := pickBest(aggs, true,
		func(a userAgg) float64 { return 1 - herdRate(a) },
		func(a userAgg) float64 { return float64(a.tipped) })
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "einzelgaenger",
		Title:       "Einzelgänger",
		Winner:      w.name,
		Detail:      fmt.Sprintf("nur %d%% mit der Mehrheit", int(roundHalf(herdRate(*w)*100))),
		Explanation: "Tippt am seltensten wie die Mehrheit der Gruppe.",
	}
}

func awardPechvogel(aggs []userAgg) Award {
	w := pickBest(aggs, false, func(a userAgg) float64 { return float64(a.tendencyOnly) }, nil)
	if w == nil {
		return Award{}
	}
	return Award{
		Key:         "pechvogel",
		Title:       "Pechvogel",
		Winner:      w.name,
		Detail:      fmt.Sprintf("%s knapp daneben", plural(w.tendencyOnly, "Mal", "Mal")),
		Explanation: "Oft die richtige Tendenz, aber selten der Volltreffer.",
	}
}

// ComputeComeback derives the "Beste Aufholjagd" award: the eligible user with
// the biggest rank improvement between the end of the group phase and the final
// standing. Returns a zero Award (empty Winner) when there is no group/KO split
// or fewer than two eligible users.
func ComputeComeback(matches []MatchInfo, users []UserTipps, matchesTotal int) Award {
	ordered := make([]MatchInfo, len(matches))
	copy(ordered, matches)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Start.Equal(ordered[j].Start) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Start.Before(ordered[j].Start)
	})

	// Index of the last group-phase match in chronological order.
	lastGroupIdx := -1
	for i, m := range ordered {
		if m.PhaseType != scoring.PhaseKO {
			lastGroupIdx = i
		}
	}
	// Need matches after the group phase for a comeback to be meaningful.
	if lastGroupIdx < 0 || lastGroupIdx >= len(ordered)-1 {
		return Award{}
	}

	matchByID := make(map[int]MatchInfo, len(matches))
	for _, m := range matches {
		matchByID[m.ID] = m
	}

	// Build eligible players with per-match points.
	type player struct {
		name   string
		byID   map[int]int
		tipped int
	}
	var players []player
	for _, u := range users {
		byID := make(map[int]int)
		tipped := 0
		for _, t := range u.Tipps {
			if _, ok := matchByID[t.MatchID]; !ok {
				continue
			}
			byID[t.MatchID] = t.Points
			tipped++
		}
		if matchesTotal > 0 && float64(tipped) >= eligibilityRatio*float64(matchesTotal) {
			players = append(players, player{name: u.UserName, byID: byID, tipped: tipped})
		}
	}
	if len(players) < 2 {
		return Award{}
	}

	cumulativeThrough := func(idx int) map[string]int {
		totals := make(map[string]int, len(players))
		for _, p := range players {
			sum := 0
			for i := 0; i <= idx; i++ {
				sum += p.byID[ordered[i].ID]
			}
			totals[p.name] = sum
		}
		return totals
	}

	groupRanks := ranksFromTotals(cumulativeThrough(lastGroupIdx))
	finalRanks := ranksFromTotals(cumulativeThrough(len(ordered) - 1))

	bestName := ""
	bestImprovement := 0
	var fromPlace, toPlace int
	for _, p := range players {
		improvement := groupRanks[p.name] - finalRanks[p.name]
		if bestName == "" || improvement > bestImprovement {
			bestName = p.name
			bestImprovement = improvement
			fromPlace = groupRanks[p.name]
			toPlace = finalRanks[p.name]
		}
	}
	if bestName == "" || bestImprovement <= 0 {
		return Award{}
	}
	return Award{
		Key:         "aufholjagd",
		Title:       "Beste Aufholjagd",
		Winner:      bestName,
		Detail:      fmt.Sprintf("von Platz %d auf Platz %d", fromPlace, toPlace),
		Explanation: "Größter Sprung nach vorne seit Ende der Gruppenphase.",
	}
}

// ranksFromTotals assigns 1-based ranks from a name->points map. Equal point
// totals share the same rank.
func ranksFromTotals(totals map[string]int) map[string]int {
	type entry struct {
		name   string
		points int
	}
	entries := make([]entry, 0, len(totals))
	for name, pts := range totals {
		entries = append(entries, entry{name, pts})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].points == entries[j].points {
			return entries[i].name < entries[j].name
		}
		return entries[i].points > entries[j].points
	})

	ranks := make(map[string]int, len(entries))
	place, prevPoints := 0, -1
	for i, e := range entries {
		if i == 0 || e.points != prevPoints {
			place = i + 1
			prevPoints = e.points
		}
		ranks[e.name] = place
	}
	return ranks
}

func avgGoals(a userAgg) float64 {
	if a.tipped == 0 {
		return 0
	}
	return float64(a.predGoalsSum) / float64(a.tipped)
}

func herdRate(a userAgg) float64 {
	if a.tipped == 0 {
		return 0
	}
	return float64(a.herdMatches) / float64(a.tipped)
}

func oneDecimal(f float64) string { return fmt.Sprintf("%.1f", f) }

func roundHalf(f float64) float64 {
	if f < 0 {
		return float64(int(f - 0.5))
	}
	return float64(int(f + 0.5))
}

// plural renders "N singular" or "N plural" depending on count.
func plural(count int, singular, pluralForm string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, pluralForm)
}
