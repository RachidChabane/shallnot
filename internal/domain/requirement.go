package domain

// Location points at a line of a file. Line 0 means the whole file.
type Location struct {
	File string
	Line int
}

// Status says whether a requirement is expected to be verified by tests.
type Status uint8

const (
	StatusActive Status = iota
	StatusNonTestable
)

var statusNames = []string{"active", "non_testable"}

const statusKind = "status"

func (s Status) String() string               { return enumString(statusKind, statusNames, s) }
func (s Status) MarshalText() ([]byte, error) { return enumText(statusKind, statusNames, s) }
func (s *Status) UnmarshalText(text []byte) (err error) {
	*s, err = parseEnum[Status](statusKind, statusNames, string(text))
	return err
}

// StatusNames lists the wire names of every status.
func StatusNames() []string { return append([]string(nil), statusNames...) }

// Requirement is one statement of intent that tests can verify.
type Requirement struct {
	ID            RequirementID
	Revision      Revision
	Title         string
	Statement     string
	Status        Status
	Justification string
	Location      Location
}

// Ref returns the reference a test must cite to verify the requirement.
func (r Requirement) Ref() Ref { return Ref{ID: r.ID, Revision: r.Revision} }

// SpecProblem is a requirement declaration that could not be read.
type SpecProblem struct {
	Location Location
	Message  string
}
