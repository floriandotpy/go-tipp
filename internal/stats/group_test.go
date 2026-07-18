package stats

import (
	"fmt"
	"testing"

	"tipp.casualcoding.com/internal/scoring"
)

// buildMatches creates n group-phase matches plus optional KO matches, all with
// a 1:0 result, spaced hourly.
func makeGroupMatches(n int) []MatchInfo {
	m := make([]MatchInfo, 0, n)
	for i := 0; i < n; i++ {
		m = append(m, matchAt(i+1, i, 1, 0, scoring.PhaseGroup))
	}
	return m
}

func findAward(awards []Award, key string) (Award, bool) {
	for _, a := range awards {
		if a.Key == key {
			return a, true
		}
	}
	return Award{}, false
}

func TestGroupWrapped_Eligibility(t *testing.T) {
	matches := makeGroupMatches(10)

	// Alice tipped all 10 (eligible); Bob tipped only 3 (< 50%, ineligible).
	alice := UserTipps{UserID: 1, UserName: "Alice"}
	for _, m := range matches {
		alice.Tipps = append(alice.Tipps, TippInfo{MatchID: m.ID, TippA: 1, TippB: 0, Points: 3, ResultCorrect: true})
	}
	bob := UserTipps{UserID: 2, UserName: "Bob", Tipps: []TippInfo{
		{MatchID: 1, TippA: 5, TippB: 4, Points: 0},
		{MatchID: 2, TippA: 5, TippB: 4, Points: 0},
		{MatchID: 3, TippA: 5, TippB: 4, Points: 0},
	}}

	g := ComputeGroupWrapped(1, "Testrunde", matches, []UserTipps{alice, bob})

	if g.PlayerCount != 2 {
		t.Errorf("PlayerCount = %d, want 2", g.PlayerCount)
	}
	if g.EligibleCount != 1 {
		t.Errorf("EligibleCount = %d, want 1", g.EligibleCount)
	}
	if g.TotalTipps != 13 {
		t.Errorf("TotalTipps = %d, want 13", g.TotalTipps)
	}
	// Volltreffer-König must be Alice (Bob is ineligible despite... well he has 0).
	if a, ok := findAward(g.Awards, "volltreffer"); !ok || a.Winner != "Alice" {
		t.Errorf("volltreffer winner = %+v, want Alice", a)
	}

	// PlayerStats surfaced for the podium: Alice tipped 10, all exact hits.
	alicePS, ok := g.PlayerStats[1]
	if !ok {
		t.Fatal("PlayerStats missing entry for Alice (id 1)")
	}
	if alicePS.Tipps != 10 || alicePS.ExactHits != 10 {
		t.Errorf("Alice PlayerStats = %+v, want Tipps 10 / ExactHits 10", alicePS)
	}
	if alicePS.HitRate != 1.0 {
		t.Errorf("Alice HitRate = %v, want 1.0", alicePS.HitRate)
	}
	if alicePS.AvgPoints != 3.0 {
		t.Errorf("Alice AvgPoints = %v, want 3.0", alicePS.AvgPoints)
	}
}

func TestGroupWrapped_MostCommonScorelines(t *testing.T) {
	matches := []MatchInfo{
		matchAt(1, 0, 2, 1, scoring.PhaseGroup),
		matchAt(2, 1, 2, 1, scoring.PhaseGroup),
		matchAt(3, 2, 0, 0, scoring.PhaseGroup),
		matchAt(4, 3, 1, 1, scoring.PhaseGroup),
	}
	// Two eligible users; everyone tips "1:0" most often.
	u1 := UserTipps{UserID: 1, UserName: "A", Tipps: []TippInfo{
		{MatchID: 1, TippA: 1, TippB: 0}, {MatchID: 2, TippA: 1, TippB: 0},
		{MatchID: 3, TippA: 1, TippB: 0}, {MatchID: 4, TippA: 2, TippB: 2},
	}}
	u2 := UserTipps{UserID: 2, UserName: "B", Tipps: []TippInfo{
		{MatchID: 1, TippA: 1, TippB: 0}, {MatchID: 2, TippA: 3, TippB: 3},
		{MatchID: 3, TippA: 1, TippB: 0}, {MatchID: 4, TippA: 1, TippB: 0},
	}}

	g := ComputeGroupWrapped(1, "R", matches, []UserTipps{u1, u2})

	if g.MostCommonTipp != "1:0" {
		t.Errorf("MostCommonTipp = %q, want 1:0", g.MostCommonTipp)
	}
	if g.MostCommonTippCount != 6 {
		t.Errorf("MostCommonTippCount = %d, want 6", g.MostCommonTippCount)
	}
	// Actual results: 2:1 appears twice, others once -> most common 2:1.
	if g.MostCommonActual != "2:1" {
		t.Errorf("MostCommonActual = %q, want 2:1", g.MostCommonActual)
	}
	if g.MostCommonActualCount != 2 {
		t.Errorf("MostCommonActualCount = %d, want 2", g.MostCommonActualCount)
	}
}

