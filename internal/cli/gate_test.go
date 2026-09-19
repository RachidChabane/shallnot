package cli_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

// copyCommand is a command line that copies a file, standing in for a test runner writing its report.
func copyCommand(from, to string) string {
	if runtime.GOOS == "windows" {
		return "copy /Y " + from + " " + to
	}
	return "cp " + from + " " + to
}

// gateProject writes a project whose "test runner" copies a prepared report into place.
func gateProject(t *testing.T, testCommands ...string) string {
	t.Helper()
	return gateProjectWithSpec(t, coveredSpec, testCommands...)
}

const (
	coveredSpec   = "- **REQ-1~1**: THE SYSTEM SHALL work.\n"
	uncoveredSpec = coveredSpec + "- **REQ-2~1**: THE SYSTEM SHALL also do this.\n"
)

func gateProjectWithSpec(t *testing.T, spec string, testCommands ...string) string {
	t.Helper()
	directory := t.TempDir()
	files := map[string]string{
		"spec.md":      spec,
		"prepared.xml": `<testsuite name="s"><testcase classname="c" name="works ` + testkit.Tag("REQ-1~1") + `"/></testsuite>`,
		"shallnot.yaml": "version: 1\nspecs: [spec.md]\nresults: [junit.xml]\ntest_commands:\n" +
			"  - " + strings.Join(testCommands, "\n  - ") + "\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func TestGateCommand(t *testing.T) {
	produce := copyCommand("prepared.xml", "junit.xml")

	t.Run("runs the test commands in the config file's directory, then checks their results [verifies SN-70~1]", func(t *testing.T) {
		directory := gateProject(t, produce)
		result := shallnot(t, "gate", "--config", filepath.Join(directory, "shallnot.yaml"), "--format", "json")
		document := decode(t, result.stdout)
		if result.exit != app.ExitClean || document.Verdict != domain.VerdictPass || document.Summary.Requirements.Covered != 1 {
			t.Fatalf("exit %d, stderr %s", result.exit, result.stderr)
		}
		if !strings.Contains(result.stderr, "running "+produce) {
			t.Fatalf("the test command was not logged on standard error:\n%s", result.stderr)
		}
	})

	t.Run("a failing test command still leads to a verdict [verifies SN-70~1]", func(t *testing.T) {
		directory := gateProject(t, produce, "exit 3")
		result := shallnot(t, "gate", "--config", filepath.Join(directory, "shallnot.yaml"))
		if result.exit != app.ExitClean || !strings.Contains(result.stderr, "exited with status 3") {
			t.Fatalf("exit %d, stderr %s", result.exit, result.stderr)
		}
	})

	t.Run("refuses results that the test commands did not write [verifies SN-71~1]", func(t *testing.T) {
		directory := gateProject(t, "exit 0")
		stale := filepath.Join(directory, "junit.xml")
		prepared, err := os.ReadFile(filepath.Join(directory, "prepared.xml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stale, prepared, 0o644); err != nil {
			t.Fatal(err)
		}
		result := shallnot(t, "gate", "--config", filepath.Join(directory, "shallnot.yaml"))
		if result.exit != app.ExitToolFailure || result.stdout != "" || !strings.Contains(result.stderr, "not written by this run") {
			t.Fatalf("exit %d\nstdout %q\nstderr %q", result.exit, result.stdout, result.stderr)
		}
	})

	t.Run("refuses to run without a test command, and when no results appear [verifies SN-71~1]", func(t *testing.T) {
		if result := shallnot(t, "gate", "--no-config", "--specs", "x.md", "--results", "x.xml"); result.exit != app.ExitToolFailure {
			t.Fatalf("no test command: exit %d", result.exit)
		}
		directory := gateProject(t, "exit 0")
		if result := shallnot(t, "gate", "--config", filepath.Join(directory, "shallnot.yaml")); result.exit != app.ExitToolFailure || result.stdout != "" {
			t.Fatalf("no results: exit %d, stdout %q", result.exit, result.stdout)
		}
	})
}

func appendToConfig(t *testing.T, project, text string) {
	t.Helper()
	path := filepath.Join(project, "shallnot.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte(text)...), 0o644); err != nil {
		t.Fatal(err)
	}
}
