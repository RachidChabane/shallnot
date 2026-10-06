package documentation_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const specFormat = "docs/spec-format.md"

func TestStatementDigestDocumentation(t *testing.T) {
	document := read(t, specFormat)
	vectors := []domain.Requirement{
		{
			Title:     "Percentage discount codes",
			Statement: "WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that percentage.",
		},
		{
			Title:         "Café",
			Statement:     "THE  checkout\tpage\r\nSHALL feel   uncluttered.",
			Status:        domain.StatusNonTestable,
			Justification: "judged in\nreview",
		},
	}

	t.Run("the specification states the canonicalization version the code computes [verifies SN-86~1]", func(t *testing.T) {
		if !strings.Contains(document, "`"+domain.StatementCanonicalization+"`") {
			t.Fatalf("%s does not name the canonicalization %s", specFormat, domain.StatementCanonicalization)
		}
	})

	t.Run("the specification's test vectors are what the code computes [verifies SN-86~1]", func(t *testing.T) {
		for _, vector := range vectors {
			if !strings.Contains(document, "```\n"+vector.CanonicalForm()+"\n```") {
				t.Errorf("%s has no block holding the canonical form %q", specFormat, vector.CanonicalForm())
			}
			if !strings.Contains(document, "`"+vector.StatementDigest()+"`") {
				t.Errorf("%s does not state the digest %s of %q", specFormat, vector.StatementDigest(), vector.CanonicalForm())
			}
		}
	})
}