func TestGroupWrapped_AwardWinners(t *testing.T) {
	matches := makeGroupMatches(10)

	// Optimist/Wildcard: tips lots of goals. Stoiker: always same low score.
	// Pechvogel: correct tendency but never exact.
	optimist := UserTipps{UserID: 1, UserName: "Opti"}
	stoiker := UserTipps{UserID: 2, UserName: "Stoic"}
	pech := UserTipps{UserID: 3, UserName: "Pech"}
	for i, m := range matches {
		// Optimist: varied high-scoring tipps (5-8 goals) -> wild + high average,
		// but many distinct scorelines so he is not "stoic".
		optimist.Tipps = append(optimist.Tipps, TippInfo{MatchID: m.ID, TippA: 4, TippB: 1 + (i % 4)})
		// Stoiker: always the exact same low scoreline.
		stoiker.Tipps = append(stoiker.Tipps, TippInfo{MatchID: m.ID, TippA: 1, TippB: 0, Points: 3, ResultCorrect: true})
		// Pechvogel: right tendency, never the exact result.
		pech.Tipps = append(pech.Tipps, TippInfo{MatchID: m.ID, TippA: 2, TippB: 0, Points: 1, TendencyCorrect: true})
	}

	g := ComputeGroupWrapped(1, "R", matches, []UserTipps{optimist, stoiker, pech})

	if a, ok := findAward(g.Awards, "optimist"); !ok || a.Winner != "Opti" {
		t.Errorf("optimist winner = %+v, want Opti", a)
	}
	if a, ok := findAward(g.Awards, "wildcard"); !ok || a.Winner != "Opti" {
		t.Errorf("wildcard winner = %+v, want Opti", a)
	}
	if a, ok := findAward(g.Awards, "stoiker"); !ok || a.Winner != "Stoic" {
		t.Errorf("stoiker winner = %+v, want Stoic", a)
	}
	if a, ok := findAward(g.Awards, "volltreffer"); !ok || a.Winner != "Stoic" {
		t.Errorf("volltreffer winner = %+v, want Stoic", a)
	}
	if a, ok := findAward(g.Awards, "pechvogel"); !ok || a.Winner != "Pech" {
		t.Errorf("pechvogel winner = %+v, want Pech", a)
	}
}

func TestGroupWrapped_HerdVsLoner(t *testing.T) {
	matches := makeGroupMatches(10)

	// Three "herd" users all tip 1:0 on every match -> majority is 1:0.
	// The loner always tips 3:2, never matching the majority.
	var users []UserTipps
	for u := 1; u <= 3; u++ {
		herd := UserTipps{UserID: u, UserName: fmt.Sprintf("Herd%d", u)}
		for _, m := range matches {
			herd.Tipps = append(herd.Tipps, TippInfo{MatchID: m.ID, TippA: 1, TippB: 0})
		}
		users = append(users, herd)
	}
	loner := UserTipps{UserID: 4, UserName: "Loner"}
	for _, m := range matches {
		loner.Tipps = append(loner.Tipps, TippInfo{MatchID: m.ID, TippA: 3, TippB: 2})
	}
	users = append(users, loner)

	g := ComputeGroupWrapped(1, "R", matches, users)

	if a, ok := findAward(g.Awards, "einzelgaenger"); !ok || a.Winner != "Loner" {
		t.Errorf("einzelgaenger winner = %+v, want Loner", a)
	}
	// A herd member should take the Herdentier title (matches majority every time).
	if a, ok := findAward(g.Awards, "herdentier"); !ok || a.Winner == "Loner" || a.Winner == "" {
		t.Errorf("herdentier winner = %+v, want a herd member", a)
	}
}

