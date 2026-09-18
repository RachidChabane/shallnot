package analysis_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

const (
	specFile    = "spec.md"
	resultsFile = "junit.xml"
	sourceFile  = "cart_test.py"
)

func defaultPolicy() domain.Policy {
	return domain.Policy{Severities: map[domain.Category]domain.Severity{}, FailOn: domain.DefaultFailOn}
}

func requirement(id string, revision, line int) analysis.FocusedRequirement {
	return analysis.FocusedRequirement{
		InFocus: true,
		Requirement: domain.Requirement{
			ID: domain.RequirementID(id), Revision: domain.Revision(revision), Statement: "THE SYSTEM SHALL work.",
			Location: domain.Location{File: specFile, Line: line},
		},
	}
}

func testCase(name string, outcome domain.Outcome, refs ...domain.Ref) domain.TestCase {
	return domain.TestCase{ResultsFile: resultsFile, ClassName: "tests.cart", Name: name, Outcome: outcome, Refs: refs}
}

func sourceTag(line int, label string, refs ...domain.Ref) domain.SourceTag {
	return domain.SourceTag{Location: domain.Location{File: sourceFile, Line: line}, Label: label, Refs: refs}
}

func rowOf(t *testing.T, result domain.Analysis, id string) domain.Row {
	t.Helper()
	for _, row := range result.Rows {
		if row.Requirement.ID == domain.RequirementID(id) {
			return row
		}
	}
	t.Fatalf("no row for %s", id)
	return domain.Row{}
}

func findingsOf(result domain.Analysis, category domain.Category) []domain.Finding {
	var found []domain.Finding
	for _, finding := range result.Findings {
		if finding.Category == category {
			found = append(found, finding)
		}
	}
	return found
}

func expectOne(t *testing.T, result domain.Analysis, category domain.Category) domain.Finding {
	t.Helper()
	found := findingsOf(result, category)
	if len(found) != 1 {
		t.Fatalf("got %d %s findings, want 1 (all findings: %+v)", len(found), category, result.Findings)
	}
	return found[0]
}

func TestCoverage(t *testing.T) {
	ref := testkit.Ref("REQ-1", 1)

	t.Run("a requirement is covered when one bound test passed [verifies SN-30~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)},
			TestCases:    []domain.TestCase{testCase("test_a", domain.OutcomeSkipped, ref), testCase("test_b", domain.OutcomePassed, ref)},
		})
		if row := rowOf(t, result, "REQ-1"); row.Coverage != domain.CoverageCovered || len(row.Tests) != 2 {
			t.Fatalf("got %+v", row)
		}
		if result.Verdict != domain.VerdictPass {
			t.Fatalf("verdict %s, findings %+v", result.Verdict, result.Findings)
		}
	})

	gaps := []struct {
		name     string
		tests    []domain.TestCase
		tags     []domain.SourceTag
		coverage domain.Coverage
		category domain.Category
	}{
		{"failed", []domain.TestCase{testCase("test_a", domain.OutcomeFailed, ref), testCase("test_b", domain.OutcomeSkipped, ref)}, nil, domain.CoverageFailed, domain.CategoryFailedRequirement},
		{"errored", []domain.TestCase{testCase("test_a", domain.OutcomeErrored, ref)}, nil, domain.CoverageFailed, domain.CategoryFailedRequirement},
		{"skipped", []domain.TestCase{testCase("test_a", domain.OutcomeSkipped, ref)}, nil, domain.CoverageSkipped, domain.CategorySkippedRequirement},
		{"absent from results", nil, []domain.SourceTag{sourceTag(4, "test_a", ref)}, domain.CoverageNotRun, domain.CategoryNotRunRequirement},
		{"bound to nothing", nil, nil, domain.CoverageUncovered, domain.CategoryUncoveredRequirement},
	}
	for _, gap := range gaps {
		t.Run("a requirement whose tests are "+gap.name+" gets its own state [verifies SN-31~1]", func(t *testing.T) {
			result := analysis.Analyze(analysis.Input{
				Policy:       defaultPolicy(),
				Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)},
				TestCases:    gap.tests,
				SourceTags:   gap.tags,
			})
			if row := rowOf(t, result, "REQ-1"); row.Coverage != gap.coverage {
				t.Fatalf("coverage %s, want %s", row.Coverage, gap.coverage)
			}
			finding := expectOne(t, result, gap.category)
			if !finding.Blocking || finding.Location.Line != 3 || result.Verdict != domain.VerdictFail {
				t.Fatalf("got %+v, verdict %s", finding, result.Verdict)
			}
		})
	}

	t.Run("a failing test beside a passing one is reported [verifies SN-37~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)},
			TestCases:    []domain.TestCase{testCase("test_a[1]", domain.OutcomePassed, ref), testCase("test_a[2]", domain.OutcomeFailed, ref)},
		})
		if row := rowOf(t, result, "REQ-1"); row.Coverage != domain.CoverageCovered {
			t.Fatalf("coverage %s", row.Coverage)
		}
		if finding := expectOne(t, result, domain.CategoryFailingBoundTest); finding.Test == nil || finding.Test.Name != "test_a[2]" {
			t.Fatalf("got %+v", finding)
		}
	})
}

