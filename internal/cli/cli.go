// Package cli is the command-line front end.
package cli

import (
	"fmt"
	"io"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/version"
	"github.com/RachidChabane/shallnot/schemas"
)

const (
	commandCheck   = "check"
	commandGate    = "gate"
	commandHook    = "hook"
	commandInit    = "init"
	commandSchema  = "schema"
	commandVersion = "version"
	commandHelp    = "help"
)

const usage = `shallnot - a spec-to-test traceability gate

Usage:
  shallnot check [flags]      trace requirements to test results and give a verdict
  shallnot gate [flags]       run the configured test commands, then check their results
  shallnot init [flags]       equip a repository: starter config, agent instructions, end-of-turn hooks
  shallnot hook <harness>     answer an agent's end-of-turn hook: claude-stop, cursor-stop
  shallnot schema <name>      print a JSON Schema: report, config or spec
  shallnot version            print the version
  shallnot help               print this help

Run "shallnot check --help" for the flags; gate takes the same ones.

Exit codes:
  0  clean: no finding reached the blocking severity (or the run is advisory)
  1  blocked: at least one finding reached the blocking severity
  2  tool failure: no verdict was produced (bad flags, bad config, unreadable input)
`

var schemasByName = map[string][]byte{
	"report": schemas.Report,
	"config": schemas.Config,
	"spec":   schemas.Spec,
}

// Main runs the command line and returns the process exit code.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) app.ExitCode {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return app.ExitToolFailure
	}
	switch args[0] {
	case commandCheck:
		return runCheck(args[1:], stdout, stderr)
	case commandGate:
		return runGate(args[1:], stdout, stderr)
	case commandInit:
		return runInit(args[1:], stdout, stderr)
	case commandHook:
		return runHook(args[1:], stdin, stdout, stderr)
	case commandSchema:
		return runSchema(args[1:], stdout, stderr)
	case commandVersion, "--version":
		fmt.Fprintf(stdout, "%s %s\n", version.Name, version.Version)
		return app.ExitClean
	case commandHelp, "--help", "-h":
		fmt.Fprint(stdout, usage)
		return app.ExitClean
	default:
		return fail(stderr, fmt.Errorf("unknown command %q; run \"shallnot help\"", args[0]))
	}
}

func runSchema(args []string, stdout, stderr io.Writer) app.ExitCode {
	if len(args) != 1 {
		return fail(stderr, fmt.Errorf("usage: shallnot schema report|config|spec"))
	}
	schema, known := schemasByName[args[0]]
	if !known {
		return fail(stderr, fmt.Errorf("unknown schema %q (expected report, config or spec)", args[0]))
	}
	if _, err := stdout.Write(schema); err != nil {
		return fail(stderr, err)
	}
	return app.ExitClean
}

func fail(stderr io.Writer, err error) app.ExitCode {
	fmt.Fprintf(stderr, "%s: error: %v\n", version.Name, err)
	return app.ExitToolFailure
}
