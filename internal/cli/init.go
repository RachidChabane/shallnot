package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RachidChabane/shallnot/internal/adapters/scaffold"
	"github.com/RachidChabane/shallnot/internal/app"
)

const (
	hooksAuto = "auto"
	hooksNone = "none"
)

// runInit equips a repository: starter config, agent instructions, end-of-turn hooks.
func runInit(args []string, stdout, stderr io.Writer) app.ExitCode {
	set := flag.NewFlagSet("shallnot init", flag.ContinueOnError)
	set.SetOutput(stderr)
	directory := set.String("dir", ".", "project `directory` to equip")
	hooks := set.String("hooks", hooksAuto, "end-of-turn hooks to install: a comma-separated `list` of "+strings.Join(scaffold.HookHarnesses(), ", ")+
		"; \""+hooksAuto+"\" installs claude, and cursor when the project has a .cursor directory; \""+hooksNone+"\" installs none")
	check := set.Bool("check", false, "change nothing; exit 1 if init would change a file")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return app.ExitClean
		}
		return app.ExitToolFailure
	}
	if set.NArg() > 0 {
		return fail(stderr, fmt.Errorf("unexpected argument %q", set.Arg(0)))
	}
	plan, err := scaffold.Build(scaffold.Options{Directory: *directory, Hooks: hookHarnesses(*directory, *hooks)})
	if err != nil {
		return fail(stderr, err)
	}
	for _, action := range plan.Actions {
		fmt.Fprintf(stdout, "%-9s %s\n", action.Change, filepath.ToSlash(action.Path))
	}
	if *check {
		if plan.Pending() {
			return app.ExitBlocked
		}
		return app.ExitClean
	}
	if err := plan.Apply(); err != nil {
		return fail(stderr, err)
	}
	for _, step := range plan.NextSteps {
		fmt.Fprintf(stdout, "next: %s\n", step)
	}
	return app.ExitClean
}

func hookHarnesses(directory, selection string) []string {
	switch selection {
	case hooksNone:
		return nil
	case hooksAuto:
		harnesses := []string{"claude"}
		if _, err := os.Stat(filepath.Join(directory, ".cursor")); err == nil {
			harnesses = append(harnesses, "cursor")
		}
		return harnesses
	default:
		return strings.Split(selection, ",")
	}
}
