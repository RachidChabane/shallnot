package domain

// Coverage is the verification state of one requirement in one run.
type Coverage uint8

const (
	CoverageCovered Coverage = iota
	CoverageFailed
	CoverageSkipped
	CoverageNotRun
	CoverageUncovered
	CoverageNonTestable
)

var coverageNames = []string{"covered", "failed", "skipped", "not_run", "uncovered", "non_testable"}

const coverageKind = "coverage"

func (c Coverage) String() string               { return enumString(coverageKind, coverageNames, c) }
func (c Coverage) MarshalText() ([]byte, error) { return enumText(coverageKind, coverageNames, c) }
func (c *Coverage) UnmarshalText(text []byte) (err error) {
	*c, err = parseEnum[Coverage](coverageKind, coverageNames, string(text))
	return err
}

// CoverageNames lists the wire names of every coverage state.
func CoverageNames() []string { return append([]string(nil), coverageNames...) }

// Coverages lists every coverage state in declaration order.
func Coverages() []Coverage {
	all := make([]Coverage, len(coverageNames))
	for index := range coverageNames {
		all[index] = Coverage(index)
	}
	return all
}

// BoundTest is one test bound to a requirement, with what happened to it.
type BoundTest struct {
	ResultsFile string
	Suite       string
	ClassName   string
	Name        string
	Outcome     Outcome
	Source      *Location
}

// Row is one line of the coverage matrix.
type Row struct {
	Requirement Requirement
	InFocus     bool
	Coverage    Coverage
	Tests       []BoundTest
}

// Verdict is the gate's decision.
type Verdict uint8

const (
	VerdictPass Verdict = iota
	VerdictFail
)

var verdictNames = []string{"pass", "fail"}

const verdictKind = "verdict"

func (v Verdict) String() string               { return enumString(verdictKind, verdictNames, v) }
func (v Verdict) MarshalText() ([]byte, error) { return enumText(verdictKind, verdictNames, v) }
func (v *Verdict) UnmarshalText(text []byte) (err error) {
	*v, err = parseEnum[Verdict](verdictKind, verdictNames, string(text))
	return err
}

// VerdictNames lists the wire names of every verdict.
func VerdictNames() []string { return append([]string(nil), verdictNames...) }

// TestTotals counts the tests seen in the results files.
type TestTotals struct {
	Total    int
	Bound    int
	Untagged int
}

// Analysis is the full outcome of a run.
type Analysis struct {
	Rows     []Row
	Findings []Finding
	Tests    TestTotals
	Verdict  Verdict
}
