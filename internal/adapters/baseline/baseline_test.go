package baseline_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/baseline"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

const digestOne = "1640b763e3b155b4799bedb9a66384bfe5dc9d52030bf3c68d14aaa1b2c89478"

func report(version, requirements string) string {
	return `{"schema_version": "` + version + `", "requirements": [` + requirements + `]}`
}

func TestBaseline(t *testing.T) {
	t.Run("reads the statement digest of each requirement by reference [verifies SN-99~1]", func(t *testing.T) {
		digests, err := baseline.Parse([]byte(report("1.2",
			`{"id": "CART-2", "revision": 2, "statement_digest": "`+digestOne+`", "tests": []}, {"id": "CART-3", "revision": 1, "statement_digest": "abc"}`)))
		if err != nil || len(digests) != 2 || digests[testkit.Ref("CART-2", 2)] != digestOne || digests[testkit.Ref("CART-3", 1)] != "abc" {
			t.Fatalf("got %v, %v", digests, err)
		}
	})

	t.Run("accepts a later minor version of the report format [verifies SN-99~1]", func(t *testing.T) {
		if _, err := baseline.Parse([]byte(report("1.10", `{"id": "A-1", "revision": 1, "statement_digest": "abc"}`))); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reads a report without requirements as an empty baseline [verifies SN-99~1]", func(t *testing.T) {
		digests, err := baseline.Parse([]byte(report("1.2", "")))
		if err != nil || len(digests) != 0 {
			t.Fatalf("got %v, %v", digests, err)
		}
	})

	refused := map[string]string{
		"an older report without digests": report("1.1", `{"id": "A-1", "revision": 1}`),
		"version 1.0":                     report("1.0", ""),
		"another major version":           report("2.2", ""),
		"a malformed version":             report("one", ""),
		"a three-part version":            report("1.2.3", ""),
		"a requirement without a digest":  report("1.2", `{"id": "A-1", "revision": 1}`),
		"JSON of another shape":           `{"hello": "world"}`,
		"a list":                          `[]`,
		"text that is not JSON":           `requirements:`,
		"an empty file":                   ``,
	}
	for name, content := range refused {
		t.Run("refuses "+name+" [verifies SN-99~1]", func(t *testing.T) {
			if _, err := baseline.Parse([]byte(content)); err == nil {
				t.Fatal("accepted")
			}
		})
	}

	t.Run("refuses a file that cannot be read, naming it [verifies SN-99~1]", func(t *testing.T) {
		if _, err := baseline.Read(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("accepted a missing file")
		}
		path := filepath.Join(t.TempDir(), "report.json")
		if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := baseline.Read(path); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("got %v", err)
		}
	})
}

func FuzzBaseline(f *testing.F) {
	for _, seed := range []string{
		"", "{}", "[]", report("1.2", ""), report("1.2", `{"id": "A-1", "revision": 1, "statement_digest": "x"}`),
		report("1.", ""), report(".2", ""), `{"schema_version": 1, "requirements": 2}`, `{"requirements": null}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, content string) {
		digests, err := baseline.Parse([]byte(content))
		if err != nil && digests != nil {
			t.Fatal("a refused report returned digests")
		}
		for ref, digest := range digests {
			if digest == "" {
				t.Fatalf("empty digest for %v", ref)
			}
		}
	})
}
