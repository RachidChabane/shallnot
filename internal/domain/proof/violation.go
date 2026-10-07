package proof

import (
	"fmt"
	"slices"
	"strings"
)

// ViolationCode names one broken invariant of the format. The codes are shared
// with the writers of proofs: golden vectors state the codes a statement must raise.
type ViolationCode string

// Violation codes.
const (
	CodeStatementType           ViolationCode = "statement_type"
	CodePredicateType           ViolationCode = "predicate_type"
	CodeSubject                 ViolationCode = "subject"
	CodeRequirement             ViolationCode = "requirement"
	CodeMissingField            ViolationCode = "missing_field"
	CodeUnknownValue            ViolationCode = "unknown_value"
	CodeBadDigest               ViolationCode = "bad_digest"
	CodeCacheKeyMismatch        ViolationCode = "cache_key_mismatch"
	CodeUnsorted                ViolationCode = "unsorted"
	CodeJudgedWithoutConfidence ViolationCode = "judged_without_confidence"
	CodeExecutedWithConfidence  ViolationCode = "executed_with_confidence"
	CodeConfidenceRange         ViolationCode = "confidence_range"
	CodeWrongClass              ViolationCode = "wrong_class"
	CodeSentinelNotEquivalent   ViolationCode = "sentinel_not_equivalent"
	CodeNotBlind                ViolationCode = "not_blind"
	CodeNotAdvisory             ViolationCode = "not_advisory"

	CodeUncorroboratedClaim       ViolationCode = "uncorroborated_claim"
	CodeExecutedWithCorroboration ViolationCode = "executed_with_corroboration"
	CodeExecutedWithExecutor      ViolationCode = "executed_with_executor"
	CodeInconsistentAttribution   ViolationCode = "inconsistent_attribution"
	CodeInputDescriptor           ViolationCode = "input_descriptor"
)

// Violation is one broken invariant at a JSON path of the statement.
type Violation struct {
	Code    ViolationCode
	Path    string
	Message string
}

// String renders the violation on one line.
func (v Violation) String() string {
	return fmt.Sprintf("%s at %s: %s", v.Code, v.Path, v.Message)
}

// Codes returns the distinct violation codes in ascending order.
func Codes(violations []Violation) []ViolationCode {
	seen := map[ViolationCode]bool{}
	codes := []ViolationCode{}
	for _, violation := range violations {
		if !seen[violation.Code] {
			seen[violation.Code] = true
			codes = append(codes, violation.Code)
		}
	}
	slices.Sort(codes)
	return codes
}

// Describe joins the violations, one per line.
func Describe(violations []Violation) string {
	lines := make([]string, len(violations))
	for index, violation := range violations {
		lines[index] = violation.String()
	}
	return strings.Join(lines, "\n")
}

// collector gathers violations while a statement is checked.
type collector struct {
	violations []Violation
}

func (c *collector) add(code ViolationCode, path, format string, args ...any) {
	c.violations = append(c.violations, Violation{Code: code, Path: path, Message: fmt.Sprintf(format, args...)})
}
