package domain_test

import (
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/RachidChabane/shallnot/internal/domain"
)

// canonicalWhitespace lists the code points c1 collapses, as the specification states them.
var canonicalWhitespace = []rune{
	0x0009, 0x000A, 0x000B, 0x000C, 0x000D, 0x0020, 0x0085, 0x00A0, 0x1680,
	0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200A,
	0x2028, 0x2029, 0x202F, 0x205F, 0x3000,
}

var sampleWords = []string{
	"WHEN", "a", "customer", "applies", "THE", "SYSTEM", "SHALL", "reduce", "total",
	"café", "naïve", "Ångström", "한국어", "日本語", "é", "x",
}

func requirementOf(title, statement string) domain.Requirement {
	return domain.Requirement{ID: "REQ-1", Revision: 1, Title: title, Statement: statement}
}

func randomText(random *rand.Rand, separators []string) (text, plain string) {
	count := 1 + random.Intn(12)
	var written, canonical []string
	for range count {
		word := sampleWords[random.Intn(len(sampleWords))]
		written = append(written, word)
		canonical = append(canonical, norm.NFC.String(word))
	}
	var builder strings.Builder
	builder.WriteString(separators[random.Intn(len(separators))])
	for index, word := range written {
		if index > 0 {
			builder.WriteString(separators[random.Intn(len(separators))])
		}
		if random.Intn(2) == 0 {
			word = norm.NFD.String(word)
		}
		builder.WriteString(word)
	}
	builder.WriteString(separators[random.Intn(len(separators))])
	return builder.String(), strings.Join(canonical, " ")
}

func spaceSeparators() []string {
	separators := []string{"\r\n", "\r", "\n", "\n\n  ", "\t"}
	for _, character := range canonicalWhitespace {
		separators = append(separators, string(character), string(character)+string(character))
	}
	return separators
}

