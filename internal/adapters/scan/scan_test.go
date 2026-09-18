package scan_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/scan"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

const root = "testdata/project"

func scanProject(t *testing.T, scanner scan.Scanner) map[string][]domain.SourceTag {
	t.Helper()
	scanner.Extractors = scan.DefaultExtractors(testkit.Grammar(t))
	tags, err := scanner.ScanRoot(root)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	byFile := map[string][]domain.SourceTag{}
	for _, tag := range tags {
		file := filepath.Base(tag.Location.File)
		byFile[file] = append(byFile[file], tag)
	}
	return byFile
}

type summary struct {
	line      int
	label     string
	refs      string
	malformed int
}

func summarize(tags []domain.SourceTag) []summary {
	var summaries []summary
	for _, tag := range tags {
		refs := ""
		for _, ref := range tag.Refs {
			refs += ref.String() + " "
		}
		summaries = append(summaries, summary{tag.Location.Line, tag.Label, refs, len(tag.Malformed)})
	}
	return summaries
}

func TestScanner(t *testing.T) {
	found := scanProject(t, scan.Scanner{Excludes: scan.DefaultExcludes})

	t.Run("finds pytest markers and record_property calls, labelled with what they decorate [verifies SN-13~1]", func(t *testing.T) {
		want := []summary{
			{3, "test_cart", "REQ-9~1 ", 0},
			{6, "test_total_sums_items", "REQ-1~1 ", 0},
			{12, "test_discount", "REQ-2~2 REQ-3~1 ", 0},
			{21, "test_checkout", "REQ-4~1 REQ-5~1 ", 0},
			{25, "TestRounding", "REQ-6~1 ", 0},
			{36, "test_variable_argument", "", 1},
			{41, "test_no_revision", "", 1},
		}
		if got := summarize(found["test_cart.py"]); !reflect.DeepEqual(got, want) {
			t.Fatalf("got  %+v\nwant %+v", got, want)
		}
	})

	t.Run("finds bracket tags in any language, labelled with the static part of their title [verifies SN-14~1]", func(t *testing.T) {
		tag := func(ref string) string { return testkit.Tag(ref) }
		want := []summary{
			{3, "discount codes " + tag("REQ-2~2"), "REQ-2~2 ", 0},
			{4, "applies a code " + tag("REQ-3~1"), "REQ-3~1 ", 0},
			{5, "off the total " + tag("REQ-2~2") + " every time", "REQ-2~2 ", 0},
			{6, "totals " + tag("REQ-4~1"), "REQ-4~1 ", 0},
			{9, "", "REQ-5~1 ", 0},
			{11, "has no revision " + "[" + domain.TagKeyword + " REQ-6]", "", 1},
		}
		if got := summarize(found["cart.test.ts"]); !reflect.DeepEqual(got, want) {
			t.Fatalf("got  %+v\nwant %+v", got, want)
		}
	})

	t.Run("skips dependency directories and binary files [verifies SN-14~1]", func(t *testing.T) {
		if len(found["dep.test.js"]) != 0 || len(found["logo.bin"]) != 0 {
			t.Fatalf("got %+v", found)
		}
		if everything := scanProject(t, scan.Scanner{}); len(everything["dep.test.js"]) != 1 {
			t.Fatal("dependency directories are skipped even without default excludes")
		}
	})

	t.Run("skips the paths the project excludes and the files that are other inputs [verifies SN-14~1]", func(t *testing.T) {
		absolute, err := filepath.Abs(filepath.Join(root, "tests", "test_cart.py"))
		if err != nil {
			t.Fatal(err)
		}
		found := scanProject(t, scan.Scanner{Excludes: []string{"**/web/**"}, Skip: map[string]bool{absolute: true}})
		if len(found["cart.test.ts"]) != 0 || len(found["test_cart.py"]) != 0 || len(found["dep.test.js"]) != 1 {
			t.Fatalf("got %+v", found)
		}
	})

	t.Run("refuses a test root that is not a directory [verifies SN-52~1]", func(t *testing.T) {
		scanner := scan.Scanner{}
		if _, err := scanner.ScanRoot("testdata/missing"); err == nil {
			t.Fatal("a missing root was accepted")
		}
		if _, err := scanner.ScanRoot("scan_test.go"); err == nil {
			t.Fatal("a file was accepted as a root")
		}
	})
}
