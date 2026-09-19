package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
	"github.com/RachidChabane/shallnot/internal/adapters/config"
	"github.com/RachidChabane/shallnot/internal/adapters/testrun"
	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
)

// hookExitFailure is the exit code of a hook that fails for its own reasons.
// It is not app.ExitToolFailure: harnesses read exit code 2 as "hold the
// agent", and a broken hook must let the turn end.
const hookExitFailure app.ExitCode = 1

func failHook(stderr io.Writer, err error) app.ExitCode {
	fail(stderr, err)
	return hookExitFailure
}

// runHook answers an agent harness's end-of-turn hook: it gates the project
// the agent works in and replies in the harness's protocol. A project without
// a shallnot.yaml is none of its business, and the turn ends silently.
func runHook(args []string, stdin io.Reader, stdout, stderr io.Writer) app.ExitCode {
	if len(args) != 1 {
		return failHook(stderr, fmt.Errorf("usage: shallnot hook <%v>", agenthook.Names()))
	}
	harness, err := agenthook.Lookup(args[0])
	if err != nil {
		return failHook(stderr, err)
	}
	input, err := io.ReadAll(stdin)
	if err != nil {
		return failHook(stderr, err)
	}
	event, err := harness.ParseEvent(input, os.Getenv)
	if err != nil {
		return failHook(stderr, err)
	}
	if event.ProjectDir != "" {
		if err := os.Chdir(event.ProjectDir); err != nil {
			return failHook(stderr, err)
		}
	}
	if _, err := os.Stat(config.DefaultFileName); errors.Is(err, os.ErrNotExist) {
		return app.ExitClean
	}

	decision, message := decide(event, agenthook.Attempts{Directory: os.TempDir()})
	response := harness.Respond(decision, message)
	fmt.Fprint(stdout, response.Stdout)
	fmt.Fprint(stderr, response.Stderr)
	return app.ExitCode(response.ExitCode)
}

// decide gates the project and applies the attempt limit.
func decide(event agenthook.Event, attempts agenthook.Attempts) (agenthook.Decision, string) {
	message, blocked := gateMessage()
	if !blocked {
		attempts.Reset(event.SessionID)
		return agenthook.DecisionAllow, ""
	}
	prior := event.PriorBlocks
	if !event.CountsBlocks {
		prior = attempts.Count(event.SessionID)
	}
	if prior >= agenthook.MaxAttempts {
		attempts.Reset(event.SessionID)
		return agenthook.DecisionGiveUp, agenthook.GiveUpMessage(prior, message)
	}
	if !event.CountsBlocks {
		// A counter that cannot be written only makes the hook more insistent; the harness's own limits still apply.
		_ = attempts.Record(event.SessionID, prior+1)
	}
	return agenthook.DecisionBlock, message
}

// gateMessage runs the gate of the working directory's shallnot.yaml. It
// checks existing results when the project configures no test command.
func gateMessage() (message string, blocked bool) {
	settings, err := config.Load(config.DefaultFileName)
	if err != nil {
		return agenthook.FailureMessage(err), true
	}
	var outcome app.Outcome
	if len(settings.TestCommands) > 0 {
		outcome, err = app.Gate(settings, testrun.Shell{}, io.Discard)
	} else {
		outcome, err = app.Run(settings)
	}
	if err != nil {
		return agenthook.FailureMessage(err), true
	}
	if outcome.Analysis.Verdict == domain.VerdictFail && !outcome.Policy.Advisory {
		return agenthook.BlockedMessage(outcome.Analysis.Findings), true
	}
	return "", false
}
