package report

import (
	"io"
	"strconv"
	"strings"

	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/version"
)

// GitHubReporter writes findings as GitHub Actions workflow commands, which
// GitHub renders as annotations on the offending lines.
type GitHubReporter struct{}

var annotationLevels = map[domain.Severity]string{
	domain.SeverityError:   "error",
	domain.SeverityWarning: "warning",
	domain.SeverityInfo:    "notice",
}

var (
	annotationData     = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	annotationProperty = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
)

func itoa(number int) string { return strconv.Itoa(number) }

// Write prints one workflow command per finding.
func (GitHubReporter) Write(writer io.Writer, outcome app.Outcome) error {
	out := &errWriter{writer: writer}
	for _, finding := range outcome.Analysis.Findings {
		properties := "file=" + annotationProperty.Replace(finding.Location.File)
		if finding.Location.Line > 0 {
			properties += ",line=" + itoa(finding.Location.Line)
		}
		properties += ",title=" + annotationProperty.Replace(version.Name+" "+finding.Category.String())
		out.printf("::%s %s::%s\n", annotationLevels[finding.Severity], properties, annotationData.Replace(finding.Message))
	}
	return out.err
}
