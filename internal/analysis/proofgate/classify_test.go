package proofgate_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func TestStaleness(t *testing.T) {
	t.Run("accepts a proof whose subject and inputs match the tree [verifies SN-111~1]", func(t *testing.T) {
		result := evaluate(config(), evidence(dp()))
		wantCategories(t, result)
		if result.Report.Consumed[0].State != proof.StateValid {
			t.Fatalf("state %s", result.Report.Consumed[0].State)
		}
	})

	cases := map[string]func(*proof.Statement, *proof.Evidence){
		"the statement digest changed": func(s *proof.Statement, _ *proof.Evidence) {
			s.Subject[0].Digest.SHA256 = prooftest.Digest("another statement")
		},
		"the revision changed": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Requirement.Revision = 2
			s.Subject[0].Name = proof.SubjectName(requirementID, 2)
		},
		"a spec input changed": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Inputs[2].Digest.SHA256 = prooftest.Digest("old specs")
		},
		"a test source input changed": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Inputs[1].Digest.SHA256 = prooftest.Digest("old tests")
		},
		"a code tree input changed": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Inputs[0].Digest.SHA256 = prooftest.Digest("old code")
		},
		"the proof narrows its own scope": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Inputs[0].Annotations = &proof.Annotations{Globs: []string{"src/only"}}
		},
		"the code set cannot be derived": func(_ *proof.Statement, ev *proof.Evidence) { ev.Derived.CodeTree = nil },
		"the spec set cannot be derived": func(_ *proof.Statement, ev *proof.Evidence) { ev.Derived.Spec = nil },
		"the requirement is no longer declared": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Requirement.ID = "PARK-9"
			s.Subject[0].Name = proof.SubjectName("PARK-9", revision)
		},
	}
	for name, mutate := range cases {
		t.Run("classifies a proof stale when "+name+" [verifies SN-111~1]", func(t *testing.T) {
			statement := dp()
			ev := evidence()
			mutate(&statement, &ev)
			statement = prooftest.Seal(statement)
			ev.Statements = loaded(statement)
			result := evaluate(config(), ev)
			wantCategories(t, result, "stale_proof")
			if result.Report.Consumed[0].State != proof.StateStale || result.Report.Consumed[0].Reason == "" {
				t.Fatalf("consumed = %+v", result.Report.Consumed[0])
			}
		})
	}

	t.Run("accepts globs that equal the configured ones [verifies SN-111~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.Inputs[0].Annotations = &proof.Annotations{Globs: []string{"src"}}
		wantCategories(t, evaluate(config(), evidence(prooftest.Seal(statement))))
	})

	t.Run("classifies stale a proof of a spec file that changed, for a completeness proof [verifies SN-111~1]", func(t *testing.T) {
		stale := prooftest.Completeness(specPath, prooftest.Digest("old spec file"), currentInputs()...)
		fresh := prooftest.Completeness(specPath, specFileDigest, currentInputs()...)
		fresh.Completeness.Questions = []proof.Question{{Dimension: "time", Question: "what about the clock?"}}
		fresh = prooftest.Seal(fresh)
		result := evaluate(config(), evidence(stale))
		if result.Report.Consumed[0].State != proof.StateStale {
			t.Fatalf("state %s", result.Report.Consumed[0].State)
		}
		wantCategories(t, evaluate(config(), evidence(dp(), fresh)), "completeness_question")
	})

	t.Run("classifies a local proof stale under trust signed and a locally accepted set too [verifies SN-111~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
		result := evaluate(config(), evidence(prooftest.Seal(statement)))
		wantCategories(t, result, "stale_proof")

		acceptedLocally := evidence(dp())
		acceptedLocally.Manifest.Trust = proof.TrustLocal
		wantCategories(t, evaluate(config(), acceptedLocally), "stale_proof")
	})

	t.Run("admits a local proof and a locally accepted set under trust local [verifies SN-111~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
		ev := evidence(prooftest.Seal(statement))
		ev.Manifest.Trust = proof.TrustLocal
		wantCategories(t, evaluate(config(local), ev))
	})
}

func proofInput(statement proof.Statement, kind proof.Kind) proof.Input {
	return proof.Input{Role: proof.RoleProof, Name: string(kind), Digest: proof.Digest{SHA256: statementFileDigest(statement)}}
}

func TestConsumedProofs(t *testing.T) {
	t.Run("keeps a consumer valid while the proof it consumes is valid [verifies SN-112~1]", func(t *testing.T) {
		power := dp()
		consumer := aligned(testName, prooftest.Judged(proof.VerdictAligned, 0.9, proof.CorroborationSpecificMatrix), proofInput(power, proof.KindDiscriminatingPower))
		result := evaluate(config(requireBoth), evidence(power, consumer))
		wantCategories(t, result)
	})

	t.Run("makes the consumer stale when the proof it consumes is absent [verifies SN-112~1]", func(t *testing.T) {
		power := dp()
		consumer := aligned(testName, prooftest.Judged(proof.VerdictAligned, 0.9, proof.CorroborationSpecificMatrix), proofInput(power, proof.KindDiscriminatingPower))
		result := evaluate(config(requireBoth), evidence(consumer))
		if got := result.Report.Consumed[0]; got.State != proof.StateStale {
			t.Fatalf("consumed = %+v", got)
		}
		wantCategories(t, result, "stale_proof", "stale_proof")
	})

	t.Run("makes the consumer stale when the proof it consumes is stale, transitively [verifies SN-112~1]", func(t *testing.T) {
		stalePower := dp()
		stalePower.DiscriminatingPower.Inputs[0].Digest.SHA256 = prooftest.Digest("old code")
		stalePower = prooftest.Seal(stalePower)
		middle := aligned(testName, prooftest.Judged(proof.VerdictAligned, 0.9, proof.CorroborationSpecificMatrix), proofInput(stalePower, proof.KindDiscriminatingPower))
		top := prooftest.Completeness(specPath, specFileDigest, append(currentInputs(), proofInput(middle, proof.KindAlignment))...)
		result := evaluate(config(requireBoth), evidence(stalePower, middle, top))
		for _, consumed := range result.Report.Consumed {
			if consumed.State != proof.StateStale {
				t.Errorf("%s is %s", consumed.Path, consumed.State)
			}
		}
	})
}
