package report

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/RachidChabane/shallnot/internal/app"
)

// TerminalReporter writes the report an engineer reads at a terminal.
type TerminalReporter struct{}

// Write prints the verdict, the matrix of requirements in focus, and the findings.
func (TerminalReporter) Write(writer io.Writer, outcome app.Outcome) error {
	summary := Summarize(outcome.Analysis)
	out := &errWriter{writer: writer}
	out.printf("shallnot: %s\n", verdictLine(outcome))
	out.printf("focus: %s\n", focusDescription(outcome.Inputs))
	if outcome.Inputs.Commit != "" {
		out.printf("commit: %s\n", outcome.Inputs.Commit)
	}
	out.printf("requirements: %d known, %d in focus (%s)\n", summary.Requirements.Known, summary.Requirements.InFocus, coverageCounts(summary.Requirements))
	out.printf("tests: %d in results, %d bound, %d untagged\n", summary.Tests.Total, summary.Tests.Bound, summary.Tests.Untagged)
	out.printf("findings: %d error, %d warning, %d info (%d blocking)\n\n", summary.Findings.Error, summary.Findings.Warning, summary.Findings.Info, summary.Findings.Blocking)

	out.printf("REQUIREMENTS\n")
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, row := range outcome.Analysis.Rows {
		if !row.InFocus {
			continue
		}
		// A line without tabs ends the aligned block, so each requirement's tests align among themselves.
		fmt.Fprintf(table, "  %s  %s  %s\n", row.Requirement.Ref(), row.Coverage, formatLocation(row.Requirement.Location))
		for _, test := range row.Tests {
			fmt.Fprintf(table, "      %s\t%s\t%s\n", test.Outcome, testLabel(test), testOrigin(test))
		}
	}
	out.keep(table.Flush())

	if len(outcome.Analysis.Findings) > 0 {
		out.printf("\nFINDINGS\n")
		table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		for _, finding := range outcome.Analysis.Findings {
			fmt.Fprintf(table, "  %s\t%s\t%s\t%s\n", finding.Severity, finding.Category, formatLocation(finding.Location), finding.Message)
		}
		out.keep(table.Flush())
	}
	return out.err
}

// errWriter remembers the first write error so that callers check once.
type errWriter struct {
	writer io.Writer
	err    error
}

func (w *errWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	written, err := w.writer.Write(data)
	w.err = err
	return written, err
}

func (w *errWriter) printf(format string, args ...any) {
	fmt.Fprintf(w, format, args...)
}

func (w *errWriter) keep(err error) {
	if w.err == nil {
		w.err = err
	}
}
