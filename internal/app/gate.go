package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// TestRunner runs one test command line in a directory.
type TestRunner interface {
	Run(commandLine, directory string, log io.Writer) (exitCode int, err error)
}

// Gate runs the project's test commands, then the traceability check on the
// results that run produced. Results older than the run are refused: a runner
// that crashed before writing its report must not leave an earlier verdict.
func Gate(settings Settings, runner TestRunner, log io.Writer) (Outcome, error) {
	if len(settings.TestCommands) == 0 {
		return Outcome{}, errors.New("no test command given: pass --test-command or set `test_commands` in the config file")
	}
	before := resultsState(settings)
	for _, commandLine := range settings.TestCommands {
		fmt.Fprintf(log, "shallnot: running %s\n", commandLine)
		exitCode, err := runner.Run(commandLine, settings.WorkDir, log)
		if err != nil {
			return Outcome{}, fmt.Errorf("test command %q could not run: %w", commandLine, err)
		}
		fmt.Fprintf(log, "shallnot: %q exited with status %d\n", commandLine, exitCode)
	}
	if err := requireFreshResults(settings, before); err != nil {
		return Outcome{}, err
	}
	return Run(settings)
}

// staleStamp is the modification time given to existing results files before
// the test commands run. A file still bearing it afterwards was not rewritten.
// Comparing a file's time before and after the run is not enough: tools that
// preserve times (`cp -p`, Windows `copy`) can rewrite a file and leave it equal.
var staleStamp = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// resultsMark is how one existing results file was marked before the run.
type resultsMark struct {
	// stamped is true when the file received staleStamp; otherwise previous is its untouched modification time.
	stamped  bool
	previous time.Time
}

// resultsState marks the results files that exist before the test commands
// run. Patterns that match nothing yet are expected.
func resultsState(settings Settings) map[string]resultsMark {
	state := map[string]resultsMark{}
	resultsFiles, err := resolveResults(settings)
	if err != nil {
		return state
	}
	for _, file := range resultsFiles {
		info, err := os.Stat(file.path)
		if err != nil {
			continue
		}
		// A file that cannot be stamped (read-only, another owner) falls back to comparing times.
		stamped := os.Chtimes(file.path, staleStamp, staleStamp) == nil
		state[file.absolute] = resultsMark{stamped: stamped, previous: info.ModTime()}
	}
	return state
}

// requireFreshResults refuses a results file that the test commands left untouched.
func requireFreshResults(settings Settings, before map[string]resultsMark) error {
	resultsFiles, err := resolveResults(settings)
	if err != nil {
		return err
	}
	for _, file := range resultsFiles {
		info, err := os.Stat(file.path)
		if err != nil {
			return err
		}
		mark, existed := before[file.absolute]
		if !existed {
			continue
		}
		untouched := info.ModTime().Equal(mark.previous)
		if mark.stamped {
			untouched = info.ModTime().Equal(staleStamp)
		}
		if untouched {
			return fmt.Errorf("results file %s was not written by this run: no test command produced it", file.displayPath)
		}
	}
	return nil
}
