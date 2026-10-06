package spec_test

import (
	"reflect"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain"
)

func detachedLines(t *testing.T, text string) []int {
	t.Helper()
	var lines []int
	for _, detached := range parseMarkdown(t, text).DetachedExamples {
		lines = append(lines, detached.Location.Line)
	}
	return lines
}

func TestMarkdownDetachedExamples(t *testing.T) {
	t.Run("an example line under its statement is absorbed, not detached [verifies SN-93~1]", func(t *testing.T) {
		text := "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n  - Example: quote(a=1) => 0.00\n  * Example: quote(a=2) => 0.00\n"
		document := parseMarkdown(t, text)
		if got := detachedLines(t, text); len(got) != 0 {
			t.Fatalf("got detached lines %v", got)
		}
		if set := domain.ParseExamples(document.Requirements[0].Statement); len(set.Examples) != 2 {
			t.Fatalf("got %+v", set)
		}
	})

	t.Run("an example under a heading-form statement is absorbed [verifies SN-93~1]", func(t *testing.T) {
		text := "### PARK-2~1: Free stays\n\nTHE SYSTEM SHALL charge 0.00.\n- Example: quote(a=1) => 0.00\n"
		if got := detachedLines(t, text); len(got) != 0 {
			t.Fatalf("got detached lines %v", got)
		}
	})

	t.Run("an example after a blank line is detached [verifies SN-93~1]", func(t *testing.T) {
		text := "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n\n  - Example: quote(a=1) => 0.00\n"
		if got := detachedLines(t, text); !reflect.DeepEqual(got, []int{3}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("an example after a non-testable justification is detached [verifies SN-93~1]", func(t *testing.T) {
		text := "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n  - Non-testable: judged by review\n  - Example: quote(a=1) => 0.00\n"
		if got := detachedLines(t, text); !reflect.DeepEqual(got, []int{3}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("an example after an out-of-scope declaration is detached [verifies SN-93~1]", func(t *testing.T) {
		text := "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n  Out of scope (concurrency): one user.\n  - Example: quote(a=1) => 0.00\n"
		if got := detachedLines(t, text); !reflect.DeepEqual(got, []int{3}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("an example outside any requirement is detached, before the first and after a heading [verifies SN-93~1]", func(t *testing.T) {
		text := "Example: quote(a=1) => 0.00\n\n- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n\n## Notes\n- Example: quote(a=2) => 0.00\n"
		if got := detachedLines(t, text); !reflect.DeepEqual(got, []int{1, 6}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("the detached line keeps its text, trimmed [verifies SN-93~1]", func(t *testing.T) {
		document := parseMarkdown(t, "\n   - Example: quote(a=1) => 0.00  \n")
		want := []domain.DetachedExample{{Location: domain.Location{File: "plan.md", Line: 2}, Text: "- Example: quote(a=1) => 0.00"}}
		if !reflect.DeepEqual(document.DetachedExamples, want) {
			t.Fatalf("got %+v", document.DetachedExamples)
		}
	})

	t.Run("a marker inside a fenced code block or inside prose is not an example line [verifies SN-93~1]", func(t *testing.T) {
		text := "```\n- Example: quote(a=1) => 0.00\n```\n\nSome prose with Example: inside it.\nexample: lowercase\n"
		if got := detachedLines(t, text); len(got) != 0 {
			t.Fatalf("got detached lines %v", got)
		}
	})

	t.Run("a YAML spec has no detached example [verifies SN-93~1]", func(t *testing.T) {
		document := parseYAML(t, "requirements:\n  - id: PARK-2\n    statement: >\n      THE SYSTEM SHALL charge 0.00. - Example: f() => 1\n")
		if len(document.DetachedExamples) != 0 {
			t.Fatalf("got %+v", document.DetachedExamples)
		}
	})
}
