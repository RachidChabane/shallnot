package spec_test

import (
	"os"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func load(t *testing.T, path string) spec.Document {
	t.Helper()
	document, err := spec.NewLoader(testkit.Grammar(t).IDs).LoadFile(path, path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	return document
}

func byID(t *testing.T, document spec.Document, id string) domain.Requirement {
	t.Helper()
	for _, requirement := range document.Requirements {
		if requirement.ID == domain.RequirementID(id) {
			return requirement
		}
	}
	t.Fatalf("requirement %s was not declared (got %+v)", id, document.Requirements)
	return domain.Requirement{}
}

func TestMarkdown(t *testing.T) {
	document := load(t, "testdata/plan.md")

	t.Run("reads a list item declaration with its wrapped statement and line [verifies SN-1~1]", func(t *testing.T) {
		got := byID(t, document, "ABC-101.AC1")
		want := `WHEN a customer clicks "Export" THE SYSTEM SHALL download the invoice as a PDF named after the invoice number.`
		if got.Revision != 1 || got.Statement != want || got.Location != (domain.Location{File: "testdata/plan.md", Line: 8}) || got.Status != domain.StatusActive {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("reads a heading declaration with its title and first paragraph [verifies SN-1~1]", func(t *testing.T) {
		got := byID(t, document, "REQ-8")
		if got.Revision != 3 || got.Title != "Audit trail" || got.Statement != "WHEN an invoice is exported THE SYSTEM SHALL record who exported it." {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("reads declarations under any list marker and emphasis [verifies SN-1~1]", func(t *testing.T) {
		if got := byID(t, document, "ABC-101.AC3"); got.Revision != 4 {
			t.Fatalf("got %+v", got)
		}
		if got := byID(t, document, "ABC-101.AC4"); got.Revision != 2 {
			t.Fatalf("got %+v", got)
		}
		byID(t, document, "REQ-9")
	})

	t.Run("does not take a mention inside prose for a declaration [verifies SN-1~1]", func(t *testing.T) {
		for _, requirement := range document.Requirements {
			if requirement.ID == "REQ-7" {
				t.Fatalf("a mention was read as a declaration: %+v", requirement)
			}
		}
	})

	t.Run("gives revision 1 to a declaration that states none [verifies SN-2~1]", func(t *testing.T) {
		if got := byID(t, document, "ABC-101.AC2"); got.Revision != domain.FirstRevision || !strings.HasPrefix(got.Statement, "WHEN the invoice") {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("reads the non-testable marker and its justification [verifies SN-3~1]", func(t *testing.T) {
		got := byID(t, document, "ABC-101.AC3")
		if got.Status != domain.StatusNonTestable || got.Justification != "judged by the brand team at the monthly review." || got.Statement != "THE PDF SHALL look professional." {
			t.Fatalf("got %+v", got)
		}
		if bare := byID(t, document, "ABC-101.AC4"); bare.Status != domain.StatusNonTestable || bare.Justification != "" {
			t.Fatalf("got %+v", bare)
		}
	})

	t.Run("ignores declarations inside fenced code blocks [verifies SN-4~1]", func(t *testing.T) {
		for _, requirement := range document.Requirements {
			if requirement.ID == "ABC-101.AC9" {
				t.Fatal("a fenced declaration was read")
			}
		}
	})

	t.Run("reports a bad revision and a missing statement [verifies SN-6~1]", func(t *testing.T) {
		if len(document.Problems) != 2 || document.Problems[0].Location.Line != 18 || document.Problems[1].Location.Line != 19 {
			t.Fatalf("got %+v", document.Problems)
		}
	})
}

func TestMarkdownCustomIDs(t *testing.T) {
	t.Run("declares requirements under the project's own ID convention [verifies SN-7~1]", func(t *testing.T) {
		parser := spec.NewMarkdownParser(testkit.GrammarFor(t, `story/[0-9]+`).IDs)
		document, err := parser.Parse(strings.NewReader("- story/12~2: THE SYSTEM SHALL work.\n- REQ-1~1: THE SYSTEM SHALL be ignored.\n"), "plan.md")
		if err != nil || len(document.Requirements) != 1 || document.Requirements[0].Ref() != testkit.Ref("story/12", 2) {
			t.Fatalf("got %+v, %v", document, err)
		}
	})
}

func FuzzMarkdown(f *testing.F) {
	seed, err := os.ReadFile("testdata/plan.md")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(seed))
	ids, err := domain.NewIDPattern(domain.DefaultIDPattern)
	if err != nil {
		f.Fatal(err)
	}
	parser := spec.NewMarkdownParser(ids)
	f.Fuzz(func(t *testing.T, text string) {
		document, err := parser.Parse(strings.NewReader(text), "fuzz.md")
		if err != nil {
			return
		}
		for _, requirement := range document.Requirements {
			if !ids.Matches(string(requirement.ID)) || requirement.Revision < domain.FirstRevision || requirement.Statement == "" || requirement.Location.Line < 1 {
				t.Fatalf("invalid requirement %+v", requirement)
			}
		}
	})
}
