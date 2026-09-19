// Package agenthook adapts the gate to the end-of-turn hooks of agent
// harnesses: it reads the harness's hook input and answers in the harness's
// own protocol, so that an agent cannot finish its turn on a blocked gate.
package agenthook

import (
	"fmt"
	"sort"
	"strings"
)

// MaxAttempts is how many consecutive times a hook sends the agent back to
// work before it lets the turn end and reports to the user instead.
const MaxAttempts = 3

// Event is what a hook needs to know from the harness.
type Event struct {
	// ProjectDir is the root of the project the agent works in.
	ProjectDir string
	// SessionID identifies the conversation, to count attempts across hook calls.
	SessionID string
	// PriorBlocks is how many times in a row the harness says this hook already sent the agent back, when it counts them itself.
	PriorBlocks int
	// CountsBlocks tells whether the harness counts PriorBlocks itself.
	CountsBlocks bool
}

// Decision is what the gate concluded for this end of turn.
type Decision uint8

const (
	// DecisionAllow lets the turn end silently.
	DecisionAllow Decision = iota
	// DecisionBlock sends the agent back to work with a message.
	DecisionBlock
	// DecisionGiveUp lets the turn end and tells the user the gate is still blocked.
	DecisionGiveUp
)

// Response is what the hook process prints and returns.
type Response struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Harness is the strategy for one agent harness's hook protocol.
type Harness interface {
	// Name is the argument of `shallnot hook`.
	Name() string
	// ParseEvent reads the hook input the harness wrote on standard input.
	ParseEvent(stdin []byte, environment func(string) string) (Event, error)
	// Respond renders a decision in the harness's protocol.
	Respond(decision Decision, message string) Response
}

var harnesses = map[string]Harness{
	ClaudeStop{}.Name(): ClaudeStop{},
	CursorStop{}.Name(): CursorStop{},
}

// Lookup returns the harness adapter of that name.
func Lookup(name string) (Harness, error) {
	harness, known := harnesses[name]
	if !known {
		return nil, fmt.Errorf("unknown hook %q (expected one of %s)", name, strings.Join(Names(), ", "))
	}
	return harness, nil
}

// Names lists the supported hooks.
func Names() []string {
	names := make([]string, 0, len(harnesses))
	for name := range harnesses {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
