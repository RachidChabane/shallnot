package proof_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

const (
	counterexamplePath = "predicate.mutants[0].extensions.counterexample"
	witnessCodeCap     = 49_152 // The format's byte limit, independently of the implementation constant.
)

type counterexampleViolation struct {
	code proof.ViolationCode
	path string
}

func wantCounterexampleViolations(t *testing.T, statement proof.Statement, want ...counterexampleViolation) {
	t.Helper()
	got := roundTrip(t, statement).Validate()
	if len(got) != len(want) {
		t.Fatalf("violations = %v, want %v", got, want)
	}
	for index, violation := range got {
		if violation.Code != want[index].code || violation.Path != want[index].path {
			t.Errorf("violation %d = %v, want %v", index, violation, want[index])
		}
	}
}

func withCounterexample(counterexample *proof.Counterexample) proof.Statement {
	statement := discriminatingPower()
	statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
	statement.DiscriminatingPower.Mutants[0].Extensions = &proof.MutantExtensions{Counterexample: counterexample}
	return statement
}

func TestCounterexampleRules(t *testing.T) {
	example := 0
	witness := proof.WitnessExample{Name: "boundary", Clause: "reaches 50", Code: "assert shipping_fee(50) == 0"}
	targeted := func() *proof.Counterexample {
		return &proof.Counterexample{Origin: proof.OriginTargeted, Slot: proof.SlotDirect, Example: &example}
	}
	operator := func() *proof.Counterexample {
		copy := witness
		return &proof.Counterexample{Origin: proof.OriginOperator, Family: "relational", Witness: &copy}
	}
	violation := func(code proof.ViolationCode, suffix string) []counterexampleViolation {
		return []counterexampleViolation{{code: code, path: counterexamplePath + suffix}}
	}

	t.Run("allows absent counterexamples for local and CI statements [verifies SN-140~1]", func(t *testing.T) {
		for _, producer := range []proof.ProducedBy{proof.ProducedByLocal, proof.ProducedByCI} {
			statement := withCounterexample(nil)
			statement.DiscriminatingPower.ProducedBy = producer
			wantCounterexampleViolations(t, statement)
			statement.DiscriminatingPower.Mutants[0].Extensions = nil
			wantCounterexampleViolations(t, statement)
		}
	})

	t.Run("requires local production whenever a counterexample is present [verifies SN-140~1]", func(t *testing.T) {
		for _, counterexample := range []*proof.Counterexample{targeted(), operator()} {
			statement := withCounterexample(counterexample)
			wantCounterexampleViolations(t, statement)
			statement.DiscriminatingPower.ProducedBy = proof.ProducedByCI
			wantCounterexampleViolations(t, statement, violation(proof.CodeCounterexampleNotLocal, "")...)
			statement.DiscriminatingPower.ProducedBy = "future-producer"
			wantCounterexampleViolations(t, statement,
				counterexampleViolation{proof.CodeUnknownValue, "predicate.producedBy"},
				counterexampleViolation{proof.CodeCounterexampleNotLocal, counterexamplePath})
		}
	})

	cases := []struct {
		name   string
		base   func() *proof.Counterexample
		mutate func(*proof.Counterexample)
		want   []counterexampleViolation
	}{
		{"targeted direct slot", targeted, func(c *proof.Counterexample) { c.Slot = proof.SlotDirect }, nil},
		{"targeted boundary slot", targeted, func(c *proof.Counterexample) { c.Slot = proof.SlotBoundary }, nil},
		{"targeted partial slot", targeted, func(c *proof.Counterexample) { c.Slot = proof.SlotPartial }, nil},
		{"unknown origin", targeted, func(c *proof.Counterexample) { c.Origin = "future" }, violation(proof.CodeUnknownValue, ".origin")},
		{"empty origin", targeted, func(c *proof.Counterexample) { c.Origin = "" }, violation(proof.CodeUnknownValue, ".origin")},
		{"targeted missing slot", targeted, func(c *proof.Counterexample) { c.Slot = "" }, violation(proof.CodeUnknownValue, ".slot")},
		{"targeted unknown slot", targeted, func(c *proof.Counterexample) { c.Slot = "S4" }, violation(proof.CodeUnknownValue, ".slot")},
		{"targeted with family", targeted, func(c *proof.Counterexample) { c.Family = "relational" }, violation(proof.CodeUnknownValue, ".slot")},
		{"operator witness", operator, func(c *proof.Counterexample) {}, nil},
		{"operator example", operator, func(c *proof.Counterexample) { c.Witness, c.Example = nil, &example }, nil},
		{"targeted witness", targeted, func(c *proof.Counterexample) { c.Example, c.Witness = nil, &witness }, nil},
		{"operator missing family", operator, func(c *proof.Counterexample) { c.Family = "" }, violation(proof.CodeMissingField, ".family")},
		{"operator with slot", operator, func(c *proof.Counterexample) { c.Slot = proof.SlotDirect }, violation(proof.CodeMissingField, ".family")},
		{"neither example nor witness", targeted, func(c *proof.Counterexample) { c.Example = nil }, violation(proof.CodeMissingField, "")},
		{"both example and witness", targeted, func(c *proof.Counterexample) { c.Witness = &witness }, violation(proof.CodeMissingField, "")},
		{"both payloads do not check witness fields", targeted, func(c *proof.Counterexample) {
			c.Witness = &proof.WitnessExample{Code: strings.Repeat("x", witnessCodeCap+1)}
		}, violation(proof.CodeMissingField, "")},
		{"witness missing name", operator, func(c *proof.Counterexample) { c.Witness.Name = "" }, violation(proof.CodeMissingField, ".witness.name")},
		{"witness missing code", operator, func(c *proof.Counterexample) { c.Witness.Code = "" }, violation(proof.CodeMissingField, ".witness.code")},
		{"witness empty clause", operator, func(c *proof.Counterexample) { c.Witness.Clause = "" }, nil},
		{"witness whitespace name and code", operator, func(c *proof.Counterexample) { c.Witness.Name, c.Witness.Code = " ", " " }, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name+" [verifies SN-140~1]", func(t *testing.T) {
			counterexample := tc.base()
			tc.mutate(counterexample)
			wantCounterexampleViolations(t, withCounterexample(counterexample), tc.want...)
		})
	}

	t.Run("collects independent counterexample violations [verifies SN-140~1]", func(t *testing.T) {
		counterexample := operator()
		counterexample.Family = ""
		counterexample.Witness.Name, counterexample.Witness.Code = "", ""
		statement := withCounterexample(counterexample)
		statement.DiscriminatingPower.ProducedBy = proof.ProducedByCI
		wantCounterexampleViolations(t, statement,
			counterexampleViolation{proof.CodeCounterexampleNotLocal, counterexamplePath},
			counterexampleViolation{proof.CodeMissingField, counterexamplePath + ".family"},
			counterexampleViolation{proof.CodeMissingField, counterexamplePath + ".witness.name"},
			counterexampleViolation{proof.CodeMissingField, counterexamplePath + ".witness.code"})
	})
}

func TestCounterexampleWitnessSize(t *testing.T) {
	for _, size := range []int{1, witnessCodeCap - 1, witnessCodeCap, witnessCodeCap + 1} {
		for _, multibyte := range []bool{false, true} {
			for _, truncated := range []bool{false, true} {
				name := fmt.Sprintf("%d bytes multibyte=%t truncated=%t [verifies SN-140~1]", size, multibyte, truncated)
				t.Run(name, func(t *testing.T) {
					code := strings.Repeat("x", size)
					if multibyte {
						code = strings.Repeat("é", size/len("é")) + strings.Repeat("x", size%len("é"))
					}
					statement := withCounterexample(&proof.Counterexample{
						Origin: proof.OriginOperator, Family: "relational",
						Witness: &proof.WitnessExample{Name: "boundary", Code: code, Truncated: truncated},
					})
					var want []counterexampleViolation
					if size > witnessCodeCap {
						want = []counterexampleViolation{{proof.CodeCounterexampleSize, counterexamplePath + ".witness.code"}}
					}
					wantCounterexampleViolations(t, statement, want...)
				})
			}
		}
	}
}
