package proofgate_test

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func TestStaleness(t *testing.T) {
	t.Run("accepts a proof whose subject and inputs match the tree [verifies SN-111~2]", func(t *testing.T) {
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
		"the requirement is no longer declared": func(s *proof.Statement, _ *proof.Evidence) {
			s.DiscriminatingPower.Requirement.ID = "PARK-9"
			s.Subject[0].Name = proof.SubjectName("PARK-9", revision)
		},
	}
	for name, mutate := range cases {
		t.Run("classifies a proof stale when "+name+" [verifies SN-111~2]", func(t *testing.T) {
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

	t.Run("classifies a proof stale when a listed code or test file differs or is absent, by name [verifies SN-132~1]", func(t *testing.T) {
		for _, file := range []string{codeFile, testFile} {
			for name, change := range map[string]func(proof.Evidence){
				"differs": func(ev proof.Evidence) { ev.Derived.Files[file] = prooftest.Digest("edited") },
				"absent":  func(ev proof.Evidence) { delete(ev.Derived.Files, file) },
			} {
				ev := evidence(dp())
				change(ev)
				result := evaluate(config(), ev)
				wantCategories(t, result, "stale_proof")
				if reason := result.Report.Consumed[0].Reason; !strings.Contains(reason, file) {
					t.Errorf("%s %s: reason %q does not name the file", file, name, reason)
				}
			}
		}
	})

	t.Run("leaves a proof valid when a file outside every descriptor changes [verifies SN-132~1]", func(t *testing.T) {
		ev := evidence(dp())
		ev.Derived.Files["src/other.py"] = prooftest.Digest("edited")
		ev.Derived.Files["tests/other_test.py"] = prooftest.Digest("edited")
		wantCategories(t, evaluate(config(), ev))
	})

	t.Run("classifies a proof stale when a descriptor lies outside the configured globs or roots [verifies SN-133~1]", func(t *testing.T) {
		outside := map[string]proof.Input{
			"a code file outside proofs.code":     prooftest.Input(proof.RoleCodeTree, "lib/parkfee.py"),
			"a test file outside the tests roots": prooftest.Input(proof.RoleTestSource, "spec/other_test.py"),
			"an excluded code file":               prooftest.Input(proof.RoleCodeTree, "src/node_modules/dep/x.js"),
			"a path that escapes the tree":        prooftest.Input(proof.RoleCodeTree, "src/../secret.py"),
		}
		for name, input := range outside {
			statement := dp()
			statement.DiscriminatingPower.Inputs = append(statement.DiscriminatingPower.Inputs, input)
			ev := evidence(prooftest.Seal(statement))
			ev.Derived.Files[input.Name] = input.Digest.SHA256
			result := evaluate(config(), ev)
			wantCategories(t, result, "stale_proof")
			if reason := result.Report.Consumed[0].Reason; !strings.Contains(reason, input.Name) {
				t.Errorf("%s: reason %q", name, reason)
			}
		}
	})

	t.Run("accepts a code descriptor under a file entry or a glob of proofs.code [verifies SN-133~1]", func(t *testing.T) {
		for _, entries := range [][]string{{codeFile}, {"src/**/*.py"}, {"lib", "src"}} {
			ev := evidence(dp())
			input := scope()
			input.Code = entries
			result := proofgate.Evaluate(proofgate.Input{Config: config(), Known: known(), Criteria: []proofgate.Criterion{criterion()}, Scope: input, Evidence: ev})
			wantCategories(t, result)
		}
	})

	t.Run("classifies a proof stale when a test file bound to the criterion is not a test_source input [verifies SN-134~2]", func(t *testing.T) {
		moved := criterion()
		moved.TestFiles = []string{testFile, "tests/new_test.py"}
		result := proofgate.Evaluate(proofgate.Input{Config: config(), Known: known(), Criteria: []proofgate.Criterion{moved}, Scope: scope(), Evidence: evidence(dp())})
		wantCategories(t, result, "stale_proof")
		if reason := result.Report.Consumed[0].Reason; !strings.Contains(reason, "tests/new_test.py") {
			t.Fatalf("reason %q", reason)
		}
	})

	specCases := map[string]func(*proof.Input){
		"names another criterion":          func(i *proof.Input) { i.Name = proof.SubjectName("PARK-3", revision) },
		"names another revision":           func(i *proof.Input) { i.Name = proof.SubjectName(requirementID, 2) },
		"names a file":                     func(i *proof.Input) { i.Name = specPath },
		"carries another statement digest": func(i *proof.Input) { i.Digest.SHA256 = prooftest.Digest("another statement") },
	}
	for name, change := range specCases {
		t.Run("classifies a proof stale when its spec descriptor "+name+" [verifies SN-135~1]", func(t *testing.T) {
			statement := dp()
			for index := range statement.DiscriminatingPower.Inputs {
				if statement.DiscriminatingPower.Inputs[index].Role == proof.RoleSpec {
					change(&statement.DiscriminatingPower.Inputs[index])
				}
			}
			wantCategories(t, evaluate(config(), evidence(prooftest.Seal(statement))), "stale_proof")
		})
	}

	t.Run("classifies stale a proof of a spec file that changed, for a completeness proof [verifies SN-111~2]", func(t *testing.T) {
		stale := prooftest.Completeness(specPath, prooftest.Digest("old spec file"), completenessInputs()...)
		fresh := prooftest.Completeness(specPath, specFileDigest, completenessInputs()...)
		fresh.Completeness.Questions = []proof.Question{{Dimension: "time", Question: "what about the clock?"}}
		fresh = prooftest.Seal(fresh)
		result := evaluate(config(), evidence(stale))
		if result.Report.Consumed[0].State != proof.StateStale {
			t.Fatalf("state %s", result.Report.Consumed[0].State)
		}
		wantCategories(t, evaluate(config(), evidence(dp(), fresh)), "completeness_question")
	})

	t.Run("classifies a local proof stale under trust signed and a locally accepted set too [verifies SN-111~2]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
		result := evaluate(config(), evidence(prooftest.Seal(statement)))
		wantCategories(t, result, "stale_proof")

		acceptedLocally := evidence(dp())
		acceptedLocally.Manifest.Trust = proof.TrustLocal
		wantCategories(t, evaluate(config(), acceptedLocally), "stale_proof")
	})

	t.Run("admits a local proof and a locally accepted set under trust local [verifies SN-111~2]", func(t *testing.T) {
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
		top := prooftest.Completeness(specPath, specFileDigest, append(completenessInputs(), proofInput(middle, proof.KindAlignment))...)
		result := evaluate(config(requireBoth), evidence(stalePower, middle, top))
		for _, consumed := range result.Report.Consumed {
			if consumed.State != proof.StateStale {
				t.Errorf("%s is %s", consumed.Path, consumed.State)
			}
		}
	})
}
