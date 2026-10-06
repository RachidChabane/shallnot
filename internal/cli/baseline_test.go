package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
)

const baselineSpec = "- **PARK-2~1**: WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00.\n" +
	"- **PARK-3~1**: THE SYSTEM SHALL print a ticket.\n"

type baselineProject struct {
	directory string
	report    string
}

// newBaselineProject writes a spec and the JSON report of a first run over it.
func newBaselineProject(t *testing.T) baselineProject {
	t.Helper()
	inRepository(t)
	project := baselineProject{directory: t.TempDir()}
	project.report = filepath.Join(project.directory, "base.json")
	project.write(t, baselineSpec)
	result := project.check(t, "--json-out", project.report)
	if result.stderr != "" {
		t.Fatalf("first run: %s", result.stderr)
	}
	return project
}

func (p baselineProject) specPath() string { return filepath.Join(p.directory, "plan.md") }

func (p baselineProject) write(t *testing.T, spec string) {
	t.Helper()
	if err := os.WriteFile(p.specPath(), []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (p baselineProject) check(t *testing.T, extra ...string) run {
	t.Helper()
	args := []string{"check", "--no-config", "--specs", p.specPath(), "--results", cartCheckResults, "--advisory", "--format", "json"}
	return shallnot(t, append(args, extra...)...)
}

func statementChanged(t *testing.T, result run) []string {
	t.Helper()
	var refs []string
	for _, finding := range decode(t, result.stdout).Findings {
		if finding.Category == domain.CategoryStatementChanged {
			refs = append(refs, string(finding.RequirementID))
		}
	}
	return refs
}

func TestBaselineFlag(t *testing.T) {
	t.Run("reports a statement edited without a revision bump, and only that one [verifies SN-99~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		project.write(t, strings.Replace(baselineSpec, "charge 0.00", "charge 1.00", 1))
		result := project.check(t, "--baseline", project.report)
		if refs := statementChanged(t, result); len(refs) != 1 || refs[0] != "PARK-2" {
			t.Fatalf("got %v\n%s", refs, result.stdout)
		}
	})

	t.Run("reports nothing for a reformatted spec or a bumped revision [verifies SN-99~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		reformatted := "- **PARK-2~1**:   WHEN a stay lasts 15 minutes\r\n  or less THE SYSTEM SHALL charge 0.00.\r\n- **PARK-3~2**: THE SYSTEM SHALL print a better ticket.\r\n"
		project.write(t, reformatted)
		if refs := statementChanged(t, project.check(t, "--baseline", project.report)); len(refs) != 0 {
			t.Fatalf("got %v", refs)
		}
	})

	t.Run("reports nothing without the flag [verifies SN-99~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		project.write(t, strings.Replace(baselineSpec, "charge 0.00", "charge 1.00", 1))
		if refs := statementChanged(t, project.check(t)); len(refs) != 0 {
			t.Fatalf("got %v", refs)
		}
	})

	t.Run("is a warning that does not block unless configured to [verifies SN-99~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		project.write(t, strings.Replace(baselineSpec, "charge 0.00", "charge 1.00", 1))
		blocked := func(extra ...string) bool {
			args := []string{"check", "--no-config", "--specs", project.specPath(), "--results", cartCheckResults, "--baseline", project.report, "--severity", "uncovered_requirement=off", "--severity", "orphan_tag=off", "--severity", "malformed_tag=off"}
			return shallnot(t, append(args, extra...)...).exit == app.ExitBlocked
		}
		if blocked() {
			t.Fatal("a statement change blocked by default")
		}
		if !blocked("--severity", "statement_changed=error") {
			t.Fatal("a statement change at error did not block")
		}
	})

	t.Run("gate takes the flag too [verifies SN-99~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		project.write(t, strings.Replace(baselineSpec, "charge 0.00", "charge 1.00", 1))
		results := filepath.Join(project.directory, "junit.xml")
		content, err := os.ReadFile(cartCheckResults)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(results, content, 0o600); err != nil {
			t.Fatal(err)
		}
		result := shallnot(t, "gate", "--no-config", "--specs", project.specPath(), "--results", results, "--advisory", "--format", "json",
			"--test-command", "touch "+results, "--baseline", project.report)
		if refs := statementChanged(t, result); len(refs) != 1 {
			t.Fatalf("exit %d, got %v, stderr %s", result.exit, refs, result.stderr)
		}
	})

	t.Run("a baseline that cannot be used is a tool failure with no report [verifies SN-52~1]", func(t *testing.T) {
		project := newBaselineProject(t)
		older := filepath.Join(project.directory, "older.json")
		if err := os.WriteFile(older, []byte(`{"schema_version": "1.1", "requirements": []}`), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{older, filepath.Join(project.directory, "missing.json"), project.specPath()} {
			result := project.check(t, "--baseline", path)
			if result.exit != app.ExitToolFailure || result.stdout != "" || result.stderr == "" {
				t.Errorf("%s: exit %d, stdout %q, stderr %q", path, result.exit, result.stdout, result.stderr)
			}
		}
	})
}