func TestTagDefects(t *testing.T) {
	t.Run("a tag citing an unknown requirement is an orphan, located at its source line [verifies SN-32~1]", func(t *testing.T) {
		orphan := testkit.Ref("REQ-99", 1)
		result := analysis.Analyze(analysis.Input{
			Policy:     defaultPolicy(),
			TestCases:  []domain.TestCase{testCase("test_a", domain.OutcomePassed, orphan)},
			SourceTags: []domain.SourceTag{sourceTag(12, "test_a", orphan)},
		})
		finding := expectOne(t, result, domain.CategoryOrphanTag)
		if finding.Location != (domain.Location{File: sourceFile, Line: 12}) || finding.RequirementID != "REQ-99" {
			t.Fatalf("got %+v", finding)
		}
	})

	t.Run("an orphan seen only in results is located at the results file [verifies SN-32~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:    defaultPolicy(),
			TestCases: []domain.TestCase{testCase("test_a", domain.OutcomePassed, testkit.Ref("REQ-99", 1))},
		})
		if finding := expectOne(t, result, domain.CategoryOrphanTag); finding.Location.File != resultsFile || finding.Test == nil {
			t.Fatalf("got %+v", finding)
		}
	})

	for _, cited := range []int{1, 3} {
		t.Run("a tag citing another revision does not cover the requirement [verifies SN-33~1]", func(t *testing.T) {
			result := analysis.Analyze(analysis.Input{
				Policy:       defaultPolicy(),
				Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 2, 3)},
				TestCases:    []domain.TestCase{testCase("test_a", domain.OutcomePassed, testkit.Ref("REQ-1", cited))},
			})
			expectOne(t, result, domain.CategoryRevisionMismatch)
			if row := rowOf(t, result, "REQ-1"); row.Coverage != domain.CoverageUncovered {
				t.Fatalf("coverage %s, want uncovered", row.Coverage)
			}
		})
	}

	t.Run("a malformed tag is reported once, at its source line when known [verifies SN-11~1]", func(t *testing.T) {
		malformed := domain.MalformedTag{Raw: "raw", Reason: "cites no revision"}
		withTag := testCase("test_a", domain.OutcomePassed)
		withTag.Malformed = []domain.MalformedTag{malformed}
		source := sourceTag(7, "test_a")
		source.Malformed = []domain.MalformedTag{malformed}
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), TestCases: []domain.TestCase{withTag}, SourceTags: []domain.SourceTag{source}})
		if finding := expectOne(t, result, domain.CategoryMalformedTag); finding.Location.Line != 7 {
			t.Fatalf("got %+v", finding)
		}
		if len(findingsOf(result, domain.CategoryUntaggedTest)) != 0 {
			t.Fatal("a test with a malformed tag was counted as untagged")
		}
	})

	t.Run("a tagged test absent from results is reported even when the requirement is covered [verifies SN-38~1]", func(t *testing.T) {
		ref := testkit.Ref("REQ-1", 1)
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)},
			TestCases:    []domain.TestCase{testCase("test_ran", domain.OutcomePassed, ref)},
			SourceTags:   []domain.SourceTag{sourceTag(4, "test_ran", ref), sourceTag(9, "test_never_ran", ref)},
		})
		finding := expectOne(t, result, domain.CategoryTagNotInResults)
		if finding.Location.Line != 9 || finding.Blocking {
			t.Fatalf("got %+v", finding)
		}
		row := rowOf(t, result, "REQ-1")
		if row.Coverage != domain.CoverageCovered || len(row.Tests) != 2 {
			t.Fatalf("got %+v", row)
		}
	})

	t.Run("a test is located at the most specific source tag that names it [verifies SN-14~1]", func(t *testing.T) {
		ref := testkit.Ref("REQ-1", 1)
		suite := "discount codes " + testkit.Tag("REQ-1~1")
		run := testCase(suite+" > applies a 10% code", domain.OutcomePassed, ref)
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)},
			TestCases:    []domain.TestCase{run},
			SourceTags:   []domain.SourceTag{sourceTag(5, testkit.Tag("REQ-1~1"), ref), sourceTag(20, suite, ref)},
		})
		row := rowOf(t, result, "REQ-1")
		if len(row.Tests) != 1 || row.Tests[0].Source == nil || row.Tests[0].Source.Line != 20 {
			t.Fatalf("got %+v", row.Tests)
		}
	})
}

