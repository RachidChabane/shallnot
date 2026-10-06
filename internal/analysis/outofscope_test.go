package analysis_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis"
	"github.com/RachidChabane/shallnot/internal/domain"
)

func declaration(file string, line int, dimension string) domain.OutOfScope {
	return domain.OutOfScope{Location: domain.Location{File: file, Line: line}, Dimension: dimension, Reason: "not covered here"}
}

func TestOutOfScope(t *testing.T) {
	t.Run("a declaration of each known dimension raises nothing [verifies SN-97~1]", func(t *testing.T) {
		var declared []domain.OutOfScope
		for index, dimension := range domain.Dimensions() {
			declared = append(declared, declaration(specFile, index+1, dimension))
		}
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), OutOfScope: declared})
		if len(result.Findings) != 0 {
			t.Fatalf("got %+v", result.Findings)
		}
	})

	t.Run("a dimension outside the closed list is a warning at its line naming the closed list [verifies SN-97~1]", func(t *testing.T) {
		result := analysis.Analyze(analysis.Input{
			Policy:     defaultPolicy(),
			OutOfScope: []domain.OutOfScope{declaration(specFile, 7, "Concurrency"), declaration(specFile, 9, "performance")},
		})
		findings := findingsOf(result, domain.CategoryUnknownDimension)
		if len(findings) != 2 || findings[0].Location.Line != 7 || findings[1].Location.Line != 9 {
			t.Fatalf("got %+v", findings)
		}
		first := findings[0]
		if first.Severity != domain.SeverityWarning || first.Blocking || !strings.Contains(first.Message, `"Concurrency"`) {
			t.Fatalf("got %+v", first)
		}
		for _, dimension := range domain.Dimensions() {
			if !strings.Contains(first.Message, dimension) {
				t.Errorf("the message does not name %s: %s", dimension, first.Message)
			}
		}
	})

	t.Run("the severity of unknown_dimension is configurable [verifies SN-97~1]", func(t *testing.T) {
		declared := []domain.OutOfScope{declaration(specFile, 1, "performance")}
		policy := defaultPolicy()
		policy.Severities[domain.CategoryUnknownDimension] = domain.SeverityError
		if result := analysis.Analyze(analysis.Input{Policy: policy, OutOfScope: declared}); result.Verdict != domain.VerdictFail {
			t.Fatalf("an unknown dimension at error severity did not block: %+v", result.Findings)
		}
		policy.Severities[domain.CategoryUnknownDimension] = domain.SeverityOff
		if result := analysis.Analyze(analysis.Input{Policy: policy, OutOfScope: declared}); len(result.Findings) != 0 {
			t.Fatalf("an unknown dimension switched off was reported: %+v", result.Findings)
		}
	})

	t.Run("the analysis hands the declarations back ordered by file then line [verifies SN-97~1]", func(t *testing.T) {
		declared := []domain.OutOfScope{
			declaration("b.md", 1, domain.DimensionBoundaries),
			declaration("a.md", 9, domain.DimensionConcurrency),
			declaration("a.md", 2, domain.DimensionPermissions),
		}
		result := analysis.Analyze(analysis.Input{Policy: defaultPolicy(), OutOfScope: declared})
		want := []domain.OutOfScope{declared[2], declared[1], declared[0]}
		if !reflect.DeepEqual(result.OutOfScope, want) {
			t.Fatalf("got %+v", result.OutOfScope)
		}
	})
}
