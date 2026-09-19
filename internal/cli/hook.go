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
	prior := event.PriorBlocks
	if !event.CountsBlocks {
		prior = attempts.Count(event.SessionID)
	}
	verdict := gateVerdict()
	if !verdict.blocked {
		attempts.Reset(event.SessionID)
		if prior > 0 {
			return agenthook.DecisionAnnouncePass, verdict.message
		}
		return agenthook.DecisionAllow, ""
	}
	if prior >= agenthook.MaxAttempts {
		attempts.Reset(event.SessionID)
		return agenthook.DecisionGiveUp, agenthook.GiveUpMessage(prior, verdict.message)
	}
	if !event.CountsBlocks {
		// A counter that cannot be written only makes the hook more insistent; the harness's own limits still apply.
		_ = attempts.Record(event.SessionID, prior+1)
	}
	return agenthook.DecisionBlock, verdict.message
}

// hookVerdict is what the gate concluded, worded for the agent or the user.
type hookVerdict struct {
	blocked bool
	message string
}

// gateVerdict runs the gate of the working directory's shallnot.yaml. It
// checks existing results when the project configures no test command.
func gateVerdict() hookVerdict {
	settings, err := config.Load(config.DefaultFileName)
	if err != nil {
		return hookVerdict{blocked: true, message: agenthook.FailureMessage(err)}
	}
	var outcome app.Outcome
	if len(settings.TestCommands) > 0 {
		outcome, err = app.Gate(settings, testrun.Shell{}, io.Discard)
	} else {
		outcome, err = app.Run(settings)
	}
	if err != nil {
		return hookVerdict{blocked: true, message: agenthook.FailureMessage(err)}
	}
	if outcome.Analysis.Verdict == domain.VerdictFail && !outcome.Policy.Advisory {
		return hookVerdict{blocked: true, message: agenthook.BlockedMessage(outcome.Analysis.Findings)}
	}
	covered, nonTestable := 0, 0
	for _, row := range outcome.Analysis.Rows {
		switch {
		case !row.InFocus:
		case row.Coverage == domain.CoverageCovered:
			covered++
		case row.Coverage == domain.CoverageNonTestable:
			nonTestable++
		}
	}
	return hookVerdict{message: agenthook.PassedMessage(covered, nonTestable)}
}
