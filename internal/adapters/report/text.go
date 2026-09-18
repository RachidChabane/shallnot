package report

import (
	"fmt"
	"strings"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
)

const advisoryNote = "advisory mode: the exit code is 0 whatever the verdict"

func verdictLine(outcome app.Outcome) string {
	line := strings.ToUpper(outcome.Analysis.Verdict.String())
	if outcome.Policy.Advisory {
		line += " (" + advisoryNote + ")"
	}
	return line
}

func formatLocation(location domain.Location) string {
	if location.Line == 0 {
		return location.File
	}
	return fmt.Sprintf("%s:%d", location.File, location.Line)
}

func testLabel(test domain.BoundTest) string {
	if test.ClassName == "" || test.ClassName == test.Name {
		return strings.TrimSpace(test.Name)
	}
	return strings.TrimSpace(test.ClassName) + " › " + strings.TrimSpace(test.Name)
}

func testOrigin(test domain.BoundTest) string {
	if test.Source != nil {
		return formatLocation(*test.Source)
	}
	return test.ResultsFile
}

func focusDescription(inputs app.Inputs) string {
	if inputs.FocusIsEverything() {
		return "every known requirement"
	}
	return strings.Join(append(append([]string(nil), inputs.FocusFiles...), inputs.FocusIDs...), ", ")
}

func coverageCounts(summary RequirementSummary) string {
	return fmt.Sprintf("%d covered, %d failed, %d skipped, %d not run, %d uncovered, %d non-testable",
		summary.Covered, summary.Failed, summary.Skipped, summary.NotRun, summary.Uncovered, summary.NonTestable)
}
