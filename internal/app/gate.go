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

// resultsState records the modification time of the results files that exist
// before the test commands run. Patterns that match nothing yet are expected.
func resultsState(settings Settings) map[string]time.Time {
	state := map[string]time.Time{}
	resultsFiles, err := resolveResults(settings)
	if err != nil {
		return state
	}
	for _, file := range resultsFiles {
		if info, err := os.Stat(file.path); err == nil {
			state[file.absolute] = info.ModTime()
		}
	}
	return state
}

// requireFreshResults refuses a results file that the test commands left untouched.
func requireFreshResults(settings Settings, before map[string]time.Time) error {
	resultsFiles, err := resolveResults(settings)
	if err != nil {
		return err
	}
	for _, file := range resultsFiles {
		info, err := os.Stat(file.path)
		if err != nil {
			return err
		}
		if previous, existed := before[file.absolute]; existed && info.ModTime().Equal(previous) {
			return fmt.Errorf("results file %s was not written by this run (last modified %s): no test command produced it",
				file.displayPath, info.ModTime().Format(time.RFC3339))
		}
	}
	return nil
}
