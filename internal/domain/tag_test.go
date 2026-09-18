package domain_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func TestTagGrammar(t *testing.T) {
	grammar := testkit.Grammar(t)

	t.Run("reads one tag citing several requirements [verifies SN-10~1]", func(t *testing.T) {
		matches := grammar.FindBracketTags("sums items " + testkit.Tag("REQ-1~1", "ABC-101.AC3~12") + " and more")
		if len(matches) != 1 {
			t.Fatalf("got %d tags, want 1", len(matches))
		}
		want := []domain.Ref{testkit.Ref("REQ-1", 1), testkit.Ref("ABC-101.AC3", 12)}
		if !reflect.DeepEqual(matches[0].Tag.Refs, want) || len(matches[0].Tag.Malformed) != 0 {
			t.Fatalf("got %+v, want refs %v", matches[0].Tag, want)
		}
	})

	t.Run("reads every tag of a text [verifies SN-10~1]", func(t *testing.T) {
		matches := grammar.FindBracketTags(testkit.Tag("REQ-1~1") + " then " + testkit.Tag("REQ-2~3"))
		if len(matches) != 2 || matches[1].Tag.Refs[0] != testkit.Ref("REQ-2", 3) {
			t.Fatalf("got %+v", matches)
		}
	})

	malformed := map[string]string{
		"no revision":        testkit.Tag("REQ-1"),
		"zero revision":      testkit.Tag("REQ-1~0"),
		"non-numeric":        testkit.Tag("REQ-1~two"),
		"padded revision":    testkit.Tag("REQ-1~01"),
		"id outside pattern": testkit.Tag("req_one~1"),
		"no requirement":     "[" + domain.TagKeyword + "]",
		"empty list":         "[" + domain.TagKeyword + " ]",
		"trailing comma":     testkit.Tag("REQ-1~1", ""),
	}
	for name, text := range malformed {
		t.Run("rejects a tag with "+name+" [verifies SN-11~1]", func(t *testing.T) {
			matches := grammar.FindBracketTags(text)
			if len(matches) != 1 || len(matches[0].Tag.Malformed) == 0 {
				t.Fatalf("%q: got %+v, want a malformed tag", text, matches)
			}
		})
	}

	t.Run("keeps the well-formed references of a partly malformed tag [verifies SN-11~1]", func(t *testing.T) {
		matches := grammar.FindBracketTags(testkit.Tag("REQ-1~1", "REQ-2"))
		if len(matches[0].Tag.Refs) != 1 || len(matches[0].Tag.Malformed) != 1 {
			t.Fatalf("got %+v", matches[0].Tag)
		}
	})

	t.Run("reads a tag whose spaces a runner turned into underscores [verifies SN-12~1]", func(t *testing.T) {
		text := strings.ReplaceAll(testkit.Tag("REQ-1~1", "REQ-2~1"), " ", "_")
		matches := grammar.FindBracketTags("TestCart/sums_items_" + text)
		want := []domain.Ref{testkit.Ref("REQ-1", 1), testkit.Ref("REQ-2", 1)}
		if len(matches) != 1 || !reflect.DeepEqual(matches[0].Tag.Refs, want) {
			t.Fatalf("got %+v, want %v", matches, want)
		}
	})

	t.Run("ignores brackets that are not tags [verifies SN-10~1]", func(t *testing.T) {
		if matches := grammar.FindBracketTags("test_total[10.00-2] [verifiesX] [verified REQ-1~1]"); len(matches) != 0 {
			t.Fatalf("got %+v, want none", matches)
		}
	})
}

func TestIDPattern(t *testing.T) {
	t.Run("accepts the IDs of a custom convention only [verifies SN-7~1]", func(t *testing.T) {
		grammar := testkit.GrammarFor(t, `[a-z]+/[0-9]+`)
		matches := grammar.FindBracketTags(testkit.Tag("story/12~2", "REQ-1~1"))
		if !reflect.DeepEqual(matches[0].Tag.Refs, []domain.Ref{testkit.Ref("story/12", 2)}) || len(matches[0].Tag.Malformed) != 1 {
			t.Fatalf("got %+v", matches[0].Tag)
		}
	})

	for _, pattern := range []string{`[`, `.*`, `[A-Z~]+`, `[A-Z,]+`} {
		t.Run("refuses the unusable pattern "+pattern+" [verifies SN-7~1]", func(t *testing.T) {
			if _, err := domain.NewIDPattern(pattern); err == nil {
				t.Fatalf("pattern %q was accepted", pattern)
			}
		})
	}
}

func FuzzFindBracketTags(f *testing.F) {
	f.Add(testkit.Tag("REQ-1~1"))
	f.Add("[" + domain.TagKeyword + "_REQ-1~1,_REQ-2~2]")
	f.Add("[" + domain.TagKeyword + " ~~,,]")
	ids, err := domain.NewIDPattern(domain.DefaultIDPattern)
	if err != nil {
		f.Fatal(err)
	}
	grammar := domain.TagGrammar{IDs: ids}
	f.Fuzz(func(t *testing.T, text string) {
		for _, match := range grammar.FindBracketTags(text) {
			if len(match.Tag.Refs) == 0 && len(match.Tag.Malformed) == 0 {
				t.Fatalf("tag %q is neither bound nor malformed", match.Tag.Raw)
			}
			for _, ref := range match.Tag.Refs {
				reparsed, err := ids.ParseRef(ref.String())
				if err != nil || reparsed != ref {
					t.Fatalf("ref %v does not round-trip: %v", ref, err)
				}
			}
		}
	})
}