func TestComputeComeback(t *testing.T) {
	// 4 group matches then 2 KO matches.
	matches := []MatchInfo{
		matchAt(1, 0, 1, 0, scoring.PhaseGroup),
		matchAt(2, 1, 1, 0, scoring.PhaseGroup),
		matchAt(3, 2, 1, 0, scoring.PhaseGroup),
		matchAt(4, 3, 1, 0, scoring.PhaseGroup),
		matchAt(5, 4, 1, 0, scoring.PhaseKO),
		matchAt(6, 5, 1, 0, scoring.PhaseKO),
	}
	// Leader dominates the group phase, then scores nothing in the KO phase.
	leader := UserTipps{UserID: 1, UserName: "Leader"}
	// Chaser is behind after groups, then wins big in the KO phase.
	chaser := UserTipps{UserID: 2, UserName: "Chaser"}
	for _, m := range matches {
		lp, cp := 0, 0
		if m.PhaseType == scoring.PhaseGroup {
			lp = 3 // leader collects in groups
		} else {
			cp = 6 // chaser collects in KO
		}
		leader.Tipps = append(leader.Tipps, TippInfo{MatchID: m.ID, TippA: 1, TippB: 0, Points: lp})
		chaser.Tipps = append(chaser.Tipps, TippInfo{MatchID: m.ID, TippA: 1, TippB: 0, Points: cp})
	}

	a := ComputeComeback(matches, []UserTipps{leader, chaser}, len(matches))
	if a.Winner != "Chaser" {
		t.Fatalf("comeback winner = %q, want Chaser", a.Winner)
	}
	if a.Key != "aufholjagd" {
		t.Errorf("comeback key = %q, want aufholjagd", a.Key)
	}
}

func TestComputeComeback_NoKOPhase(t *testing.T) {
	matches := makeGroupMatches(6)
	u1 := UserTipps{UserID: 1, UserName: "A"}
	u2 := UserTipps{UserID: 2, UserName: "B"}
	for _, m := range matches {
		u1.Tipps = append(u1.Tipps, TippInfo{MatchID: m.ID, Points: 3})
		u2.Tipps = append(u2.Tipps, TippInfo{MatchID: m.ID, Points: 1})
	}
	a := ComputeComeback(matches, []UserTipps{u1, u2}, len(matches))
	if a.Winner != "" {
		t.Errorf("expected no comeback award without a KO phase, got %+v", a)
	}
}

func TestGroupWrapped_GoalDistPercentages(t *testing.T) {
	// 2 matches, both 1:0 (1 goal each) -> actual 100% in bucket "1".
	matches := []MatchInfo{
		matchAt(1, 0, 1, 0, scoring.PhaseGroup),
		matchAt(2, 1, 1, 0, scoring.PhaseGroup),
	}
	u := UserTipps{UserID: 1, UserName: "A", Tipps: []TippInfo{
		{MatchID: 1, TippA: 1, TippB: 0}, // 1 goal
		{MatchID: 2, TippA: 2, TippB: 1}, // 3 goals
	}}
	g := ComputeGroupWrapped(1, "R", matches, []UserTipps{u})

	byLabel := map[string]GoalDistBin{}
	for _, b := range g.GoalDist {
		byLabel[b.Label] = b
	}
	if byLabel["1"].Actual != 100 {
		t.Errorf("actual %% in bucket 1 = %d, want 100", byLabel["1"].Actual)
	}
	if byLabel["1"].Predicted != 50 {
		t.Errorf("predicted %% in bucket 1 = %d, want 50", byLabel["1"].Predicted)
	}
	if byLabel["3"].Predicted != 50 {
		t.Errorf("predicted %% in bucket 3 = %d, want 50", byLabel["3"].Predicted)
	}
}
