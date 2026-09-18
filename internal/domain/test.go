package domain

// Outcome is what happened to one test in one run.
type Outcome uint8

const (
	OutcomePassed Outcome = iota
	OutcomeFailed
	OutcomeErrored
	OutcomeSkipped
	OutcomeNotRun
)

var outcomeNames = []string{"passed", "failed", "errored", "skipped", "not_run"}

const outcomeKind = "outcome"

func (o Outcome) String() string               { return enumString(outcomeKind, outcomeNames, o) }
func (o Outcome) MarshalText() ([]byte, error) { return enumText(outcomeKind, outcomeNames, o) }
func (o *Outcome) UnmarshalText(text []byte) (err error) {
	*o, err = parseEnum[Outcome](outcomeKind, outcomeNames, string(text))
	return err
}

// OutcomeNames lists the wire names of every outcome.
func OutcomeNames() []string { return append([]string(nil), outcomeNames...) }

// MalformedTag is a binding tag that could not be read.
type MalformedTag struct {
	Raw    string
	Reason string
}

// Tag is the parsed content of one binding tag.
type Tag struct {
	Raw       string
	Refs      []Ref
	Malformed []MalformedTag
}

// TestCase is one test as recorded in a results file, with the requirements it cites there.
type TestCase struct {
	ResultsFile string
	Suite       string
	ClassName   string
	Name        string
	Outcome     Outcome
	Refs        []Ref
	Malformed   []MalformedTag
}

// Identity is the text in which a source tag's label is looked up.
func (t TestCase) Identity() string {
	return t.Suite + " " + t.ClassName + " " + t.Name
}

// SourceTag is a binding tag found in a test source file.
type SourceTag struct {
	Location  Location
	Label     string
	Refs      []Ref
	Malformed []MalformedTag
}
