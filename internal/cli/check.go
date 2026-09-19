package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RachidChabane/shallnot/internal/adapters/config"
	"github.com/RachidChabane/shallnot/internal/adapters/report"
	"github.com/RachidChabane/shallnot/internal/adapters/testrun"
	"github.com/RachidChabane/shallnot/internal/app"
)

// stringList is a flag that may be repeated.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

type checkFlags struct {
	configPath        string
	noConfig          bool
	idPattern         string
	specs             stringList
	focus             stringList
	focusIDs          stringList
	tests             stringList
	results           stringList
	exclude           stringList
	testCommands      stringList
	severities        stringList
	noDefaultExcludes bool
	advisory          bool
	strict            bool
	failOn            string
	format            string
	jsonOut           string
	markdownOut       string
	githubAnnotations bool
}

func newCheckFlagSet(command string, flags *checkFlags, stderr io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet("shallnot "+command, flag.ContinueOnError)
	set.SetOutput(stderr)
	set.StringVar(&flags.configPath, "config", "", "configuration `file` (default: "+config.DefaultFileName+" in the working directory, if present)")
	set.BoolVar(&flags.noConfig, "no-config", false, "ignore "+config.DefaultFileName+" in the working directory")
	set.StringVar(&flags.idPattern, "id-pattern", "", "regular `expression` a requirement ID must match")
	set.Var(&flags.specs, "specs", "known specs: a file, directory or `glob` (repeatable)")
	set.Var(&flags.focus, "focus", "specs whose requirements this run must find covered: a file, directory or `glob` (repeatable)")
	set.Var(&flags.focusIDs, "focus-id", "`glob` on requirement IDs this run must find covered (repeatable)")
	set.Var(&flags.tests, "tests", "test source root `directory` scanned for tags (repeatable)")
	set.Var(&flags.results, "results", "JUnit XML results: a file, directory or `glob` (repeatable)")
	set.Var(&flags.exclude, "exclude", "`glob` of source paths the scan ignores (repeatable)")
	set.Var(&flags.testCommands, "test-command", "command `line` that `gate` runs before checking (repeatable)")
	set.BoolVar(&flags.noDefaultExcludes, "no-default-excludes", false, "scan dependency and build directories too")
	set.Var(&flags.severities, "severity", "override as `category=severity` (repeatable)")
	set.BoolVar(&flags.advisory, "advisory", false, "report everything but always exit 0")
	set.BoolVar(&flags.strict, "strict", false, "treat untagged tests as errors")
	set.StringVar(&flags.failOn, "fail-on", "", "lowest blocking `severity`: error, warning or info")
	set.StringVar(&flags.format, "format", report.FormatTerminal.String(), "format printed on standard output: "+strings.Join(report.FormatNames(), ", "))
	set.StringVar(&flags.jsonOut, "json-out", "", "also write the JSON report to this `file`")
	set.StringVar(&flags.markdownOut, "markdown-out", "", "also write the Markdown summary to this `file`")
	set.BoolVar(&flags.githubAnnotations, "github-annotations", false, "also print GitHub Actions annotations on standard output")
	return set
}

// verdictFunc produces the outcome of a command that ends in a verdict.
type verdictFunc func(settings app.Settings, log io.Writer) (app.Outcome, error)

func runCheck(args []string, stdout, stderr io.Writer) app.ExitCode {
	return runVerdict(commandCheck, args, stdout, stderr, func(settings app.Settings, _ io.Writer) (app.Outcome, error) {
		return app.Run(settings)
	})
}

func runGate(args []string, stdout, stderr io.Writer) app.ExitCode {
	return runVerdict(commandGate, args, stdout, stderr, func(settings app.Settings, log io.Writer) (app.Outcome, error) {
		return app.Gate(settings, testrun.Shell{}, log)
	})
}

func runVerdict(command string, args []string, stdout, stderr io.Writer, verdict verdictFunc) app.ExitCode {
	var flags checkFlags
	set := newCheckFlagSet(command, &flags, stderr)
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return app.ExitClean
		}
		return app.ExitToolFailure
	}
	if set.NArg() > 0 {
		return fail(stderr, fmt.Errorf("unexpected argument %q", set.Arg(0)))
	}
	if err := removeStaleReports(flags); err != nil {
		return fail(stderr, err)
	}
	format, err := report.ParseFormat(flags.format)
	if err != nil {
		return fail(stderr, err)
	}
	settings, err := resolveSettings(flags)
	if err != nil {
		return fail(stderr, err)
	}
	outcome, err := verdict(settings, stderr)
	if err != nil {
		return fail(stderr, err)
	}
	if err := writeReports(flags, format, outcome, stdout); err != nil {
		return fail(stderr, err)
	}
	return outcome.ExitCode()
}

// removeStaleReports deletes report files of an earlier run, so that a tool
// failure never leaves a verdict behind for a consumer to read.
func removeStaleReports(flags checkFlags) error {
	for _, path := range []string{flags.jsonOut, flags.markdownOut} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// resolveSettings layers the flags over the configuration file over the defaults.
func resolveSettings(flags checkFlags) (app.Settings, error) {
	settings, err := loadConfig(flags)
	if err != nil {
		return settings, err
	}
	if flags.idPattern != "" {
		settings.IDPattern = flags.idPattern
	}
	override := func(target *[]string, values stringList) {
		if len(values) > 0 {
			*target = values
		}
	}
	override(&settings.Specs, flags.specs)
	override(&settings.Focus, flags.focus)
	override(&settings.FocusIDs, flags.focusIDs)
	override(&settings.Tests, flags.tests)
	override(&settings.Results, flags.results)
	override(&settings.Exclude, flags.exclude)
	override(&settings.TestCommands, flags.testCommands)
	if flags.noDefaultExcludes {
		settings.UseDefaultExcludes = false
	}
	if flags.advisory {
		settings.Policy.Advisory = true
	}
	if flags.strict {
		config.ApplyStrict(&settings.Policy)
	}
	if flags.failOn != "" {
		if err := config.SetFailOn(&settings.Policy, flags.failOn); err != nil {
			return settings, err
		}
	}
	for _, assignment := range flags.severities {
		category, severity, found := strings.Cut(assignment, "=")
		if !found {
			return settings, fmt.Errorf("--severity %q: expected category=severity", assignment)
		}
		if err := config.SetSeverity(&settings.Policy, category, severity); err != nil {
			return settings, err
		}
	}
	return settings, nil
}

func loadConfig(flags checkFlags) (app.Settings, error) {
	if flags.configPath != "" && flags.noConfig {
		return app.Settings{}, errors.New("--config and --no-config exclude each other")
	}
	if flags.configPath != "" {
		return config.Load(flags.configPath)
	}
	if _, err := os.Stat(config.DefaultFileName); err == nil && !flags.noConfig {
		return config.Load(config.DefaultFileName)
	}
	return app.DefaultSettings(), nil
}

func writeReports(flags checkFlags, format report.Format, outcome app.Outcome, stdout io.Writer) error {
	if err := report.For(format).Write(stdout, outcome); err != nil {
		return err
	}
	if flags.githubAnnotations && format != report.FormatGitHub {
		if err := (report.GitHubReporter{}).Write(stdout, outcome); err != nil {
			return err
		}
	}
	if err := writeFile(flags.jsonOut, report.JSONReporter{}, outcome); err != nil {
		return err
	}
	return writeFile(flags.markdownOut, report.MarkdownReporter{}, outcome)
}

func writeFile(path string, reporter report.Reporter, outcome app.Outcome) error {
	if path == "" {
		return nil
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := reporter.Write(file, outcome); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
