package spec_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func TestYAML(t *testing.T) {
	document := load(t, "testdata/spec.yaml")

	t.Run("maps a YAML requirement to the same model as Markdown [verifies SN-5~1]", func(t *testing.T) {
		got := byID(t, document, "REQ-1")
		want := domain.Requirement{
			ID: "REQ-1", Revision: 3, Title: "Totals", Status: domain.StatusActive,
			Statement: "WHEN a cart holds line items THE SYSTEM SHALL compute the total.",
			Location:  domain.Location{File: "testdata/spec.yaml", Line: 2},
		}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
		if byID(t, document, "REQ-2").Revision != domain.FirstRevision {
			t.Fatal("revision does not default to 1")
		}
	})

	t.Run("reads the non-testable status with and without justification [verifies SN-3~1]", func(t *testing.T) {
		if got := byID(t, document, "REQ-3"); got.Status != domain.StatusNonTestable || got.Justification != "Judged in design review." {
			t.Fatalf("got %+v", got)
		}
		if got := byID(t, document, "REQ-4"); got.Status != domain.StatusNonTestable || got.Justification != "" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("reports each unreadable requirement at its line [verifies SN-6~1]", func(t *testing.T) {
		wantLines := []int{17, 20, 22, 25, 28, 29}
		if len(document.Problems) != len(wantLines) {
			t.Fatalf("got %d problems, want %d: %+v", len(document.Problems), len(wantLines), document.Problems)
		}
		for index, line := range wantLines {
			if document.Problems[index].Location.Line != line {
				t.Errorf("problem %d at line %d, want %d: %s", index, document.Problems[index].Location.Line, line, document.Problems[index].Message)
			}
		}
		if len(document.Requirements) != 4 {
			t.Fatalf("got %d requirements, want 4", len(document.Requirements))
		}
	})

	t.Run("refuses a document that is not a spec [verifies SN-52~1]", func(t *testing.T) {
		parser := spec.YAMLParser{IDs: testkit.Grammar(t).IDs}
		for _, text := range []string{"- a list", "requirements: nope", "requirements: [unclosed"} {
			if _, err := parser.Parse(strings.NewReader(text), "spec.yaml"); err == nil {
				t.Errorf("%q was accepted", text)
			}
		}
	})
}
