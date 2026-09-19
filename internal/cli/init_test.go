package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/plugin"
)

func writeFiles(t *testing.T, directory string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readFile(t *testing.T, directory, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestInitCommand(t *testing.T) {
	t.Run("writes a config for the test runners it finds [verifies SN-74~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"pytest.ini": "", "tests/test_x.py": "", "package.json": `{"devDependencies": {"vitest": "1"}}`, "src/x.ts": ""})
		if result := shallnot(t, "init", "--dir", directory, "--hooks", "none"); result.exit != app.ExitClean {
			t.Fatalf("got %+v", result)
		}
		config := readFile(t, directory, "shallnot.yaml")
		for _, expected := range []string{"version: 1", "  - specs", "  - tests", "  - src", "test-results/pytest.xml", "test-results/vitest.xml", "pytest -o junit_family=xunit1", "npx vitest run"} {
			if !strings.Contains(config, expected) {
				t.Errorf("no %q in:\n%s", expected, config)
			}
		}
		if result := shallnot(t, "check", "--config", filepath.Join(directory, "shallnot.yaml")); !strings.Contains(result.stderr, `spec "`) {
			t.Fatalf("the written config is not accepted up to the missing spec: %+v", result)
		}
	})

	t.Run("never rewrites an existing config [verifies SN-74~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"shallnot.yaml": "version: 1\nspecs: [plans]\n", "go.mod": "module x\n"})
		shallnot(t, "init", "--dir", directory, "--hooks", "none")
		if config := readFile(t, directory, "shallnot.yaml"); config != "version: 1\nspecs: [plans]\n" {
			t.Fatalf("got %q", config)
		}
	})

	t.Run("installs the agent instructions every harness reads [verifies SN-75~2]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"AGENTS.md": "# House rules\n\nBe kind.\n", "CLAUDE.md": "# Claude\n"})
		shallnot(t, "init", "--dir", directory, "--hooks", "none")
		agents := readFile(t, directory, "AGENTS.md")
		if !strings.HasPrefix(agents, "# House rules\n\nBe kind.\n\n<!-- shallnot:begin -->\n## Tracing requirements") || !strings.HasSuffix(agents, "<!-- shallnot:end -->\n") {
			t.Fatalf("got:\n%s", agents)
		}
		if strings.Contains(agents, "name: shallnot") {
			t.Fatal("the skill's front matter leaked into AGENTS.md")
		}
		if claude := readFile(t, directory, "CLAUDE.md"); claude != "# Claude\n\n@AGENTS.md\n" {
			t.Fatalf("got %q", claude)
		}
		for _, skill := range plugin.Skills() {
			if installed := readFile(t, directory, ".claude/skills/"+skill.Name+"/SKILL.md"); installed != string(skill.Content) {
				t.Fatalf("the installed %s skill differs from the packaged one", skill.Name)
			}
		}
		for _, heading := range []string{"## Tracing requirements to tests", "## Writing requirements for shallnot", "## Reviewing tests against their requirements"} {
			if !strings.Contains(agents, "\n"+heading) && !strings.Contains(agents, heading) {
				t.Errorf("AGENTS.md lacks %q", heading)
			}
		}
	})

	t.Run("updates its AGENTS.md section in place and leaves the rest alone [verifies SN-75~2]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"AGENTS.md": "Before.\n\n<!-- shallnot:begin -->\nold text\n<!-- shallnot:end -->\n\nAfter.\n"})
		shallnot(t, "init", "--dir", directory, "--hooks", "none")
		agents := readFile(t, directory, "AGENTS.md")
		if !strings.HasPrefix(agents, "Before.\n\n<!-- shallnot:begin -->\n## Tracing") || !strings.HasSuffix(agents, "<!-- shallnot:end -->\n\nAfter.\n") || strings.Contains(agents, "old text") {
			t.Fatalf("got:\n%s", agents)
		}
	})

	t.Run("adds the pytest hook to conftest.py once [verifies SN-74~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"pytest.ini": "", "conftest.py": "import os\n"})
		shallnot(t, "init", "--dir", directory, "--hooks", "none")
		conftest := readFile(t, directory, "conftest.py")
		if !strings.HasPrefix(conftest, "import os\n") || strings.Count(conftest, "def pytest_collection_modifyitems") != 1 || !strings.Contains(conftest, `addinivalue_line("markers"`) {
			t.Fatalf("got:\n%s", conftest)
		}
	})

	t.Run("refuses to merge into a conftest.py that defines the same hooks [verifies SN-74~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"pytest.ini": "", "conftest.py": "def pytest_configure(config):\n    pass\n"})
		if result := shallnot(t, "init", "--dir", directory); result.exit != app.ExitToolFailure || !strings.Contains(result.stderr, "by hand") {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("installs the end-of-turn hooks beside the project's other settings [verifies SN-76~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{
			".claude/settings.json": `{"permissions": {"allow": ["Bash"]}, "hooks": {"Stop": [{"hooks": [{"type": "command", "command": "make lint"}]}]}}`,
			".cursor/rules/x.mdc":   "",
		})
		if result := shallnot(t, "init", "--dir", directory); result.exit != app.ExitClean {
			t.Fatalf("got %+v", result)
		}
		var claude struct {
			Permissions map[string][]string `json:"permissions"`
			Hooks       map[string][]struct {
				Hooks []struct {
					Command string `json:"command"`
					Timeout int    `json:"timeout"`
				} `json:"hooks"`
			} `json:"hooks"`
		}
		if err := json.Unmarshal([]byte(readFile(t, directory, ".claude/settings.json")), &claude); err != nil {
			t.Fatal(err)
		}
		stop := claude.Hooks["Stop"]
		if len(claude.Permissions["allow"]) != 1 || len(stop) != 2 || stop[0].Hooks[0].Command != "make lint" || stop[1].Hooks[0].Command != "shallnot hook claude-stop" || stop[1].Hooks[0].Timeout == 0 {
			t.Fatalf("got %+v", claude)
		}
		cursor := readFile(t, directory, ".cursor/hooks.json")
		if !strings.Contains(cursor, `"version": 1`) || !strings.Contains(cursor, `"command": "shallnot hook cursor-stop"`) {
			t.Fatalf("got:\n%s", cursor)
		}
	})

	t.Run("is idempotent, and --check reports pending changes without writing [verifies SN-77~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"go.mod": "module x\n"})
		if pending := shallnot(t, "init", "--dir", directory, "--check"); pending.exit != app.ExitBlocked || !strings.Contains(pending.stdout, "create    AGENTS.md") {
			t.Fatalf("got %+v", pending)
		}
		if _, err := os.Stat(filepath.Join(directory, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatal("--check wrote a file")
		}
		shallnot(t, "init", "--dir", directory)
		again := shallnot(t, "init", "--dir", directory, "--check")
		if again.exit != app.ExitClean || strings.Contains(again.stdout, "create") || strings.Contains(again.stdout, "update") {
			t.Fatalf("got %+v", again)
		}
	})

	t.Run("tells what the project must still do [verifies SN-74~1]", func(t *testing.T) {
		directory := t.TempDir()
		writeFiles(t, directory, map[string]string{"pom.xml": "<project/>"})
		result := shallnot(t, "init", "--dir", directory, "--hooks", "none")
		if !strings.Contains(result.stdout, "next: Maven:") || !strings.Contains(result.stdout, "next: Write the requirements in specs/") {
			t.Fatalf("got:\n%s", result.stdout)
		}
	})

	t.Run("refuses an unknown hook harness [verifies SN-76~1]", func(t *testing.T) {
		if result := shallnot(t, "init", "--dir", t.TempDir(), "--hooks", "emacs"); result.exit != app.ExitToolFailure {
			t.Fatalf("got %+v", result)
		}
	})
}
