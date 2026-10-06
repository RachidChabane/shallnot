package spec_test

import (
	"math/rand"
	"strings"
	"testing"
	"testing/quick"

	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

const (
	untitledStatement = "WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that percentage."
	titledName        = "Percentage discount codes"
)

func parseOne(t *testing.T, format spec.Format, text string) domain.Requirement {
	t.Helper()
	ids := testkit.Grammar(t).IDs
	var document spec.Document
	var err error
	if format == spec.FormatYAML {
		document, err = spec.YAMLParser{IDs: ids}.Parse(strings.NewReader(text), "spec.yaml")
	} else {
		document, err = spec.NewMarkdownParser(ids).Parse(strings.NewReader(text), "spec.md")
	}
	if err != nil || len(document.Requirements) != 1 || len(document.Problems) != 0 {
		t.Fatalf("parsing %q: %+v, %v", text, document, err)
	}
	return document.Requirements[0]
}

func TestStatementDigestAcrossFormats(t *testing.T) {
	t.Run("a requirement without a title has one digest in Markdown and in both YAML scalar styles [verifies SN-88~1]", func(t *testing.T) {
		forms := map[string]domain.Requirement{
			"markdown list": parseOne(t, spec.FormatMarkdown,
				"- **CART-2~2**: WHEN a customer applies a percentage discount code THE SYSTEM\n  SHALL reduce the total by that percentage.\n"),
			"markdown paragraph": parseOne(t, spec.FormatMarkdown,
				"CART-2~2: WHEN a customer applies a percentage discount\ncode THE SYSTEM SHALL reduce the total by that percentage.\n"),
			"markdown CRLF": parseOne(t, spec.FormatMarkdown,
				"- **CART-2~2**: WHEN a customer applies a percentage discount code THE SYSTEM\r\n  SHALL reduce the total by that percentage.\r\n"),
			"yaml folded": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-2\n    revision: 2\n    statement: >\n      WHEN a customer applies a percentage discount code THE SYSTEM\n      SHALL reduce the total by that percentage.\n"),
			"yaml literal": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-2\n    revision: 2\n    statement: |\n      WHEN a customer applies a percentage discount\n      code THE SYSTEM SHALL reduce the total by that percentage.\n"),
			"yaml plain": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-2\n    revision: 2\n    statement: "+untitledStatement+"\n"),
		}
		assertOneDigest(t, forms)
	})

	t.Run("a requirement with a title has one digest in heading form and in YAML [verifies SN-88~1]", func(t *testing.T) {
		forms := map[string]domain.Requirement{
			"markdown heading": parseOne(t, spec.FormatMarkdown,
				"### CART-2~2: "+titledName+"\n\nWHEN a customer applies a percentage discount code THE SYSTEM\nSHALL reduce the total by that percentage.\n"),
			"yaml folded": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-2\n    revision: 2\n    title: "+titledName+"\n    statement: >\n      WHEN a customer applies a percentage discount code THE SYSTEM\n      SHALL reduce the total by that percentage.\n"),
			"yaml literal": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-2\n    revision: 2\n    title: "+titledName+"\n    statement: |\n      WHEN a customer applies a percentage discount code THE SYSTEM\n      SHALL reduce the total by that percentage.\n"),
		}
		assertOneDigest(t, forms)
	})

	t.Run("a non-testable requirement has one digest in Markdown and in YAML [verifies SN-88~1]", func(t *testing.T) {
		forms := map[string]domain.Requirement{
			"markdown": parseOne(t, spec.FormatMarkdown,
				"- **CART-7~1**: THE checkout page SHALL feel uncluttered.\n  - Non-testable: judged in the quarterly design review;\n    no automated check can stand in for it.\n"),
			"yaml folded": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-7\n    statement: THE checkout page SHALL feel uncluttered.\n    status: non_testable\n    justification: >\n      judged in the quarterly design review;\n      no automated check can stand in for it.\n"),
			"yaml literal": parseOne(t, spec.FormatYAML,
				"requirements:\n  - id: CART-7\n    statement: THE checkout page SHALL feel uncluttered.\n    status: non_testable\n    justification: |\n      judged in the quarterly design review;\n      no automated check can stand in for it.\n"),
		}
		assertOneDigest(t, forms)
	})

	t.Run("a different statement has another digest in every format [verifies SN-88~1]", func(t *testing.T) {
		markdown := parseOne(t, spec.FormatMarkdown, "- **CART-2~2**: "+untitledStatement+"\n")
		edited := parseOne(t, spec.FormatYAML,
			"requirements:\n  - id: CART-2\n    revision: 2\n    statement: "+strings.Replace(untitledStatement, "reduce", "raise", 1)+"\n")
		if markdown.StatementDigest() == edited.StatementDigest() {
			t.Fatal("an edited statement kept the digest")
		}
	})
}

func assertOneDigest(t *testing.T, forms map[string]domain.Requirement) {
	t.Helper()
	digests := map[string][]string{}
	for name, requirement := range forms {
		digest := requirement.StatementDigest()
		digests[digest] = append(digests[digest], name)
	}
	if len(digests) != 1 {
		t.Fatalf("the forms disagree: %v", digests)
	}
}

var propertyWords = []string{"WHEN", "a", "customer", "applies", "THE", "SYSTEM", "SHALL", "reduce", "total", "café", "naïve", "日本語", "x"}

// wrapped lays words out on lines of one to four words.
func wrapped(random *rand.Rand, words []string, indent string) string {
	var lines []string
	for len(words) > 0 {
		take := min(1+random.Intn(4), len(words))
		lines = append(lines, indent+strings.Join(words[:take], " "))
		words = words[take:]
	}
	return strings.Join(lines, "\n")
}

func TestStatementDigestProperty(t *testing.T) {
	t.Run("any wrapping of a statement has one digest in Markdown and in both YAML scalar styles [verifies SN-88~1]", func(t *testing.T) {
		property := func(seed int64) bool {
			random := rand.New(rand.NewSource(seed))
			words := make([]string, 1+random.Intn(15))
			for index := range words {
				words[index] = propertyWords[random.Intn(len(propertyWords))]
			}
			markdown := parseOne(t, spec.FormatMarkdown, "- **CART-2~2**: "+strings.TrimLeft(wrapped(random, words, "  "), " ")+"\n")
			folded := parseOne(t, spec.FormatYAML, "requirements:\n  - id: CART-2\n    revision: 2\n    statement: >\n"+wrapped(random, words, "      ")+"\n")
			literal := parseOne(t, spec.FormatYAML, "requirements:\n  - id: CART-2\n    revision: 2\n    statement: |\n"+wrapped(random, words, "      ")+"\n")
			return markdown.StatementDigest() == folded.StatementDigest() && folded.StatementDigest() == literal.StatementDigest()
		}
		if err := quick.Check(property, nil); err != nil {
			t.Fatal(err)
		}
	})
}
