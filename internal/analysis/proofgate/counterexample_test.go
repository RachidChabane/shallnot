package proofgate_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

// boxExample is the example of the criterion a survivor fails.
var boxExample = domain.Example{
	Index: 1, Operation: "ticket_price", Args: []domain.Argument{{Name: "age", Value: domain.Literal{Type: domain.LiteralInteger, Value: "65"}}},
	Expect: domain.ExpectedOutcome{Kind: domain.OutcomeReturns, Value: domain.Literal{Type: domain.LiteralInteger, Value: "8"}},
}

// withCounterexample gives a mutant a counterexample.
func withCounterexample(mutant proof.Mutant, counterexample proof.Counterexample) proof.Mutant {
	if mutant.Extensions == nil {
		mutant.Extensions = &proof.MutantExtensions{}
	}
	mutant.Extensions.Counterexample = &counterexample
	return mutant
}

// survivorMessage evaluates a pool holding the survivor under the trust and the
// producer, and returns the message of its surviving_valid_mutant finding.
func survivorMessage(t *testing.T, trust proof.Trust, producedBy proof.ProducedBy, mutant proof.Mutant) string {
	t.Helper()
	statement := withCredit(mutant)
	statement.DiscriminatingPower.ProducedBy = producedBy
	ev := evidence(statement)
	ev.Manifest.Trust = trust
	ev.Statements = loaded(prooftest.Seal(statement))
	cfg := config()
	if trust == proof.TrustLocal {
		cfg = config(local)
	}
	withExamples := criterion()
	withExamples.Examples = []domain.Example{boxExample}
	result := proofgate.Evaluate(proofgate.Input{Config: cfg, Known: known(), Criteria: []proofgate.Criterion{withExamples}, Scope: scope(), Evidence: ev})
	for _, finding := range result.Findings {
		if finding.Category == domain.CategorySurvivingValidMutant {
			return finding.Message
		}
	}
	t.Fatalf("no surviving_valid_mutant finding: %v", messages(result))
	return ""
}

func TestCounterexample(t *testing.T) {
	example := 1
	exampleSurvivor := withCounterexample(survivor("s", valid(lowConfidence, proof.CorroborationExampleFailed)),
		proof.Counterexample{Origin: proof.OriginTargeted, Slot: "S1", Example: &example})
	witnessSurvivor := withCounterexample(survivor("s", valid(lowConfidence, proof.CorroborationExampleFailed)),
		proof.Counterexample{Origin: proof.OriginOperator, Family: "relational", Witness: &proof.WitnessExample{
			Name: "senior_at_sixty_five", Clause: "65 or older", Code: "def senior_at_sixty_five():\n    assert ticket_price(age=65) == 8\n", Truncated: true,
		}})

	t.Run("names how the mutant was drawn and the criterion's example it fails, as the statement writes it [verifies SN-139~1]", func(t *testing.T) {
		message := survivorMessage(t, proof.TrustLocal, proof.ProducedByLocal, exampleSurvivor)
		want := "; every test passes on a targeted mutant of slot S1 (direct) that fails the criterion's example 1: ticket_price(age=65) => 8"
		if !strings.HasSuffix(message, want) || !strings.Contains(message, "violates it and survived every test") {
			t.Errorf("message %q", message)
		}
	})

	t.Run("shows a witness the mutant fails with its name, clause and code, and says it was cut [verifies SN-139~1]", func(t *testing.T) {
		message := survivorMessage(t, proof.TrustLocal, proof.ProducedByLocal, witnessSurvivor)
		for _, wanted := range []string{
			`every test passes on an operator mutant of family relational that fails the witness senior_at_sixty_five, which checks "65 or older":`,
			"\n    def senior_at_sixty_five():\n        assert ticket_price(age=65) == 8\n    (the witness is cut there)",
		} {
			if !strings.Contains(message, wanted) {
				t.Errorf("message %q lacks %q", message, wanted)
			}
		}
	})

	t.Run("adds nothing under signed trust, for a statement produced by CI, or for an unknown origin [verifies SN-139~1]", func(t *testing.T) {
		bare := "requirement " + requirementID + "~1: mutant "
		unknown := withCounterexample(survivor("s", valid(lowConfidence, proof.CorroborationExampleFailed)), proof.Counterexample{Origin: "promised", Example: &example})
		for name, message := range map[string]string{
			"signed trust":   survivorMessage(t, proof.TrustSigned, proof.ProducedByCI, exampleSurvivor),
			"produced by CI": survivorMessage(t, proof.TrustLocal, proof.ProducedByCI, exampleSurvivor),
			"unknown origin": survivorMessage(t, proof.TrustLocal, proof.ProducedByLocal, unknown),
		} {
			if !strings.HasPrefix(message, bare) || strings.Contains(message, "every test passes on") {
				t.Errorf("%s: message %q", name, message)
			}
		}
	})
}

func TestExampleText(t *testing.T) {
	t.Run("writes an example back as a statement writes it [verifies SN-139~1]", func(t *testing.T) {
		raises := domain.Example{
			Operation: "quote", Args: []domain.Argument{
				{Name: "entry", Value: domain.Literal{Type: domain.LiteralDatetime, Value: "2026-03-02T10:00"}},
				{Name: "label", Value: domain.Literal{Type: domain.LiteralString, Value: "a\"b\\c\nd"}},
			},
			Expect: domain.ExpectedOutcome{Kind: domain.OutcomeRaises, Value: domain.Literal{Type: domain.LiteralName, Value: "ValueError"}},
		}
		for want, example := range map[string]domain.Example{
			"ticket_price(age=65) => 8": boxExample,
			`quote(entry=2026-03-02T10:00, label="a\"b\\c\nd") => raises ValueError`: raises,
		} {
			if got := example.String(); got != want {
				t.Errorf("got %s, want %s", got, want)
			}
			parsed := domain.ParseExamples("Example: " + want)
			if len(parsed.Examples) != 1 || parsed.Examples[0].String() != want {
				t.Errorf("%s does not read back: %+v", want, parsed)
			}
		}
	})
}
