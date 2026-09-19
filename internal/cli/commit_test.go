package cli_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestMain keeps the commit of the CI job that runs these tests out of the reports they compare.
func TestMain(m *testing.M) {
	for _, variable := range []string{"GITHUB_SHA", "CI_COMMIT_SHA"} {
		os.Unsetenv(variable)
	}
	os.Exit(m.Run())
}

func TestCommit(t *testing.T) {
	quickstart := []string{"check", "--no-config", "--specs", "examples/quickstart/spec.md", "--tests", "examples/quickstart/tests", "--results", "examples/quickstart/junit.xml"}
	reportedCommit := func(t *testing.T, args ...string) (string, bool) {
		t.Helper()
		var document struct {
			SchemaVersion string `json:"schema_version"`
			Run           map[string]any
		}
		result := shallnot(t, append(append([]string{}, quickstart...), append(args, "--format", "json")...)...)
		if err := json.Unmarshal([]byte(result.stdout), &document); err != nil {
			t.Fatalf("%v in %+v", err, result)
		}
		commit, present := document.Run["commit"]
		text, _ := commit.(string)
		return text, present
	}

	t.Run("records the commit given on the command line in every report [verifies SN-83~1]", func(t *testing.T) {
		inRepository(t)
		t.Setenv("GITHUB_SHA", "0123abc")
		if commit, _ := reportedCommit(t, "--commit", "feedbee"); commit != "feedbee" {
			t.Fatalf("got %q", commit)
		}
		for _, format := range []string{"terminal", "markdown"} {
			result := shallnot(t, append(append([]string{}, quickstart...), "--commit", "feedbee", "--format", format)...)
			if !strings.Contains(result.stdout, "feedbee") {
				t.Errorf("no commit in the %s report:\n%s", format, result.stdout)
			}
		}
	})

	t.Run("takes the commit from the CI environment when none is given [verifies SN-83~1]", func(t *testing.T) {
		inRepository(t)
		t.Setenv("CI_COMMIT_SHA", "cafe123")
		if commit, _ := reportedCommit(t); commit != "cafe123" {
			t.Fatalf("got %q", commit)
		}
		t.Setenv("GITHUB_SHA", "0123abc")
		if commit, _ := reportedCommit(t); commit != "0123abc" {
			t.Fatalf("got %q", commit)
		}
	})

	t.Run("carries no commit when none is known [verifies SN-83~1]", func(t *testing.T) {
		inRepository(t)
		if commit, present := reportedCommit(t); present {
			t.Fatalf("got %q", commit)
		}
		if result := shallnot(t, quickstart...); strings.Contains(result.stdout, "commit") {
			t.Fatalf("got:\n%s", result.stdout)
		}
	})
}
