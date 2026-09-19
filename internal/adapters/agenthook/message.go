package agenthook

import (
	"fmt"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const prohibitions = "Never delete or alter a tag, mark a requirement non-testable, skip a test or weaken an assertion to clear a finding. " +
	"If a requirement cannot be met or tested as written, stop and say so."

// BlockedMessage tells the agent which findings block the gate.
func BlockedMessage(findings []domain.Finding) string {
	var blocking []string
	for _, finding := range findings {
		if !finding.Blocking {
			continue
		}
		location := finding.Location.File
		if finding.Location.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, finding.Location.Line)
		}
		blocking = append(blocking, fmt.Sprintf("- %s at %s: %s", finding.Category, location, finding.Message))
	}
	return fmt.Sprintf("shallnot: the traceability gate is blocked by %d finding(s). Resolve them before finishing:\n%s\n%s\n",
		len(blocking), strings.Join(blocking, "\n"), prohibitions)
}

// FailureMessage tells the agent that the gate produced no verdict.
func FailureMessage(err error) string {
	return fmt.Sprintf("shallnot: the traceability gate could not produce a verdict: %v\n"+
		"The test commands of shallnot.yaml must run and write the results files it lists. Fix that before finishing. "+
		"If the specs themselves are missing, stop and tell the user; never write requirements yourself.\n", err)
}

// GiveUpMessage tells the user that the agent stopped with the gate still blocked.
func GiveUpMessage(attempts int, last string) string {
	return fmt.Sprintf("shallnot: the gate is still blocked after %d attempts; run `shallnot gate` to see why.\n%s", attempts, last)
}
