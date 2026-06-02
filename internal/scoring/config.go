package scoring

const (
	PhaseGroup = "phase_group"
	PhaseKO    = "phase_ko"
)

type PhasePoints struct {
	CorrectResult          int
	CorrectTendencyAndDiff int
	CorrectTendency        int
}

var PhasePointsMap = map[string]PhasePoints{
	PhaseGroup: {
		CorrectResult:          3,
		CorrectTendencyAndDiff: 2,
		CorrectTendency:        1,
	},
	PhaseKO: {
		CorrectResult:          6,
		CorrectTendencyAndDiff: 4,
		CorrectTendency:        2,
	},
}
