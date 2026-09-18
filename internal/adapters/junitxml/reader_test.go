package junitxml_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/junitxml"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

func read(t *testing.T, path string) []domain.TestCase {
	t.Helper()
	testCases, err := junitxml.Reader{Grammar: testkit.Grammar(t)}.ReadFile(path, filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return testCases
}

func TestReader(t *testing.T) {
	testCases := read(t, "testdata/shapes.xml")
	if len(testCases) != 9 {
		t.Fatalf("got %d test cases, want 9", len(testCases))
	}

	t.Run("keeps passed, failed, errored and skipped apart [verifies SN-20~1]", func(t *testing.T) {
		want := []domain.Outcome{domain.OutcomePassed, domain.OutcomeFailed, domain.OutcomeErrored, domain.OutcomeSkipped}
		for index, outcome := range want {
			if testCases[index].Outcome != outcome {
				t.Errorf("%s: outcome %s, want %s", testCases[index].Name, testCases[index].Outcome, outcome)
			}
		}
	})

	t.Run("reads test cases of nested suites with their suite name [verifies SN-20~1]", func(t *testing.T) {
		if testCases[0].Suite != "outer" || testCases[4].Suite != "inner" || testCases[0].ResultsFile != "testdata/shapes.xml" {
			t.Fatalf("got %+v and %+v", testCases[0], testCases[4])
		}
	})

	t.Run("reads tags from the name, the classname and the verifies property [verifies SN-21~1]", func(t *testing.T) {
		want := map[int][]domain.Ref{
			0: {testkit.Ref("REQ-1", 1)},
			4: {testkit.Ref("REQ-5", 1)},
			5: {testkit.Ref("REQ-6", 1), testkit.Ref("REQ-7", 2)},
			6: {testkit.Ref("REQ-8", 1)},
			8: nil,
		}
		for index, refs := range want {
			if !reflect.DeepEqual(testCases[index].Refs, refs) {
				t.Errorf("%s: refs %v, want %v", testCases[index].Name, testCases[index].Refs, refs)
			}
		}
		if len(testCases[7].Malformed) != 1 || len(testCases[7].Refs) != 0 {
			t.Errorf("got %+v", testCases[7])
		}
	})

	for name, text := range map[string]string{
		"an empty file":      "",
		"another XML format": "<coverage/>",
		"broken XML":         "<testsuite><testcase></testsuite>",
		"plain text":         "1 passed",
	} {
		t.Run("refuses "+name+" [verifies SN-23~1]", func(t *testing.T) {
			if _, err := (junitxml.Reader{Grammar: testkit.Grammar(t)}).Read(strings.NewReader(text), "junit.xml"); err == nil {
				t.Fatalf("%q was accepted", text)
			}
		})
	}
}

// outcomes counts the test cases of a results directory by outcome.
func outcomes(t *testing.T, pattern string) (map[domain.Outcome]int, int) {
	t.Helper()
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		t.Fatalf("no results match %s (%v)", pattern, err)
	}
	counts := map[domain.Outcome]int{}
	bound := 0
	for _, file := range files {
		for _, testCase := range read(t, file) {
			counts[testCase.Outcome]++
			if len(testCase.Refs) > 0 {
				bound++
			}
		}
	}
	return counts, bound
}

func TestRealRunnerOutput(t *testing.T) {
	const fixtures = "../../../fixtures/cart/"
	runners := []struct {
		name, results                  string
		passed, failed, skipped, bound int
	}{
		{"pytest with junit_family=xunit1", "pytest/results/junit.xml", 14, 1, 1, 13},
		{"pytest with junit_family=xunit2", "pytest/results/junit-xunit2.xml", 14, 1, 1, 13},
		{"Jest with jest-junit", "jest/results/junit.xml", 11, 1, 1, 11},
		{"Vitest", "vitest/results/junit.xml", 11, 1, 1, 11},
		{"JUnit 5 under Maven Surefire", "junit5-maven/results/*.xml", 10, 1, 1, 10},
		{"JUnit 5 under Gradle", "junit5-gradle/results/*.xml", 10, 1, 1, 6},
		{"JUnit 5 in Kotlin under Maven Surefire", "junit5-kotlin/results/*.xml", 11, 1, 1, 11},
	}
	for _, runner := range runners {
		t.Run("reads the results of "+runner.name+" [verifies SN-22~1]", func(t *testing.T) {
			counts, bound := outcomes(t, fixtures+runner.results)
			got := []int{counts[domain.OutcomePassed], counts[domain.OutcomeFailed], counts[domain.OutcomeSkipped], bound}
			want := []int{runner.passed, runner.failed, runner.skipped, runner.bound}
			if !reflect.DeepEqual(got, want) || counts[domain.OutcomeErrored] != 0 {
				t.Fatalf("passed/failed/skipped/bound = %v, want %v (errored %d)", got, want, counts[domain.OutcomeErrored])
			}
		})
	}
}

func FuzzReader(f *testing.F) {
	seed, err := os.ReadFile("testdata/shapes.xml")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(seed))
	ids, err := domain.NewIDPattern(domain.DefaultIDPattern)
	if err != nil {
		f.Fatal(err)
	}
	reader := junitxml.Reader{Grammar: domain.TagGrammar{IDs: ids}}
	f.Fuzz(func(t *testing.T, text string) {
		testCases, err := reader.Read(strings.NewReader(text), "fuzz.xml")
		if err != nil {
			return
		}
		for _, testCase := range testCases {
			if testCase.Outcome == domain.OutcomeNotRun {
				t.Fatalf("a results file yielded a not_run test: %+v", testCase)
			}
		}
	})
}
