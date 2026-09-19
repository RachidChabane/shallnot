package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
	"github.com/RachidChabane/shallnot/internal/app"
)

const (
	claudeExitBlock  app.ExitCode = 2
	claudeExitNotice app.ExitCode = 1
)

// isolateAttempts keeps the attempt counters of a test out of the machine's temporary directory.
func isolateAttempts(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	t.Setenv("TMPDIR", directory)
	t.Setenv("TMP", directory)
	t.Setenv("TEMP", directory)
}

func claudeInput(t *testing.T, projectDir string) string {
	t.Helper()
	input, err := json.Marshal(map[string]any{"session_id": "session-1", "cwd": projectDir, "hook_event_name": "Stop"})
	if err != nil {
		t.Fatal(err)
	}
	return string(input)
}

func cursorInput(t *testing.T, projectDir string, loopCount int) string {
	t.Helper()
	input, err := json.Marshal(map[string]any{"conversation_id": "c-1", "workspace_roots": []string{projectDir}, "loop_count": loopCount, "status": "completed"})
	if err != nil {
		t.Fatal(err)
	}
	return string(input)
}

func TestHookCommand(t *testing.T) {
	produce := copyCommand("prepared.xml", "junit.xml")

	t.Run("stays silent in a project without a shallnot.yaml [verifies SN-72~1]", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", "")
		result := shallnotWithInput(t, claudeInput(t, t.TempDir()), "hook", "claude-stop")
		if result.exit != app.ExitClean || result.stdout != "" || result.stderr != "" {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("lets a Claude Code turn end when the gate passes [verifies SN-72~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", gateProject(t, produce))
		result := shallnotWithInput(t, "", "hook", "claude-stop")
		if result.exit != app.ExitClean || result.stdout != "" || result.stderr != "" {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("sends a Claude Code agent back to work with the blocking findings [verifies SN-72~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", "")
		project := gateProjectWithSpec(t, uncoveredSpec, produce)
		result := shallnotWithInput(t, claudeInput(t, project), "hook", "claude-stop")
		if result.exit != claudeExitBlock || result.stdout != "" {
			t.Fatalf("got %+v", result)
		}
		for _, expected := range []string{"uncovered_requirement at spec.md:2", "REQ-2~1", "Never delete or alter a tag"} {
			if !strings.Contains(result.stderr, expected) {
				t.Errorf("no %q in:\n%s", expected, result.stderr)
			}
		}
	})

	t.Run("sends the agent back when the test commands produce no results [verifies SN-72~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", gateProject(t, "exit 0"))
		result := shallnotWithInput(t, "", "hook", "claude-stop")
		if result.exit != claudeExitBlock || !strings.Contains(result.stderr, "could not produce a verdict") {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("gives up after the attempt limit and tells the user [verifies SN-73~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", "")
		input := claudeInput(t, gateProjectWithSpec(t, uncoveredSpec, produce))
		for attempt := 1; attempt <= agenthook.MaxAttempts; attempt++ {
			if result := shallnotWithInput(t, input, "hook", "claude-stop"); result.exit != claudeExitBlock {
				t.Fatalf("attempt %d: %+v", attempt, result)
			}
		}
		result := shallnotWithInput(t, input, "hook", "claude-stop")
		if result.exit != claudeExitNotice || !strings.HasPrefix(result.stderr, "shallnot: the gate is still blocked after 3 attempts") {
			t.Fatalf("got %+v", result)
		}
		if again := shallnotWithInput(t, input, "hook", "claude-stop"); again.exit != claudeExitBlock {
			t.Fatalf("the count was not reset after giving up: %+v", again)
		}
	})

	t.Run("submits the findings to a Cursor agent as a follow-up message [verifies SN-72~1]", func(t *testing.T) {
		t.Chdir(t.TempDir())
		project := gateProjectWithSpec(t, uncoveredSpec, produce)
		result := shallnotWithInput(t, cursorInput(t, project, 0), "hook", "cursor-stop")
		var output struct {
			FollowupMessage string `json:"followup_message"`
		}
		if err := json.Unmarshal([]byte(result.stdout), &output); err != nil || result.exit != app.ExitClean {
			t.Fatalf("got %+v (%v)", result, err)
		}
		if !strings.Contains(output.FollowupMessage, "uncovered_requirement at spec.md:2") {
			t.Fatalf("got %q", output.FollowupMessage)
		}
	})

	t.Run("stops following up once Cursor reports the attempt limit [verifies SN-73~1]", func(t *testing.T) {
		t.Chdir(t.TempDir())
		project := gateProjectWithSpec(t, uncoveredSpec, produce)
		result := shallnotWithInput(t, cursorInput(t, project, agenthook.MaxAttempts), "hook", "cursor-stop")
		if result.exit != app.ExitClean || result.stdout != "" || !strings.Contains(result.stderr, "still blocked") {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("an advisory project never holds the agent back [verifies SN-72~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		t.Setenv("CLAUDE_PROJECT_DIR", "")
		project := gateProjectWithSpec(t, uncoveredSpec, produce)
		appendToConfig(t, project, "advisory: true\n")
		if result := shallnotWithInput(t, claudeInput(t, project), "hook", "claude-stop"); result.exit != app.ExitClean || result.stderr != "" {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("a hook that fails for its own reasons never holds the agent [verifies SN-72~1]", func(t *testing.T) {
		if result := shallnotWithInput(t, "", "hook", "vim-stop"); result.exit != claudeExitNotice {
			t.Fatalf("unknown harness: %+v", result)
		}
		if result := shallnotWithInput(t, "not json", "hook", "claude-stop"); result.exit != claudeExitNotice || result.exit == claudeExitBlock {
			t.Fatalf("bad input: %+v", result)
		}
	})
}
