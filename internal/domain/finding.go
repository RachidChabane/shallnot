package domain

// Severity ranks a finding. SeverityOff silences its category.
type Severity uint8

const (
	SeverityOff Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

var severityNames = []string{"off", "info", "warning", "error"}

const severityKind = "severity"

func (s Severity) String() string               { return enumString(severityKind, severityNames, s) }
func (s Severity) MarshalText() ([]byte, error) { return enumText(severityKind, severityNames, s) }
func (s *Severity) UnmarshalText(text []byte) (err error) {
	*s, err = parseEnum[Severity](severityKind, severityNames, string(text))
	return err
}

// ParseSeverity resolves a severity wire name.
func ParseSeverity(text string) (Severity, error) {
	return parseEnum[Severity](severityKind, severityNames, text)
}

// SeverityNames lists the wire names of every severity.
func SeverityNames() []string { return append([]string(nil), severityNames...) }

// Category classifies a finding.
type Category uint8

const (
	CategoryUncoveredRequirement Category = iota
	CategoryFailedRequirement
	CategorySkippedRequirement
	CategoryNotRunRequirement
	CategoryFailingBoundTest
	CategoryTagNotInResults
	CategoryOrphanTag
	CategoryRevisionMismatch
	CategoryMalformedTag
	CategoryUnjustifiedNonTestable
	CategoryBoundNonTestable
	CategoryDuplicateID
	CategoryMalformedRequirement
	CategoryUntaggedTest
)

var categoryNames = []string{
	"uncovered_requirement",
	"failed_requirement",
	"skipped_requirement",
	"not_run_requirement",
	"failing_bound_test",
	"tag_not_in_results",
	"orphan_tag",
	"revision_mismatch",
	"malformed_tag",
	"unjustified_non_testable",
	"bound_non_testable",
	"duplicate_id",
	"malformed_requirement",
	"untagged_test",
}

var categoryDefaultSeverities = map[Category]Severity{
	CategoryUncoveredRequirement:   SeverityError,
	CategoryFailedRequirement:      SeverityError,
	CategorySkippedRequirement:     SeverityError,
	CategoryNotRunRequirement:      SeverityError,
	CategoryFailingBoundTest:       SeverityError,
	CategoryTagNotInResults:        SeverityWarning,
	CategoryOrphanTag:              SeverityError,
	CategoryRevisionMismatch:       SeverityError,
	CategoryMalformedTag:           SeverityError,
	CategoryUnjustifiedNonTestable: SeverityError,
	CategoryBoundNonTestable:       SeverityWarning,
	CategoryDuplicateID:            SeverityError,
	CategoryMalformedRequirement:   SeverityError,
	CategoryUntaggedTest:           SeverityInfo,
}

const categoryKind = "category"

func (c Category) String() string               { return enumString(categoryKind, categoryNames, c) }
func (c Category) MarshalText() ([]byte, error) { return enumText(categoryKind, categoryNames, c) }
func (c *Category) UnmarshalText(text []byte) (err error) {
	*c, err = parseEnum[Category](categoryKind, categoryNames, string(text))
	return err
}

// ParseCategory resolves a category wire name.
func ParseCategory(text string) (Category, error) {
	return parseEnum[Category](categoryKind, categoryNames, text)
}

// CategoryNames lists the wire names of every category.
func CategoryNames() []string { return append([]string(nil), categoryNames...) }

// Categories lists every category in declaration order.
func Categories() []Category {
	all := make([]Category, len(categoryNames))
	for index := range categoryNames {
		all[index] = Category(index)
	}
	return all
}

// DefaultSeverity is the severity of a category that the policy does not override.
func (c Category) DefaultSeverity() Severity { return categoryDefaultSeverities[c] }

// TestIdentity names a test in a results file.
type TestIdentity struct {
	ResultsFile string
	ClassName   string
	Name        string
}

// Finding is one classified observation of a run.
type Finding struct {
	Category      Category
	Severity      Severity
	Blocking      bool
	Message       string
	RequirementID RequirementID
	Location      Location
	Test          *TestIdentity
}
