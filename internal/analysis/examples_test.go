package analysis_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis"
	"github.com/RachidChabane/shallnot/internal/domain"
)

func withStatement(id string, line int, statement string) analysis.FocusedRequirement {
	focused := requirement(id, 1, line)
	focused.Requirement.Statement = statement
	return focused
}

func TestExamples(t *testing.T) {
	const statement = "THE SYSTEM SHALL charge 0.00. - Example: quote(a=1) => 0.00 - Example: broken - Example: quote(a=2) => raises Boom"

	t.Run("a row carries the well-formed examples of its statement [verifies SN-92~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), Requirements: []analysis.FocusedRequirement{withStatement("PARK-2", 4, statement)}})
		examples := rowOf(t, result, "PARK-2").Examples
		if len(examples) != 2 || examples[0].Index != 1 || examples[1].Index != 3 || examples[1].Expect.Kind != domain.OutcomeRaises {
			t.Fatalf("got %+v", examples)
		}
	})

	t.Run("a malformed example is a warning at the declaration naming its position and text [verifies SN-92~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), Requirements: []analysis.FocusedRequirement{withStatement("PARK-2", 4, statement)}})
		finding := expectOne(t, result, domain.CategoryMalformedExample)
		if finding.Severity != domain.SeverityWarning || finding.Blocking || finding.RequirementID != "PARK-2" ||
			finding.Location != (domain.Location{File: specFile, Line: 4}) ||
			!strings.Contains(finding.Message, "example 2") || !strings.Contains(finding.Message, "broken") {
			t.Fatalf("got %+v", finding)
		}
	})

	t.Run("a statement without examples raises nothing and has no examples [verifies SN-92~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)}})
		if len(findingsOf(result, domain.CategoryMalformedExample)) != 0 || len(rowOf(t, result, "REQ-1").Examples) != 0 {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("a repeated ID is not read for examples twice [verifies SN-92~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), Requirements: []analysis.FocusedRequirement{
			withStatement("PARK-2", 4, "THE SYSTEM SHALL work. Example: broken"),
			withStatement("PARK-2", 9, "THE SYSTEM SHALL work. Example: also broken"),
		}})
		expectOne(t, result, domain.CategoryMalformedExample)
	})

	t.Run("the severity of malformed_example is configurable [verifies SN-92~1]", func(t *testing.T) {
		policy := defaultPolicy()
		policy.Severities[domain.CategoryMalformedExample] = domain.SeverityOff
		result := analysis.Analyze(analysis.Input{Policy: policy, Requirements: []analysis.FocusedRequirement{withStatement("PARK-2", 4, statement)}})
		if len(findingsOf(result, domain.CategoryMalformedExample)) != 0 {
			t.Fatalf("got %+v", result.Findings)
		}
	})
}