func TestStatementDigest(t *testing.T) {
	t.Run("whitespace is exactly the code points the specification lists [verifies SN-86~1]", func(t *testing.T) {
		listed := map[rune]bool{}
		for _, character := range canonicalWhitespace {
			listed[character] = true
		}
		for character := rune(0); character <= unicode.MaxRune; character++ {
			if unicode.IsSpace(character) != listed[character] {
				t.Fatalf("U+%04X: collapsed %v, listed %v", character, unicode.IsSpace(character), listed[character])
			}
		}
	})

	t.Run("harmless reformatting keeps the digest [verifies SN-86~1]", func(t *testing.T) {
		base := requirementOf("Percentage discount codes",
			"WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that percentage.")
		variants := map[string]string{
			"line wrapping":     "WHEN a customer applies a percentage\ndiscount code THE SYSTEM SHALL reduce\nthe total by that percentage.",
			"CRLF endings":      "WHEN a customer applies a percentage\r\ndiscount code THE SYSTEM SHALL reduce\r\nthe total by that percentage.",
			"lone CR endings":   "WHEN a customer applies a percentage\rdiscount code THE SYSTEM SHALL reduce\rthe total by that percentage.",
			"repeated spaces":   "WHEN  a customer applies   a percentage discount code THE SYSTEM SHALL reduce the total by that percentage.",
			"tabs":              "WHEN\ta customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that percentage.",
			"outer whitespace":  "\n  WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that percentage.  \n",
			"no-break spaces":   "WHEN a customer applies a percentage discount code THE SYSTEM SHALL reduce the total by that　percentage.",
			"blank line inside": "WHEN a customer applies a percentage discount code\n\nTHE SYSTEM SHALL reduce the total by that percentage.",
		}
		for name, statement := range variants {
			variant := base
			variant.Statement = statement
			if variant.StatementDigest() != base.StatementDigest() {
				t.Errorf("%s changed the digest", name)
			}
		}
	})

	t.Run("canonically equivalent Unicode forms give one digest [verifies SN-86~1]", func(t *testing.T) {
		composed := requirementOf("Café", "THE SYSTEM SHALL open the café.")
		decomposed := requirementOf(norm.NFD.String("Café"), norm.NFD.String("THE SYSTEM SHALL open the café."))
		if composed.StatementDigest() != decomposed.StatementDigest() {
			t.Fatal("NFC and NFD of the same text gave different digests")
		}
	})

	t.Run("a random reformatting of a text keeps the digest [verifies SN-86~1]", func(t *testing.T) {
		separators := spaceSeparators()
		property := func(seed int64) bool {
			random := rand.New(rand.NewSource(seed))
			reformatted, plain := randomText(random, separators)
			return requirementOf("", reformatted).StatementDigest() == requirementOf("", plain).StatementDigest()
		}
		if err := quick.Check(property, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("the canonical form is idempotent and holds no line feed inside a field [verifies SN-86~1]", func(t *testing.T) {
		property := func(text string) bool {
			once := domain.CanonicalText(text)
			return domain.CanonicalText(once) == once && !strings.ContainsAny(once, "\n\r")
		}
		if err := quick.Check(property, nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("the digest is 64 lowercase hexadecimal digits [verifies SN-86~1]", func(t *testing.T) {
		digest := requirementOf("", "THE SYSTEM SHALL work.").StatementDigest()
		if len(digest) != 64 || strings.Trim(digest, "0123456789abcdef") != "" {
			t.Fatalf("got %q", digest)
		}
	})

	t.Run("a real edit changes the digest [verifies SN-87~1]", func(t *testing.T) {
		base := domain.Requirement{
			ID: "REQ-1", Revision: 1, Title: "Totals", Status: domain.StatusNonTestable,
			Statement: "THE SYSTEM SHALL feel fast.", Justification: "judged in usability sessions",
		}
		edits := map[string]func(*domain.Requirement){
			"title":         func(r *domain.Requirement) { r.Title = "Totals!" },
			"statement":     func(r *domain.Requirement) { r.Statement = "THE SYSTEM SHALL feel slow." },
			"status":        func(r *domain.Requirement) { r.Status = domain.StatusActive },
			"justification": func(r *domain.Requirement) { r.Justification = "judged in design reviews" },
			"case":          func(r *domain.Requirement) { r.Statement = "THE SYSTEM SHALL feel Fast." },
			"punctuation":   func(r *domain.Requirement) { r.Statement = "THE SYSTEM SHALL feel fast" },
		}
		for name, edit := range edits {
			edited := base
			edit(&edited)
			if edited.StatementDigest() == base.StatementDigest() {
				t.Errorf("editing the %s kept the digest", name)
			}
		}
	})

	t.Run("text moved between fields changes the digest [verifies SN-87~1]", func(t *testing.T) {
		asTitle := requirementOf("alpha beta", "")
		asStatement := requirementOf("", "alpha beta")
		split := requirementOf("alpha", "beta")
		digests := map[string]bool{}
		for _, requirement := range []domain.Requirement{asTitle, asStatement, split} {
			digests[requirement.StatementDigest()] = true
		}
		if len(digests) != 3 {
			t.Fatal("two placements of the same words gave one digest")
		}
	})

	t.Run("replacing one word of a text changes the digest [verifies SN-87~1]", func(t *testing.T) {
		property := func(seed int64) bool {
			random := rand.New(rand.NewSource(seed))
			count := 2 + random.Intn(10)
			words := make([]string, count)
			for index := range words {
				words[index] = sampleWords[random.Intn(len(sampleWords))]
			}
			edited := append([]string(nil), words...)
			position := random.Intn(count)
			edited[position] += "z"
			return requirementOf("", strings.Join(words, " ")).StatementDigest() !=
				requirementOf("", strings.Join(edited, " ")).StatementDigest()
		}
		if err := quick.Check(property, nil); err != nil {
			t.Fatal(err)
		}
	})
}

func FuzzStatementDigest(f *testing.F) {
	for _, seed := range []string{"", "a  b", "a\r\nb", "é", "\xff\xfe", " x y"} {
		f.Add(seed, seed)
	}
	f.Fuzz(func(t *testing.T, title, statement string) {
		requirement := requirementOf(title, statement)
		canonical := requirementOf(domain.CanonicalText(title), domain.CanonicalText(statement))
		if requirement.StatementDigest() != canonical.StatementDigest() {
			t.Fatal("the digest of a text differs from the digest of its canonical form")
		}
		if once := domain.CanonicalText(statement); domain.CanonicalText(once) != once {
			t.Fatal("the canonical form is not idempotent")
		}
	})
}
