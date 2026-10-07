package proof_test

import (
	"slices"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const (
	powerCriterion = "PARK-2~1"
	otherCriterion = "PARK-3~1"
)

func alignmentWith(inputs ...proof.Input) proof.Statement {
	return prooftest.Alignment(prooftest.RequirementID, prooftest.RequirementRevision, statementDigest, "test_one", inputs...)
}

func specInput() proof.Input {
	return proof.Input{Role: proof.RoleSpec, Name: powerCriterion, Digest: proof.Digest{SHA256: statementDigest}}
}

func TestInputDescriptors(t *testing.T) {
	proofOf := func(criterion string) proof.Input {
		return prooftest.ProofInput(proof.KindDiscriminatingPower, criterion, prooftest.Digest("consumed "+criterion))
	}

	t.Run("accepts a spec input that repeats the subject and a proof input that names its criterion [verifies SN-136~1]", func(t *testing.T) {
		wantCodes(t, alignmentWith(specInput(), proofOf(powerCriterion)))
	})

	t.Run("reports a spec input that differs from the subject [verifies SN-136~1]", func(t *testing.T) {
		wrongName := specInput()
		wrongName.Name = otherCriterion
		wantCodes(t, alignmentWith(wrongName), proof.CodeInputDescriptor)

		wrongDigest := specInput()
		wrongDigest.Digest.SHA256 = prooftest.Digest("another statement")
		wantCodes(t, alignmentWith(wrongDigest), proof.CodeInputDescriptor)
	})

	t.Run("reports a file input whose name is not a relative POSIX path [verifies SN-136~1]", func(t *testing.T) {
		for _, name := range []string{"/abs/a.py", "../a.py", "a//b.py", `a\b.py`} {
			wantCodes(t, alignmentWith(prooftest.Input(proof.RoleCodeTree, name)), proof.CodeInputDescriptor)
			wantCodes(t, alignmentWith(prooftest.Input(proof.RoleTestSource, name)), proof.CodeInputDescriptor)
		}
		wantCodes(t, alignmentWith(prooftest.Input(proof.RoleCodeTree, "src/a.py"), prooftest.Input(proof.RoleTestSource, "tests/t.py")))
	})

	t.Run("reports a proof input that is not named by a proof kind or names no criterion [verifies SN-136~1]", func(t *testing.T) {
		notAKind := proofOf(powerCriterion)
		notAKind.Name = "PARK-2~1"
		wantCodes(t, alignmentWith(notAKind), proof.CodeInputDescriptor)

		withoutAnnotation := proofOf(powerCriterion)
		withoutAnnotation.Annotations = nil
		wantCodes(t, alignmentWith(withoutAnnotation), proof.CodeInputDescriptor)

		withoutCriterion := proofOf("")
		wantCodes(t, alignmentWith(withoutCriterion), proof.CodeInputDescriptor)

		for _, criterion := range []string{"PARK-2", "PARK-2~0", "PARK-2~x", "~1", "PARK-2~01"} {
			wantCodes(t, alignmentWith(proofOf(criterion)), proof.CodeInputDescriptor)
		}
	})

	t.Run("accepts several proof inputs of one kind that name different criteria, ordered by criterion [verifies SN-136~1]", func(t *testing.T) {
		both := alignmentWith(proofOf(otherCriterion), proofOf(powerCriterion))
		inputs := both.Alignment.Inputs
		criteria := []string{}
		for _, input := range inputs {
			if input.Role == proof.RoleProof {
				criteria = append(criteria, input.Annotations.Criterion)
			}
		}
		if !slices.Equal(criteria, []string{powerCriterion, otherCriterion}) {
			t.Fatalf("criteria = %v", criteria)
		}
		wantCodes(t, both)
	})

	t.Run("reports as unsorted two proof inputs of one kind that name one criterion, and inputs out of criterion order [verifies SN-136~1]", func(t *testing.T) {
		wantCodes(t, alignmentWith(proofOf(powerCriterion), proofOf(powerCriterion)), proof.CodeUnsorted)

		outOfOrder := alignmentWith(proofOf(powerCriterion), proofOf(otherCriterion))
		inputs := outOfOrder.Alignment.Inputs
		slices.Reverse(inputs)
		outOfOrder.Alignment.CacheKey, _ = proof.CacheKey(outOfOrder.Subject, inputs)
		wantCodes(t, outOfOrder, proof.CodeUnsorted)
	})

	t.Run("leaves the criterion annotation out of the cache key and keeps the globs in it [verifies SN-136~1]", func(t *testing.T) {
		subject := alignment().Subject
		key := func(inputs ...proof.Input) string {
			result, err := proof.CacheKey(subject, inputs)
			if err != nil {
				t.Fatal(err)
			}
			return result
		}
		consumed := prooftest.Digest("consumed")
		withCriterion := prooftest.ProofInput(proof.KindDiscriminatingPower, powerCriterion, consumed)
		withAnotherCriterion := prooftest.ProofInput(proof.KindDiscriminatingPower, otherCriterion, consumed)
		if key(withCriterion) != key(withAnotherCriterion) {
			t.Fatal("the criterion annotation changed the cache key")
		}
		withGlobs := withCriterion
		withGlobs.Annotations = &proof.Annotations{Criterion: powerCriterion, Globs: []string{"src/**"}}
		if key(withCriterion) == key(withGlobs) {
			t.Fatal("the globs did not change the cache key")
		}
	})
}

func TestConfidenceSource(t *testing.T) {
	decided := func(source proof.ConfidenceSource, mapping string) proof.Statement {
		statement := alignment()
		statement.Alignment.Extensions.Decision = &proof.Decision{
			Basis: proof.BasisJudges, Deciders: []string{"claude-sonnet-5-5"}, ConfidenceSource: source, ConfidenceMapping: mapping,
			RawConfidence: 0.9, Threshold: 0.7,
		}
		return statement
	}

	t.Run("reads the confidence source and its mapping of a decision [verifies SN-137~1]", func(t *testing.T) {
		decoded := roundTrip(t, decided("host-votes(2)", "host-votes.modal-share"))
		decision := decoded.Alignment.Extensions.Decision
		if decision == nil || decision.ConfidenceSource != "host-votes(2)" || decision.ConfidenceMapping != "host-votes.modal-share" {
			t.Fatalf("decision = %+v", decision)
		}
	})

	t.Run("accepts the sources of the format [verifies SN-137~1]", func(t *testing.T) {
		for _, source := range []proof.ConfidenceSource{"execution", "host-votes(1)", "host-votes(12)", "system-one:model-a@0123456789abcdef"} {
			wantCodes(t, decided(source, ""))
		}
	})

	t.Run("reports an unknown value for a source outside the format, an empty source included [verifies SN-137~1]", func(t *testing.T) {
		for _, source := range []proof.ConfidenceSource{"", "vibes", "host-votes", "host-votes()", "host-votes(0)", "host-votes(x)", "host-votes(2", "system-one", "system-one:", "Execution"} {
			wantCodes(t, decided(source, ""), proof.CodeUnknownValue)
		}
	})

	t.Run("gives each source its kind [verifies SN-137~1]", func(t *testing.T) {
		cases := map[proof.ConfidenceSource]proof.ConfidenceSource{
			"execution": "execution", "host-votes(2)": "host-votes", "system-one:m@d": "system-one",
		}
		for source, kind := range cases {
			if got, known := source.Kind(); !known || got != kind {
				t.Errorf("%q: kind %q, %v", source, got, known)
			}
		}
		if _, known := proof.ConfidenceSource("vibes").Kind(); known {
			t.Error("an unknown source has a kind")
		}
	})
}

func TestThresholdForSource(t *testing.T) {
	threshold := proof.Threshold{Default: 0.7, BySource: map[proof.ConfidenceSource]float64{
		"host-votes": 0.8, "host-votes(2)": 0.85, "system-one": 0.9,
	}}
	cases := map[proof.ConfidenceSource]float64{
		"host-votes(2)": 0.85, "host-votes(3)": 0.8, "system-one:m@d": 0.9, "execution": 0.7, "": 0.7, "vibes": 0.7,
	}
	for source, want := range cases {
		t.Run("takes "+string(source)+" [verifies SN-138~1]", func(t *testing.T) {
			if got := threshold.For(source); got != want {
				t.Fatalf("For(%q) = %v, want %v", source, got, want)
			}
		})
	}
}
