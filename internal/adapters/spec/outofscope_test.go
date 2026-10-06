package spec_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func parseMarkdown(t *testing.T, text string) spec.Document {
	t.Helper()
	document, err := spec.NewMarkdownParser(testkit.Grammar(t).IDs).Parse(strings.NewReader(text), "plan.md")
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func outOfScope(line int, dimension, reason string) domain.OutOfScope {
	return domain.OutOfScope{Location: domain.Location{File: "plan.md", Line: line}, Dimension: dimension, Reason: reason}
}

func TestMarkdownOutOfScope(t *testing.T) {
	t.Run("reads a declaration under any list marker and emphasis, with its line [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, strings.Join([]string{
			"Out of scope (concurrency): one user at a time.",
			"- Out of scope (permissions): everyone may export.",
			"  * **Out of scope (boundaries):** no limit is stated.",
			"__Out of scope (compatibility)__: current browsers only.",
			"**Out of scope (time_and_locale): UTC only.**",
			"+ OUT OF SCOPE (error_handling): the caller handles errors.",
			"- Out-of-scope ( input_validation ): the form validates.",
		}, "\n"))
		want := []domain.OutOfScope{
			outOfScope(1, "concurrency", "one user at a time."),
			outOfScope(2, "permissions", "everyone may export."),
			outOfScope(3, "boundaries", "no limit is stated."),
			outOfScope(4, "compatibility", "current browsers only."),
			outOfScope(5, "time_and_locale", "UTC only."),
			outOfScope(6, "error_handling", "the caller handles errors."),
			outOfScope(7, "input_validation", "the form validates."),
		}
		if !reflect.DeepEqual(document.OutOfScope, want) || len(document.Problems) != 0 {
			t.Fatalf("got %+v, problems %+v", document.OutOfScope, document.Problems)
		}
	})

	t.Run("a declaration ends the open statement and is never absorbed [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, strings.Join([]string{
			"- **CART-1~1**: WHEN a cart holds items THE SYSTEM SHALL total them",
			"  Out of scope (concurrency): carts have one writer.",
			"  and this line is dropped too.",
			"- **CART-2~1**: THE SYSTEM SHALL round.",
			"",
			"  Out of scope (boundaries): amounts stay small.",
		}, "\n"))
		if len(document.Requirements) != 2 || document.Requirements[0].Statement != "WHEN a cart holds items THE SYSTEM SHALL total them" ||
			document.Requirements[1].Statement != "THE SYSTEM SHALL round." {
			t.Fatalf("got %+v", document.Requirements)
		}
		if len(document.OutOfScope) != 2 || document.OutOfScope[0].Location.Line != 2 {
			t.Fatalf("got %+v", document.OutOfScope)
		}
	})

	t.Run("a declaration ends an open justification [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, strings.Join([]string{
			"- **CART-7~1**: THE page SHALL feel calm.",
			"  - Non-testable: judged in design review",
			"    Out of scope (permissions): public page.",
			"    and not part of the justification.",
		}, "\n"))
		if got := document.Requirements[0]; got.Justification != "judged in design review" || got.Status != domain.StatusNonTestable {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("a declaration outside any requirement is kept [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, "# Plan\n\nOut of scope (concurrency): none.\n\n## Items\n")
		if !reflect.DeepEqual(document.OutOfScope, []domain.OutOfScope{outOfScope(3, "concurrency", "none.")}) {
			t.Fatalf("got %+v", document.OutOfScope)
		}
	})

	t.Run("a declaration inside a fenced code block is ignored [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, "```\nOut of scope (concurrency): quoted.\n```\n")
		if len(document.OutOfScope) != 0 {
			t.Fatalf("got %+v", document.OutOfScope)
		}
	})

	t.Run("a heading, a mention in prose and a line without parentheses declare nothing [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, "## Out of scope (concurrency): none\n\nWe said Out of scope (boundaries): nothing.\nOut of scope: concurrency\n")
		if len(document.OutOfScope) != 0 || len(document.Problems) != 0 {
			t.Fatalf("got %+v, problems %+v", document.OutOfScope, document.Problems)
		}
	})

	t.Run("a declaration without a dimension or without a reason is a problem at its line [verifies SN-95~1]", func(t *testing.T) {
		document := parseMarkdown(t, "Out of scope (): reason.\nOut of scope (concurrency):\nOut of scope (concurrency): **\n")
		if len(document.OutOfScope) != 0 || len(document.Problems) != 3 {
			t.Fatalf("got %+v, problems %+v", document.OutOfScope, document.Problems)
		}
		for index, problem := range document.Problems {
			if problem.Location.Line != index+1 {
				t.Errorf("problem %d is at line %d", index, problem.Location.Line)
			}
		}
	})
}

