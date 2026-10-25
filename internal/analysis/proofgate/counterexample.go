package proofgate

import (
	"fmt"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// Words of a counterexample in a finding.
const (
	witnessIndent  = "    "
	witnessCutNote = "(the witness is cut there)"
)

// slotRoles name what a generation slot of a targeted mutant asks for.
var slotRoles = map[string]string{"S1": "direct", "S2": "boundary", "S3": "partial"}

// counterexampleText is what the finding of a blocking survivor adds under local
// trust when its local statement records a counterexample: how the mutant was
// drawn and the input of the criterion that every test passes on while the
// mutant fails it, one of the criterion's examples as its statement writes it or
// a witness test written from the spec alone. It never shows the mutant's code,
// and adds nothing under signed trust or for a statement produced elsewhere.
func (p policy) counterexampleText(criterion Criterion, predicate *proof.DiscriminatingPower, mutant proof.Mutant) string {
	if p.config.Trust != proof.TrustLocal || predicate.ProducedBy != proof.ProducedByLocal || mutant.Extensions == nil || mutant.Extensions.Counterexample == nil {
		return ""
	}
	counterexample := mutant.Extensions.Counterexample
	drawn, known := drawnAs(*counterexample)
	if !known {
		return ""
	}
	head := fmt.Sprintf("; every test passes on %s that ", drawn)
	switch {
	case counterexample.Example != nil:
		for _, example := range criterion.Examples {
			if example.Index == *counterexample.Example {
				return head + fmt.Sprintf("fails the criterion's example %d: %s", example.Index, example)
			}
		}
		return head + fmt.Sprintf("fails the criterion's example %d", *counterexample.Example)
	case counterexample.Witness != nil:
		witness := counterexample.Witness
		lines := []string{head + fmt.Sprintf("fails the witness %s, which checks %q:", witness.Name, witness.Clause)}
		for _, line := range strings.Split(strings.TrimRight(witness.Code, "\n"), "\n") {
			lines = append(lines, witnessIndent+line)
		}
		if witness.Truncated {
			lines = append(lines, witnessIndent+witnessCutNote)
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

// drawnAs says how the mutant of a counterexample was drawn, and whether the
// counterexample names a known origin.
func drawnAs(counterexample proof.Counterexample) (string, bool) {
	switch counterexample.Origin {
	case proof.OriginTargeted:
		role, known := slotRoles[counterexample.Slot]
		return fmt.Sprintf("a targeted mutant of slot %s (%s)", counterexample.Slot, role), known
	case proof.OriginOperator:
		return fmt.Sprintf("an operator mutant of family %s", counterexample.Family), counterexample.Family != ""
	}
	return "", false
}
