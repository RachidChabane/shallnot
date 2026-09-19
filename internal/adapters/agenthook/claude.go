package agenthook

import (
	"encoding/json"
	"fmt"
)

const (
	claudeProjectDirVariable = "CLAUDE_PROJECT_DIR"
	// claudeExitBlock makes Claude Code continue the turn with standard error as the reason.
	claudeExitBlock = 2
	// claudeExitNotice lets the turn end and shows the first line of standard error to the user.
	claudeExitNotice = 1
)

// ClaudeStop speaks the protocol of Claude Code's Stop hook.
type ClaudeStop struct{}

type claudeStopInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
}

// Name is the argument of `shallnot hook`.
func (ClaudeStop) Name() string { return "claude-stop" }

// ParseEvent reads the Stop hook input. The project is $CLAUDE_PROJECT_DIR, else the session's working directory.
func (ClaudeStop) ParseEvent(stdin []byte, environment func(string) string) (Event, error) {
	var input claudeStopInput
	if len(stdin) > 0 {
		if err := json.Unmarshal(stdin, &input); err != nil {
			return Event{}, fmt.Errorf("claude-stop: hook input is not JSON: %w", err)
		}
	}
	projectDir := environment(claudeProjectDirVariable)
	if projectDir == "" {
		projectDir = input.Cwd
	}
	return Event{ProjectDir: projectDir, SessionID: input.SessionID}, nil
}

// Respond blocks with exit code 2 and the message on standard error.
func (ClaudeStop) Respond(decision Decision, message string) Response {
	switch decision {
	case DecisionBlock:
		return Response{Stderr: message, ExitCode: claudeExitBlock}
	case DecisionGiveUp:
		return Response{Stderr: message, ExitCode: claudeExitNotice}
	default:
		return Response{}
	}
}
