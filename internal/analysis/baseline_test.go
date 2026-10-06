package analysis_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func TestBaseline(t *testing.T) {
	current := withStatement("REQ-1", 3, "THE SYSTEM SHALL reject empty carts.")
	digest := current.Requirement.StatementDigest()
	ref := testkit.Ref("REQ-1", 1)

	run := func(policy domain.Policy, baseline map[domain.Ref]string, requirements ...analysis.FocusedRequirement) domain.Analysis {
		return analysis.Analyze(analysis.Input{Policy: policy, Requirements: requirements, Baseline: baseline})
	}

	t.Run("a statement that changed at the same reference is a warning at its declaration [verifies SN-99~1]", func(t *testing.T) {
		result := run(defaultPolicy(), map[domain.Ref]string{ref: strings.Repeat("0", 64)}, current)
		finding := expectOne(t, result, domain.CategoryStatementChanged)
		if finding.Severity != domain.SeverityWarning || finding.Blocking || finding.RequirementID != "REQ-1" ||
			finding.Location != (domain.Location{File: specFile, Line: 3}) || !strings.Contains(finding.Message, "REQ-1~1") ||
			!strings.Contains(finding.Message, digest[:12]) {
			t.Fatalf("got %+v", finding)
		}
	})

	t.Run("an unchanged statement raises nothing [verifies SN-99~1]", func(t *testing.T) {
		if result := run(defaultPolicy(), map[domain.Ref]string{ref: digest}, current); len(findingsOf(result, domain.CategoryStatementChanged)) != 0 {
			t.Fatalf("got %+v", result.Findings)
		}
	})

	t.Run("a reformatted statement raises nothing [verifies SN-99~1]", func(t *testing.T) {
		reformatted := withStatement("REQ-1", 3, "THE SYSTEM  SHALL\nreject empty   carts.")
		if result := run(defaultPolicy(), map[domain.Ref]string{ref: digest}, reformatted); len(findingsOf(result, domain.CategoryStatementChanged)) != 0 {
			t.Fatalf("got %+v", result.Findings)
		}
	})

	t.Run("a requirement that has no entry at its reference in the baseline raises nothing [verifies SN-99~1]", func(t *testing.T) {
		bumped := current
		bumped.Requirement.Revision = 2
		other := map[domain.Ref]string{testkit.Ref("REQ-9", 1): "x"}
		for _, baseline := range []map[domain.Ref]string{nil, {}, other, {ref: "x"}} {
			result := run(defaultPolicy(), baseline, bumped)
			if len(findingsOf(result, domain.CategoryStatementChanged)) != 0 {
				t.Errorf("baseline %v: got %+v", baseline, result.Findings)
			}
		}
	})

	t.Run("a requirement outside the focus is compared too [verifies SN-99~1]", func(t *testing.T) {
		outside := current
		outside.InFocus = false
		result := run(defaultPolicy(), map[domain.Ref]string{ref: "x"}, outside)
		expectOne(t, result, domain.CategoryStatementChanged)
	})

	t.Run("the severity of statement_changed is configurable [verifies SN-99~1]", func(t *testing.T) {
		baseline := map[domain.Ref]string{ref: "x"}
		policy := defaultPolicy()
		policy.Severities[domain.CategoryStatementChanged] = domain.SeverityError
		if result := run(policy, baseline, current); !expectOne(t, result, domain.CategoryStatementChanged).Blocking {
			t.Fatal("not blocking at error")
		}
		policy.Severities[domain.CategoryStatementChanged] = domain.SeverityOff
		if result := run(policy, baseline, current); len(findingsOf(result, domain.CategoryStatementChanged)) != 0 {
			t.Fatalf("got %+v", result.Findings)
		}
	})
}
