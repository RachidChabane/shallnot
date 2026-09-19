package agenthook

import (
	"encoding/json"
	"fmt"
)

// CursorStop speaks the protocol of Cursor's `stop` hook.
type CursorStop struct{}

type cursorStopInput struct {
	ConversationID string   `json:"conversation_id"`
	WorkspaceRoots []string `json:"workspace_roots"`
	LoopCount      int      `json:"loop_count"`
}

type cursorStopOutput struct {
	FollowupMessage string `json:"followup_message"`
}

// Name is the argument of `shallnot hook`.
func (CursorStop) Name() string { return "cursor-stop" }

// ParseEvent reads the stop hook input. Cursor counts the follow-ups it already submitted.
func (CursorStop) ParseEvent(stdin []byte, _ func(string) string) (Event, error) {
	var input cursorStopInput
	if len(stdin) > 0 {
		if err := json.Unmarshal(stdin, &input); err != nil {
			return Event{}, fmt.Errorf("cursor-stop: hook input is not JSON: %w", err)
		}
	}
	event := Event{SessionID: input.ConversationID, PriorBlocks: input.LoopCount, CountsBlocks: true}
	if len(input.WorkspaceRoots) > 0 {
		event.ProjectDir = input.WorkspaceRoots[0]
	}
	return event, nil
}

// Respond blocks by submitting the message as the next user message.
func (CursorStop) Respond(decision Decision, message string) Response {
	switch decision {
	case DecisionBlock:
		output, err := json.Marshal(cursorStopOutput{FollowupMessage: message})
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
