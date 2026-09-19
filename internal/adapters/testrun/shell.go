// Package testrun runs a project's test commands through the platform shell.
package testrun

import (
	"errors"
	"io"
	"os/exec"
	"runtime"
)

const windows = "windows"

// Shell runs commands with `sh -c`, or `cmd /C` on Windows.
type Shell struct{}

// Run executes one command line in a directory, sending its output to log.
// A non-zero exit status is reported through exitCode, not as an error: test
// runners exit non-zero when tests fail, which is a result, not a failure to run.
func (Shell) Run(commandLine, directory string, log io.Writer) (exitCode int, err error) {
	command := exec.Command("sh", "-c", commandLine)
	if runtime.GOOS == windows {
		command = exec.Command("cmd", "/C", commandLine)
	}
	command.Dir = directory
	command.Stdout = log
	command.Stderr = log
	err = command.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}