func TestSpecDefects(t *testing.T) {
	t.Run("a non-testable requirement without justification is a finding [verifies SN-34~1]", func(t *testing.T) {
		unjustified := requirement("REQ-1", 1, 3)
		unjustified.Requirement.Status = domain.StatusNonTestable
		justified := requirement("REQ-2", 1, 8)
		justified.Requirement.Status = domain.StatusNonTestable
		justified.Requirement.Justification = "Judged by a design review."
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), Requirements: []analysis.FocusedRequirement{unjustified, justified}})
		if finding := expectOne(t, result, domain.CategoryUnjustifiedNonTestable); finding.RequirementID != "REQ-1" {
			t.Fatalf("got %+v", finding)
		}
		if row := rowOf(t, result, "REQ-2"); row.Coverage != domain.CoverageNonTestable {
			t.Fatalf("coverage %s", row.Coverage)
		}
		if len(findingsOf(result, domain.CategoryUncoveredRequirement)) != 0 {
			t.Fatal("a non-testable requirement was reported uncovered")
		}
	})

	t.Run("a test bound to a non-testable requirement is a warning [verifies SN-39~1]", func(t *testing.T) {
		nonTestable := requirement("REQ-1", 1, 3)
		nonTestable.Requirement.Status = domain.StatusNonTestable
		nonTestable.Requirement.Justification = "Judged by a design review."
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{nonTestable},
			TestCases:    []domain.TestCase{testCase("test_a", domain.OutcomePassed, testkit.Ref("REQ-1", 1))},
		})
		if finding := expectOne(t, result, domain.CategoryBoundNonTestable); finding.Severity != domain.SeverityWarning {
			t.Fatalf("got %+v", finding)
		}
	})

	t.Run("a second declaration of an ID is a finding and the first one stands [verifies SN-35~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 2, 30), requirement("REQ-1", 1, 3)},
			TestCases:    []domain.TestCase{testCase("test_a", domain.OutcomePassed, testkit.Ref("REQ-1", 1))},
		})
		if finding := expectOne(t, result, domain.CategoryDuplicateID); finding.Location.Line != 30 {
			t.Fatalf("got %+v", finding)
		}
		if len(result.Rows) != 1 || result.Rows[0].Requirement.Revision != 1 || result.Rows[0].Coverage != domain.CoverageCovered {
			t.Fatalf("got %+v", result.Rows)
		}
	})

	t.Run("an unreadable declaration is a finding [verifies SN-6~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			SpecProblems: []domain.SpecProblem{{Location: domain.Location{File: specFile, Line: 5}, Message: "revision \"x\" is not a positive integer"}},
		})
		if finding := expectOne(t, result, domain.CategoryMalformedRequirement); !finding.Blocking || finding.Location.Line != 5 {
			t.Fatalf("got %+v", finding)
		}
	})
}

