package proofgate_test

import (
	"fmt"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const (
	highConfidence = 0.9
	lowConfidence  = 0.5
)

var (
	witness     = []proof.CorroborationKind{proof.CorroborationWitness, proof.CorroborationBuilds, proof.CorroborationOtherCriteriaPass}
	creditKinds = []proof.CorroborationKind{proof.CorroborationSpecificKill, proof.CorroborationBuilds, proof.CorroborationOtherCriteriaPass, proof.CorroborationWitness}
)

// withCredit is a pool in which a killed VALID mutant earns credit under every basis, so that a row's own effect shows alone.
func withCredit(mutants ...proof.Mutant) proof.Statement {
	credit := killed("credit", valid(highConfidence, proof.CorroborationExampleFailed))
	return dp(append([]proof.Mutant{credit}, mutants...)...)
}

func expect(t *testing.T, cfg proof.Config, statement proof.Statement, want ...string) proofgate.Result {
	t.Helper()
	result := evaluate(cfg, evidence(statement))
	wantCategories(t, result, want...)
	return result
}

func TestSurvivingMutants(t *testing.T) {
	bases := []proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated, proof.BasisJudged}

	for _, signedBasis := range bases {
		t.Run(fmt.Sprintf("blocks a survivor that fails an example of the criterion under %s [verifies SN-115~1]", signedBasis), func(t *testing.T) {
			survivor := survivor("s", valid(lowConfidence, proof.CorroborationExampleFailed))
			result := expect(t, config(basis(signedBasis)), withCredit(survivor), "surviving_valid_mutant")
			if outcome(t, result) != proofgate.OutcomeBlock {
				t.Fatal("not blocked")
			}
		})

		t.Run(fmt.Sprintf("blocks a survivor judged VALID at the threshold with its full witness under %s [verifies SN-115~1]", signedBasis), func(t *testing.T) {
			expect(t, config(basis(signedBasis)), withCredit(survivor("s", valid(0.7, witness...))), "surviving_valid_mutant")
		})
	}

	t.Run("escalates a survivor judged VALID below the threshold [verifies SN-115~1]", func(t *testing.T) {
		result := expect(t, config(), withCredit(survivor("s", valid(0.69, witness...))), "escalated_mutant")
		if outcome(t, result) != proofgate.OutcomeEscalate {
			t.Fatal("not escalated")
		}
	})

	t.Run("escalates a survivor judged VALID that lacks a part of the witness [verifies SN-115~1]", func(t *testing.T) {
		for _, missing := range witness {
			var kinds []proof.CorroborationKind
			for _, kind := range witness {
				if kind != missing {
					kinds = append(kinds, kind)
				}
			}
			kinds = append(kinds, proof.CorroborationSpecificKill)
			expect(t, config(), withCredit(survivor("s", valid(highConfidence, kinds...))), "escalated_mutant")
		}
	})

	t.Run("escalates a survivor whose validity is UNDETERMINED [verifies SN-115~1]", func(t *testing.T) {
		expect(t, config(), withCredit(survivor("s", judged(proof.ValidityUndetermined, highConfidence))), "escalated_mutant")
	})

	t.Run("blocks a survivor on the strength of a judgment executor below the floor, which keeps its block power [verifies SN-115~1]", func(t *testing.T) {
		mutant := survivor("s", valid(highConfidence, witness...))
		executor := prooftest.Executor(proof.IsolationInline)
		mutant.Validity.Executor = &executor
		expect(t, config(), withCredit(mutant), "surviving_valid_mutant")
	})

	t.Run("does not count a mutant that errored as a survivor or a kill [verifies SN-115~1]", func(t *testing.T) {
		errored := prooftest.Mutant("e", valid(highConfidence, witness...), map[string]proof.Outcome{testName: proof.OutcomeError})
		expect(t, config(), withCredit(errored))
	})
}

