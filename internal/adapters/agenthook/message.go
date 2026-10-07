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

// PassedMessage tells the user that the gate passes, once it had held the agent earlier in the session.
func PassedMessage(covered, nonTestable int) string {
	return fmt.Sprintf("shallnot: gate passed: %d requirement(s) in focus covered by passing tests, %d non-testable, no blocking finding.\n", covered, nonTestable)
}

const (
	pendingHeader  = "shallnot: the proof gate is waiting for %d proof(s) that a spec-proof module produces through the host harness:\n%s\n"
	pendingFulfil  = "Run the fulfil skill each finding names (or spec-proof:prove-all) and finish when it reports that the requests are answered. Never answer the requests yourself.\n"
	pendingStalled = "A module's host did not answer its requests: report this to the user, do not loop.\n"
)

// PendingMessage tells the agent that the gate waits for proofs that a module's host has to produce, and what to run.
func PendingMessage(findings []domain.Finding) string {
	var waiting []string
	unresponsive := false
	for _, finding := range findings {
		if !finding.Blocking {
			continue
		}
		unresponsive = unresponsive || finding.Category == domain.CategoryHostUnresponsive
		waiting = append(waiting, fmt.Sprintf("- %s at %s: %s", finding.Category, locationOf(finding), finding.Message))
	}
	message := fmt.Sprintf(pendingHeader, len(waiting), strings.Join(waiting, "\n"))
	if unresponsive {
		return message + pendingStalled
	}
	return message + pendingFulfil
}

func locationOf(finding domain.Finding) string {
	if finding.Location.Line > 0 {
		return fmt.Sprintf("%s:%d", finding.Location.File, finding.Location.Line)
	}
	return finding.Location.File
}

// OnlyPending reports whether the blocking findings are all pending proofs, and at least one blocks.
func OnlyPending(findings []domain.Finding) bool {
	blocking := 0
	for _, finding := range findings {
		if !finding.Blocking {
			continue
		}
		if !finding.Category.IsPending() {
			return false
		}
		blocking++
	}
	return blocking > 0
}