func TestUntaggedTests(t *testing.T) {
	input := analysis.Input{Policy: defaultPolicy(), TestCases: []domain.TestCase{testCase("test_helper", domain.OutcomePassed)}}

	t.Run("an untagged test is information by default [verifies SN-36~1]", func(t *testing.T) {
		result := analysis.Analyze(input)
		if finding := expectOne(t, result, domain.CategoryUntaggedTest); finding.Severity != domain.SeverityInfo || finding.Blocking {
			t.Fatalf("got %+v", finding)
		}
		if result.Verdict != domain.VerdictPass || result.Tests.Untagged != 1 {
			t.Fatalf("verdict %s, totals %+v", result.Verdict, result.Tests)
		}
	})

	t.Run("an untagged test blocks once its category is an error [verifies SN-36~1]", func(t *testing.T) {
		strict := input
		strict.Policy.Severities = map[domain.Category]domain.Severity{domain.CategoryUntaggedTest: domain.SeverityError}
		if result := analysis.Analyze(strict); result.Verdict != domain.VerdictFail {
			t.Fatalf("verdict %s", result.Verdict)
		}
	})
}

func TestFocus(t *testing.T) {
	t.Run("coverage is demanded of requirements in focus only, while every known requirement resolves tags [verifies SN-40~1]", func(t *testing.T) {
		earlier := requirement("ABC-100.AC1", 1, 3)
		earlier.InFocus = false
		uncoveredEarlier := requirement("ABC-100.AC2", 1, 5)
		uncoveredEarlier.InFocus = false
		current := requirement("ABC-101.AC1", 1, 3)
		current.Requirement.Location.File = "plans/ABC-101.md"
		result := analysis.Analyze(analysis.Input{
			Policy:       defaultPolicy(),
			Requirements: []analysis.FocusedRequirement{earlier, uncoveredEarlier, current},
			TestCases: []domain.TestCase{
				testCase("test_earlier", domain.OutcomePassed, testkit.Ref("ABC-100.AC1", 1)),
				testCase("test_current", domain.OutcomePassed, testkit.Ref("ABC-101.AC1", 1)),
			},
		})
		if result.Verdict != domain.VerdictPass || len(result.Findings) != 0 {
			t.Fatalf("verdict %s, findings %+v", result.Verdict, result.Findings)
		}
		if row := rowOf(t, result, "ABC-100.AC2"); row.InFocus || row.Coverage != domain.CoverageUncovered {
			t.Fatalf("got %+v", row)
		}
	})
}

func TestPolicy(t *testing.T) {
	input := func(policy domain.Policy) analysis.Input {
		return analysis.Input{Policy: policy, Requirements: []analysis.FocusedRequirement{requirement("REQ-1", 1, 3)}}
	}

	t.Run("a category's severity is configurable and decides blocking [verifies SN-50~1]", func(t *testing.T) {
		policy := defaultPolicy()
		policy.Severities[domain.CategoryUncoveredRequirement] = domain.SeverityWarning
		result := analysis.Analyze(input(policy))
		if finding := expectOne(t, result, domain.CategoryUncoveredRequirement); finding.Blocking || result.Verdict != domain.VerdictPass {
			t.Fatalf("got %+v, verdict %s", finding, result.Verdict)
		}
		policy.FailOn = domain.SeverityWarning
		if result := analysis.Analyze(input(policy)); result.Verdict != domain.VerdictFail {
			t.Fatalf("verdict %s with fail_on warning", result.Verdict)
		}
	})

	t.Run("a category set to off is not reported [verifies SN-50~1]", func(t *testing.T) {
		policy := defaultPolicy()
		policy.Severities[domain.CategoryUncoveredRequirement] = domain.SeverityOff
		if result := analysis.Analyze(input(policy)); len(result.Findings) != 0 || result.Verdict != domain.VerdictPass {
			t.Fatalf("got %+v", result.Findings)
		}
	})

	t.Run("every category has a default severity [verifies SN-50~1]", func(t *testing.T) {
		for _, category := range domain.Categories() {
			if category.DefaultSeverity() == domain.SeverityOff {
				t.Errorf("category %s has no default severity", category)
			}
		}
	})
}
