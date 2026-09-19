// Package scaffold equips a repository for shallnot: a starter configuration
// for the test runners it finds, the agent instructions, and the end-of-turn
// hooks. Every file it writes is derived from the project's current state, so
// running it again changes nothing.
package scaffold

import (
	"os"
	"path/filepath"
	"strings"
)

// Runner is a test runner that init knows how to configure.
type Runner struct {
	Name        string
	TestCommand string
	Results     string
	// TestRoots are the candidate source roots, in order of preference; the ones that exist are scanned.
	TestRoots []string
	// NextStep is what the project must still do for tags to reach the results.
	NextStep string
	detect   func(directory string) bool
}

const resultsDirectory = "test-results"

var runners = []Runner{
	{
		Name:        "pytest",
		TestCommand: "pytest -o junit_family=xunit1 --junitxml=" + resultsDirectory + "/pytest.xml",
		Results:     resultsDirectory + "/pytest.xml",
		TestRoots:   []string{"tests", "test"},
		detect: func(directory string) bool {
			return exists(directory, "pytest.ini") || exists(directory, "conftest.py") || exists(directory, "tox.ini") ||
				fileContains(directory, "pyproject.toml", "pytest") || fileContains(directory, "setup.cfg", "pytest") ||
				fileContains(directory, "requirements.txt", "pytest")
		},
	},
	{
		Name:        "vitest",
		TestCommand: "npx vitest run --reporter=default --reporter=junit --outputFile.junit=" + resultsDirectory + "/vitest.xml",
		Results:     resultsDirectory + "/vitest.xml",
		TestRoots:   []string{"src", "test", "tests", "__tests__"},
		detect:      func(directory string) bool { return fileContains(directory, "package.json", `"vitest"`) },
	},
	{
		Name:        "jest",
		TestCommand: "npx jest --reporters=default --reporters=jest-junit",
		Results:     "junit.xml",
		TestRoots:   []string{"src", "test", "tests", "__tests__"},
		NextStep:    "Jest: install the reporter with `npm install --save-dev jest-junit`; it writes junit.xml in the project root.",
		detect:      func(directory string) bool { return fileContains(directory, "package.json", `"jest"`) },
	},
	{
		Name:        "maven",
		TestCommand: "mvn -B test",
		Results:     "target/surefire-reports/*.xml",
		TestRoots:   []string{"src/test"},
		NextStep:    "Maven: make Surefire report display names, or tags in @DisplayName never reach the results; see the Maven Surefire section of docs/binding.md.",
		detect:      func(directory string) bool { return exists(directory, "pom.xml") },
	},
	{
		Name:        "gradle",
		TestCommand: "./gradlew test",
		Results:     "build/test-results/test/*.xml",
		TestRoots:   []string{"src/test"},
		detect: func(directory string) bool {
			return exists(directory, "build.gradle") || exists(directory, "build.gradle.kts")
		},
	},
	{
		Name:        "go",
		TestCommand: "go run gotest.tools/gotestsum@latest --junitfile " + resultsDirectory + "/go.xml ./...",
		Results:     resultsDirectory + "/go.xml",
		TestRoots:   []string{"."},
		detect:      func(directory string) bool { return exists(directory, "go.mod") },
	},
}

// DetectRunners returns the test runners a project uses.
func DetectRunners(directory string) []Runner {
	var found []Runner
	for _, runner := range runners {
		if runner.detect(directory) {
			found = append(found, runner)
		}
	}
	return found
}

func exists(directory, name string) bool {
	_, err := os.Stat(filepath.Join(directory, name))
	return err == nil
}

func fileContains(directory, name, text string) bool {
	content, err := os.ReadFile(filepath.Join(directory, name))
	return err == nil && strings.Contains(string(content), text)
}
