package report

import (
	"io"
	"strings"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
)

// MarkdownReporter writes a summary suited to a GitHub step summary.
type MarkdownReporter struct{}

var cellEscaper = strings.NewReplacer("|", "\\|", "\n", " ", "\r", "")

func cell(text string) string { return cellEscaper.Replace(text) }

// Write prints the verdict, the matrix of requirements in focus, and the findings as tables.
func (MarkdownReporter) Write(writer io.Writer, outcome app.Outcome) error {
	summary := Summarize(outcome.Analysis)
	out := &errWriter{writer: writer}
	out.printf("## shallnot: %s\n\n", verdictLine(outcome))
	out.printf("- **Focus:** %s\n", focusDescription(outcome.Inputs))
	if outcome.Inputs.Commit != "" {
		out.printf("- **Commit:** `%s`\n", outcome.Inputs.Commit)
	}
	out.printf("- **Requirements:** %d known, %d in focus (%s)\n", summary.Requirements.Known, summary.Requirements.InFocus, coverageCounts(summary.Requirements))
	out.printf("- **Tests:** %d in results, %d bound, %d untagged\n", summary.Tests.Total, summary.Tests.Bound, summary.Tests.Untagged)
	out.printf("- **Findings:** %d error, %d warning, %d info (%d blocking)\n\n", summary.Findings.Error, summary.Findings.Warning, summary.Findings.Info, summary.Findings.Blocking)

	out.printf("### Requirements in focus\n\n| Requirement | Coverage | Bound tests | Declared at |\n|---|---|---|---|\n")
	for _, row := range outcome.Analysis.Rows {
		if !row.InFocus {
			continue
		}
		out.printf("| `%s` | %s | %s | `%s` |\n", row.Requirement.Ref(), row.Coverage, cell(outcomeCounts(row.Tests)), cell(formatLocation(row.Requirement.Location)))
	}

	if len(outcome.Analysis.Findings) > 0 {
		out.printf("\n### Findings\n\n| Severity | Category | Location | Message |\n|---|---|---|---|\n")
		for _, finding := range outcome.Analysis.Findings {
			out.printf("| %s | `%s` | `%s` | %s |\n", finding.Severity, finding.Category, cell(formatLocation(finding.Location)), cell(finding.Message))
		}
	}
	return out.err
}

// outcomeCounts renders "2 passed, 1 failed" in the declaration order of outcomes.
func outcomeCounts(tests []domain.BoundTest) string {
	if len(tests) == 0 {
		return "none"
	}
	counts := map[domain.Outcome]int{}
	for _, test := range tests {
		counts[test.Outcome]++
	}
	var parts []string
	for _, name := range domain.OutcomeNames() {
		var outcome domain.Outcome
		if err := outcome.UnmarshalText([]byte(name)); err != nil {
			continue
		}
		if counts[outcome] > 0 {
			parts = append(parts, itoa(counts[outcome])+" "+strings.ReplaceAll(name, "_", " "))
		}
	}
	return strings.Join(parts, ", ")
}
