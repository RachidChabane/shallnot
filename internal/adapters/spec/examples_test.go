package spec_test

import (
	"reflect"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/domain"
)

func TestExamplesAcrossFormats(t *testing.T) {
	t.Run("a statement has the same examples in a Markdown list, a folded scalar and a literal scalar [verifies SN-90~1]", func(t *testing.T) {
		forms := map[string]domain.Requirement{
			"markdown": parseOne(t, spec.FormatMarkdown,
				"- **PARK-2~1**: WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00.\n"+
					"  - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:15) => 0.00\n"+
					"  - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:00) => 0.00\n"),
			"yaml folded": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: PARK-2\n    statement: >\n"+
					"      WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00.\n"+
					"      - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:15) => 0.00\n"+
					"      - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:00) => 0.00\n"),
			"yaml literal": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: PARK-2\n    statement: |\n"+
					"      WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00.\n"+
					"      - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:15) => 0.00\n"+
					"      - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:00) => 0.00\n"),
		}
		var reference domain.ExampleSet
		for name, requirement := range forms {
			set := domain.ParseExamples(requirement.Statement)
			if len(set.Examples) != 2 || len(set.Malformed) != 0 {
				t.Fatalf("%s: got %+v", name, set)
			}
			if reference.Examples == nil {
				reference = set
			}
			if !reflect.DeepEqual(set, reference) {
				t.Errorf("%s: got %+v, want %+v", name, set, reference)
			}
		}
	})

	t.Run("examples do not change the statement digest rules: editing one changes the digest [verifies SN-90~1]", func(t *testing.T) {
		original := parseOne(t, spec.FormatMarkdown, "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n  - Example: quote(a=1) => 0.00\n")
		edited := parseOne(t, spec.FormatMarkdown, "- **PARK-2~1**: THE SYSTEM SHALL charge 0.00.\n  - Example: quote(a=2) => 0.00\n")
		if original.StatementDigest() == edited.StatementDigest() {
			t.Fatal("editing an example kept the statement digest")
		}
	})
}
