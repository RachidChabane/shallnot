package agenthook

import (
	"encoding/json"
	"fmt"
)

const (
	// StopName is the hook that answers in the exit-code protocol.
	StopName = "stop"
	// ClaudeStopName is the name Claude Code configurations call the same hook by.
	ClaudeStopName = "claude-stop"

	// exitBlock makes the harness continue the turn with standard error as the reason.
	exitBlock = 2
	// exitNotice lets the turn end and shows standard error to the user.
	exitNotice = 1
)

// projectDirVariables are the environment variables in which harnesses of the
// exit-code protocol name the project root, in order of preference.
var projectDirVariables = []string{"CLAUDE_PROJECT_DIR", "GEMINI_PROJECT_DIR", "QWEN_PROJECT_DIR", "FACTORY_PROJECT_DIR"}

// ExitCodeStop speaks the end-of-turn protocol defined by Claude Code's Stop
// hook and shared by other harnesses: a JSON event on standard input, exit
// code 2 with the reason on standard error to continue the turn.
type ExitCodeStop struct {
	// HookName is the argument of `shallnot hook` this adapter answers to.
	HookName string
}

// stopOutput is the JSON a Stop hook may print; systemMessage is shown to the user.
type stopOutput struct {
	SystemMessage string `json:"systemMessage"`
}

type stopInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
}

// Name is the argument of `shallnot hook`.
func (s ExitCodeStop) Name() string { return s.HookName }

// ParseEvent reads the Stop hook input. The project is the root the harness
// names in its environment, else the session's working directory.
func (s ExitCodeStop) ParseEvent(stdin []byte, environment func(string) string) (Event, error) {
	var input stopInput
	if len(stdin) > 0 {
		if err := json.Unmarshal(stdin, &input); err != nil {
			return Event{}, fmt.Errorf("%s: hook input is not JSON: %w", s.HookName, err)
		}
	}
	projectDir := input.Cwd
	for _, variable := range projectDirVariables {
		if value := environment(variable); value != "" {
			projectDir = value
			break
		}
	}
	return Event{ProjectDir: projectDir, SessionID: input.SessionID}, nil
}

// Respond blocks with exit code 2 and the message on standard error.
func (ExitCodeStop) Respond(decision Decision, message string) Response {
	switch decision {
	case DecisionBlock:
		return Response{Stderr: message, ExitCode: exitBlock}
	case DecisionGiveUp:
		return Response{Stderr: message, ExitCode: exitNotice}
	case DecisionAnnouncePass:
		output, err := json.Marshal(stopOutput{SystemMessage: message})
		if err != nil {
			return Response{}
		}
		return Response{Stdout: string(output) + "\n"}
	default:
		return Response{}
	}
}