func parseYAML(t *testing.T, text string) spec.Document {
	t.Helper()
	document, err := spec.YAMLParser{IDs: testkit.Grammar(t).IDs}.Parse(strings.NewReader(text), "plan.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestYAMLOutOfScope(t *testing.T) {
	t.Run("reads the out_of_scope list beside the requirements, with each item's line [verifies SN-96~1]", func(t *testing.T) {
		document := parseYAML(t, strings.Join([]string{
			"requirements:",
			"  - id: CART-1",
			"    statement: THE SYSTEM SHALL total.",
			"out_of_scope:",
			"  - dimension: concurrency",
			"    reason: one writer per cart",
			"  - dimension: boundaries",
			"    reason: >",
			"      amounts stay",
			"      small",
		}, "\n"))
		want := []domain.OutOfScope{
			{Location: domain.Location{File: "plan.yaml", Line: 5}, Dimension: "concurrency", Reason: "one writer per cart"},
			{Location: domain.Location{File: "plan.yaml", Line: 7}, Dimension: "boundaries", Reason: "amounts stay small"},
		}
		if len(document.Requirements) != 1 || !reflect.DeepEqual(document.OutOfScope, want) || len(document.Problems) != 0 {
			t.Fatalf("got %+v, problems %+v", document, document.Problems)
		}
	})

	t.Run("reads a spec that holds out_of_scope only [verifies SN-96~1]", func(t *testing.T) {
		document := parseYAML(t, "out_of_scope:\n  - {dimension: permissions, reason: public}\n")
		if len(document.OutOfScope) != 1 || len(document.Problems) != 0 {
			t.Fatalf("got %+v", document)
		}
	})

	t.Run("reports each unreadable declaration at its line and keeps the others [verifies SN-96~1]", func(t *testing.T) {
		document := parseYAML(t, strings.Join([]string{
			"out_of_scope:",
			"  - dimension: concurrency",
			"  - reason: no dimension",
			"  - dimension: boundaries",
			"    reason: fine",
			"    extra: key",
			"  - dimension: [a, b]",
			"    reason: not a scalar",
			"  - just a string",
			"  - dimension: permissions",
			"    reason: kept",
		}, "\n"))
		wantLines := []int{2, 3, 6, 7, 9}
		if len(document.Problems) != len(wantLines) || len(document.OutOfScope) != 1 || document.OutOfScope[0].Dimension != "permissions" {
			t.Fatalf("got %+v, problems %+v", document.OutOfScope, document.Problems)
		}
		for index, line := range wantLines {
			if document.Problems[index].Location.Line != line {
				t.Errorf("problem %d at line %d, want %d: %s", index, document.Problems[index].Location.Line, line, document.Problems[index].Message)
			}
		}
	})

	t.Run("reports an out_of_scope that is not a list without failing the run [verifies SN-96~1]", func(t *testing.T) {
		document := parseYAML(t, "out_of_scope: nope\n")
		if len(document.Problems) != 1 {
			t.Fatalf("got %+v", document.Problems)
		}
	})

	t.Run("still reports a key that is neither requirements nor out_of_scope [verifies SN-96~1]", func(t *testing.T) {
		if document := parseYAML(t, "requirements: []\nbogus: 1\n"); len(document.Problems) != 1 {
			t.Fatalf("got %+v", document.Problems)
		}
	})
}
