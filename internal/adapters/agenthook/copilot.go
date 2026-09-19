package agenthook

import (
	"encoding/json"
	"fmt"
)

// CopilotStopName is the hook for GitHub Copilot's `agentStop` event.
const CopilotStopName = "copilot-stop"

const copilotDecisionBlock = "block"

// CopilotStop speaks the protocol of GitHub Copilot's `agentStop` hook, which
// reads its decision from standard output and treats any non-zero exit as a
// hook failure.
type CopilotStop struct{}

type copilotStopInput struct {
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
}

type copilotStopOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// Name is the argument of `shallnot hook`.
func (CopilotStop) Name() string { return CopilotStopName }

// ParseEvent reads the agentStop input.
func (CopilotStop) ParseEvent(stdin []byte, _ func(string) string) (Event, error) {
	var input copilotStopInput
	if len(stdin) > 0 {
		if err := json.Unmarshal(stdin, &input); err != nil {
			return Event{}, fmt.Errorf("%s: hook input is not JSON: %w", CopilotStopName, err)
		}
	}
	return Event{ProjectDir: input.Cwd, SessionID: input.SessionID}, nil
}

// Respond blocks with a decision on standard output; the reason becomes the agent's next prompt.
func (CopilotStop) Respond(decision Decision, message string) Response {
	switch decision {
	case DecisionBlock:
		output, err := json.Marshal(copilotStopOutput{Decision: copilotDecisionBlock, Reason: message})
		if err != nil {
			return Response{Stderr: err.Error()}
		}
		return Response{Stdout: string(output) + "\n"}
	case DecisionGiveUp, DecisionAnnouncePass:
		return Response{Stderr: message}
	default:
		return Response{}
	}
}