func TestExclusions(t *testing.T) {
	t.Run("excludes an EQUIVALENT survivor only on a differential search that found no difference [verifies SN-116~1]", func(t *testing.T) {
		for _, signedBasis := range []proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated, proof.BasisJudged} {
			expect(t, config(basis(signedBasis)), withCredit(survivor("s", judged(proof.ValidityEquivalent, highConfidence, proof.CorroborationNoDiffFound))))
			expect(t, config(basis(signedBasis)), withCredit(survivor("s", judged(proof.ValidityEquivalent, 1))), "escalated_mutant")
		}
	})

	t.Run("excludes a NOT_VIOLATING survivor on an example of another criterion under every basis [verifies SN-116~1]", func(t *testing.T) {
		for _, signedBasis := range []proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated, proof.BasisJudged} {
			result := expect(t, config(basis(signedBasis)), withCredit(survivor("s", judged(proof.ValidityNotViolating, lowConfidence, proof.CorroborationOtherExampleFailed))))
			if result.Report.Criteria[0].HostExclusions != 0 {
				t.Errorf("%s: an executed exclusion counted as a host exclusion", signedBasis)
			}
		}
	})

	t.Run("escalates a NOT_VIOLATING survivor on a judgment alone unless the basis is judged [verifies SN-116~1]", func(t *testing.T) {
		pool := withCredit(survivor("s", judged(proof.ValidityNotViolating, highConfidence)))
		expect(t, config(basis(proof.BasisExecutedOnly)), pool, "escalated_mutant")
		expect(t, config(basis(proof.BasisCorroborated)), pool, "escalated_mutant")
		result := expect(t, config(basis(proof.BasisJudged)), pool)
		if result.Report.Criteria[0].HostExclusions != 1 {
			t.Fatalf("host exclusions %d", result.Report.Criteria[0].HostExclusions)
		}
	})

	t.Run("escalates a judged exclusion below the threshold or below the isolation floor [verifies SN-116~1]", func(t *testing.T) {
		expect(t, config(basis(proof.BasisJudged)), withCredit(survivor("s", judged(proof.ValidityNotViolating, lowConfidence))), "escalated_mutant")
		mutant := survivor("s", judged(proof.ValidityNotViolating, highConfidence))
		executor := prooftest.Executor(proof.IsolationUnrestricted)
		mutant.Validity.Executor = &executor
		expect(t, config(basis(proof.BasisJudged)), withCredit(mutant), "escalated_mutant")
	})

	t.Run("stops excluding on a judgment alone once max_host_exclusions survivors are excluded [verifies SN-116~1]", func(t *testing.T) {
		pool := withCredit(
			survivor("a", judged(proof.ValidityNotViolating, highConfidence)),
			survivor("b", judged(proof.ValidityNotViolating, highConfidence)),
			survivor("c", judged(proof.ValidityNotViolating, highConfidence)),
		)
		cap := func(limit int) func(*proof.Config) {
			return func(c *proof.Config) { c.SignedPassBasis, c.MaxHostExclusions = proof.BasisJudged, limit }
		}
		expect(t, config(cap(2)), pool, "escalated_mutant")
		expect(t, config(cap(3)), pool)
		expect(t, config(cap(0)), pool, "escalated_mutant", "escalated_mutant", "escalated_mutant")
	})

	t.Run("treats an off-target and an unattributed survivor as NOT_VIOLATING [verifies SN-116~1]", func(t *testing.T) {
		offTarget := survivor("o", judged(proof.ValidityNotViolating, highConfidence, proof.CorroborationOtherExampleFailed))
		offTarget.Extensions = &proof.MutantExtensions{OffTarget: "PARK-3"}
		unattributed := survivor("u", judged(proof.ValidityNotViolating, highConfidence))
		unattributed.Extensions = &proof.MutantExtensions{Unattributed: true}
		expect(t, config(), withCredit(offTarget))
		expect(t, config(), withCredit(unattributed), "escalated_mutant")
	})

	t.Run("reports the screened operator mutants of the criterion [verifies SN-116~1]", func(t *testing.T) {
		statement := withCredit()
		statement.DiscriminatingPower.Extensions.OperatorScreen = &proof.OperatorScreen{Flagged: 2, ScreenedOut: 37}
		result := expect(t, config(), prooftest.Seal(statement))
		if result.Report.Criteria[0].ScreenExclusions != 37 {
			t.Fatalf("screen exclusions %d", result.Report.Criteria[0].ScreenExclusions)
		}
	})
}

