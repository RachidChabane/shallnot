// Package app orchestrates a run: it resolves inputs, loads them through the
// adapters and hands them to the analysis.
package app

import "github.com/RachidChabane/shallnot/internal/domain"

// Settings are the fully resolved options of one run.
type Settings struct {
	IDPattern string
	Specs     []string
	Focus     []string
	FocusIDs  []string
	Tests     []string
	Results   []string
	Exclude   []string
	// TestCommands are the command lines `gate` runs before checking, in WorkDir.
	TestCommands []string
	// WorkDir is where test commands run: the config file's directory, or "" for the working directory.
	WorkDir            string
	UseDefaultExcludes bool
	Policy             domain.Policy
}

// DefaultSettings returns the options of a run that configures nothing.
func DefaultSettings() Settings {
	return Settings{
		IDPattern:          domain.DefaultIDPattern,
		UseDefaultExcludes: true,
		Policy: domain.Policy{
			Severities: map[domain.Category]domain.Severity{},
			FailOn:     domain.DefaultFailOn,
		},
	}
}

// Inputs records what a run actually read, for the report.
type Inputs struct {
	IDPattern    string
	SpecFiles    []string
	FocusFiles   []string
	FocusIDs     []string
	TestRoots    []string
	ResultsFiles []string
}

// FocusIsEverything reports whether the run demanded coverage of every known requirement.
func (i Inputs) FocusIsEverything() bool {
	return len(i.FocusFiles) == 0 && len(i.FocusIDs) == 0
}

// Outcome is a completed run.
type Outcome struct {
	Inputs   Inputs
	Policy   domain.Policy
	Analysis domain.Analysis
}

// ExitCode is the process exit status of a run.
type ExitCode int

const (
	// ExitClean means no finding reached the blocking severity, or the run was advisory.
	ExitClean ExitCode = 0
	// ExitBlocked means at least one finding reached the blocking severity.
	ExitBlocked ExitCode = 1
	// ExitToolFailure means the tool could not produce a verdict.
	ExitToolFailure ExitCode = 2
)

// ExitCode maps the verdict and the advisory mode to the process exit status.
func (o Outcome) ExitCode() ExitCode {
	if o.Analysis.Verdict == domain.VerdictFail && !o.Policy.Advisory {
		return ExitBlocked
	}
	return ExitClean
}