func TestPassCredit(t *testing.T) {
	credited := func(confidence float64, kinds ...proof.CorroborationKind) proof.Statement {
		return dp(killed("k", valid(confidence, kinds...)))
	}

	t.Run("credits a kill of a mutant that fails an example of the criterion under every basis and floor [verifies SN-117~1]", func(t *testing.T) {
		for _, signedBasis := range []proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated, proof.BasisJudged} {
			statement := credited(lowConfidence, proof.CorroborationExampleFailed)
			statement.DiscriminatingPower.Mutants[0].Validity.Executor = ptr(prooftest.Executor(proof.IsolationInline))
			expect(t, config(basis(signedBasis)), statement)
		}
	})

	t.Run("credits a corroborated kill only with its full evidence under corroborated [verifies SN-117~1]", func(t *testing.T) {
		expect(t, config(basis(proof.BasisCorroborated)), credited(highConfidence, creditKinds...))
		for _, missing := range creditKinds {
			var kinds []proof.CorroborationKind
			for _, kind := range creditKinds {
				if kind != missing {
					kinds = append(kinds, kind)
				}
			}
			expect(t, config(basis(proof.BasisCorroborated)), credited(highConfidence, kinds...), "no_killed_valid_mutant")
		}
	})

	t.Run("credits a judged kill with any corroboration under judged and none under executed-only [verifies SN-117~1]", func(t *testing.T) {
		expect(t, config(basis(proof.BasisJudged)), credited(highConfidence, proof.CorroborationBuilds))
		expect(t, config(basis(proof.BasisExecutedOnly)), credited(highConfidence, creditKinds...), "no_killed_valid_mutant")
	})

	t.Run("credits no kill judged below the threshold or by an executor below the floor [verifies SN-117~1]", func(t *testing.T) {
		expect(t, config(basis(proof.BasisJudged)), credited(0.69, creditKinds...), "no_killed_valid_mutant")
		statement := credited(highConfidence, creditKinds...)
		statement.DiscriminatingPower.Mutants[0].Validity.Executor = ptr(prooftest.Executor(proof.IsolationRestrictedUnverified))
		expect(t, config(basis(proof.BasisJudged)), statement, "no_killed_valid_mutant")
		lowered := func(c *proof.Config) { c.MinIsolation = proof.IsolationRestrictedUnverified }
		expect(t, config(basis(proof.BasisJudged), lowered), statement)
	})

	t.Run("escalates when no mutant was judged VALID [verifies SN-117~1]", func(t *testing.T) {
		statement := dp()
		statement.DiscriminatingPower.Mutants = nil
		expect(t, config(), prooftest.Seal(statement), "no_valid_mutant")
		onlyExcluded := dp(killed("e", judged(proof.ValidityEquivalent, highConfidence)))
		expect(t, config(), onlyExcluded, "no_valid_mutant")
	})

	t.Run("escalates with no_killed_valid_mutant when VALID mutants exist and none earned credit [verifies SN-117~1]", func(t *testing.T) {
		result := expect(t, config(), dp(killed("k", valid(0.3, creditKinds...))), "no_killed_valid_mutant")
		if outcome(t, result) != proofgate.OutcomeEscalate {
			t.Fatal("not escalated")
		}
	})

	t.Run("reports the channels and the isolation levels of the judged claims [verifies SN-117~1]", func(t *testing.T) {
		result := expect(t, config(), dp(killed("k", valid(highConfidence, creditKinds...))))
		report := result.Report.Criteria[0]
		if report.Channels[proof.ChannelHostSubagent] != 1 || report.Isolation[proof.IsolationRestricted] != 1 {
			t.Fatalf("report %+v", report)
		}
	})
}

func ptr[T any](value T) *T { return &value }

func TestSentinels(t *testing.T) {
	sentinel := func(outcomes map[string]proof.Outcome) proof.Mutant {
		mutant := prooftest.Mutant("sentinel", judged(proof.ValidityEquivalent, 1), outcomes)
		mutant.Sentinel = true
		return mutant
	}

	t.Run("escalates a test that killed a sentinel and gives its kills no credit [verifies SN-118~1]", func(t *testing.T) {
		pool := dp(killed("k", valid(highConfidence, proof.CorroborationExampleFailed)), sentinel(map[string]proof.Outcome{testName: proof.OutcomeKilled}))
		result := expect(t, config(), pool, "mutation_sensitive_test", "no_killed_valid_mutant")
		if outcome(t, result) != proofgate.OutcomeEscalate {
			t.Fatal("not escalated")
		}
	})

	t.Run("leaves a test that killed no sentinel with its credit [verifies SN-118~1]", func(t *testing.T) {
		pool := dp(killed("k", valid(highConfidence, proof.CorroborationExampleFailed)), sentinel(map[string]proof.Outcome{testName: proof.OutcomeSurvived}))
		expect(t, config(), pool)
	})

	t.Run("caps the alignment of a test that killed a sentinel at UNDETERMINED [verifies SN-118~1]", func(t *testing.T) {
		power := dp(killed("k", valid(highConfidence, proof.CorroborationExampleFailed)), sentinel(map[string]proof.Outcome{testName: proof.OutcomeKilled}))
		alignment := aligned(testName, prooftest.Judged(proof.VerdictAligned, highConfidence, proof.CorroborationSpecificMatrix))
		result := evaluate(config(requireBoth), evidence(power, alignment))
		wantCategories(t, result, "mutation_sensitive_test", "misaligned_test", "no_killed_valid_mutant")
	})
}

func TestLocalTrust(t *testing.T) {
	t.Run("applies the judged basis under trust local, with the thresholds and the floor [verifies SN-120~1]", func(t *testing.T) {
		pool := withCredit(survivor("s", judged(proof.ValidityNotViolating, highConfidence)))
		for _, requested := range []proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated} {
			ev := evidence(pool)
			ev.Manifest.Trust = proof.TrustLocal
			statement := pool
			statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
			ev.Statements = loaded(prooftest.Seal(statement))
			wantCategories(t, evaluate(config(local, basis(requested)), ev))

			belowThreshold := killed("k", valid(0.3, proof.CorroborationBuilds))
			low := dp(belowThreshold)
			low.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
			ev.Statements = loaded(prooftest.Seal(low))
			wantCategories(t, evaluate(config(local, basis(requested)), ev), "no_killed_valid_mutant")
		}
	})
}
