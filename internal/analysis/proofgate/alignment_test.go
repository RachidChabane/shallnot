package proofgate_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func verdict(value proof.Verdict, confidence float64, kinds ...proof.CorroborationKind) proof.Claim[proof.Verdict] {
	return prooftest.Judged(value, confidence, kinds...)
}

func alignmentOf(claim proof.Claim[proof.Verdict], mutate ...func(*proof.Config)) proofgate.Result {
	return evaluate(config(append([]func(*proof.Config){requireBoth}, mutate...)...), evidence(dp(), aligned(testName, claim)))
}

func TestAlignment(t *testing.T) {
	matrix := proof.CorroborationSpecificMatrix

	t.Run("clears a test judged ALIGNED at the threshold with matrix evidence [verifies SN-119~2]", func(t *testing.T) {
		result := alignmentOf(verdict(proof.VerdictAligned, 0.7, matrix))
		wantCategories(t, result)
		if outcome(t, result) != proofgate.OutcomePass || result.Report.Criteria[0].Modules[proof.KindAlignment].Outcome != proof.OutcomePass {
			t.Fatalf("report %+v", result.Report.Criteria[0])
		}
	})

	t.Run("escalates MISALIGNED and UNDETERMINED and never blocks [verifies SN-119~2]", func(t *testing.T) {
		for _, value := range []proof.Verdict{proof.VerdictMisaligned, proof.VerdictUndetermined} {
			result := alignmentOf(verdict(value, 1, matrix))
			wantCategories(t, result, "misaligned_test")
			if result.Findings[0].Effect != proofgate.EffectEscalate || outcome(t, result) != proofgate.OutcomeEscalate {
				t.Errorf("%s: %+v", value, result.Findings[0])
			}
		}
	})

	t.Run("escalates an ALIGNED verdict below the alignment threshold [verifies SN-119~2]", func(t *testing.T) {
		wantCategories(t, alignmentOf(verdict(proof.VerdictAligned, 0.69, matrix)), "misaligned_test")
		lenient := func(c *proof.Config) { c.Thresholds.Alignment = proof.Threshold{Default: 0.5} }
		wantCategories(t, alignmentOf(verdict(proof.VerdictAligned, 0.6, matrix), lenient))
	})

	t.Run("escalates an ALIGNED verdict from an executor below the isolation floor [verifies SN-119~2]", func(t *testing.T) {
		claim := verdict(proof.VerdictAligned, 0.9, matrix)
		claim.Executor = ptr(prooftest.Executor(proof.IsolationInline))
		wantCategories(t, alignmentOf(claim), "misaligned_test")
		wantCategories(t, alignmentOf(claim, func(c *proof.Config) { c.MinIsolation = proof.IsolationInline }))
	})

	t.Run("treats an ALIGNED verdict without matrix evidence as uncorroborated: it escalates under signed unless the basis is judged [verifies SN-119~2]", func(t *testing.T) {
		claim := verdict(proof.VerdictAligned, 0.9)
		wantCategories(t, alignmentOf(claim), "misaligned_test")
		wantCategories(t, alignmentOf(claim, basis(proof.BasisExecutedOnly)), "misaligned_test")
		wantCategories(t, alignmentOf(claim, basis(proof.BasisJudged)))
	})

	t.Run("clears an uncorroborated ALIGNED verdict under trust local [verifies SN-119~2]", func(t *testing.T) {
		power := dp()
		power.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
		alignment := aligned(testName, verdict(proof.VerdictAligned, 0.9))
		alignment.Alignment.ProducedBy = proof.ProducedByLocal
		ev := evidence(prooftest.Seal(power), prooftest.Seal(alignment))
		ev.Manifest.Trust = proof.TrustLocal
		wantCategories(t, evaluate(config(requireBoth, local), ev))
	})

	t.Run("matches the proof's test to the bound test by name or by class name and name [verifies SN-119~2]", func(t *testing.T) {
		qualified := aligned("tests.ParkTest.test_one", verdict(proof.VerdictAligned, 0.9, matrix))
		input := proofgate.Input{
			Config: config(requireBoth), Known: known(), Scope: scope(), Evidence: evidence(dp(), qualified),
			Criteria: []proofgate.Criterion{{
				Ref: domain.Ref{ID: requirementID, Revision: revision}, Digest: statementDigest,
				PassingTests: []proofgate.TestName{{Name: "test_one", ClassName: "tests.ParkTest"}}, TestFiles: []string{testFile},
			}},
		}
		wantCategories(t, proofgate.Evaluate(input))
	})

	t.Run("asks an alignment proof of every passing test and names the ones without [verifies SN-119~2]", func(t *testing.T) {
		result := evaluate(config(requireBoth), evidence(dp(), aligned(testName, verdict(proof.VerdictAligned, 0.9, matrix))), testName, "test_two")
		wantCategories(t, result, "stale_proof")
		if got := result.Report.Criteria[0].Absent; len(got) != 1 || got[0] != proof.KindAlignment {
			t.Fatalf("absent %v", got)
		}
	})

	t.Run("does not require alignment unless configured, and still judges one that is present [verifies SN-119~2]", func(t *testing.T) {
		only := func(c *proof.Config) { c.Require = []proof.Kind{proof.KindDiscriminatingPower} }
		wantCategories(t, evaluate(config(only), evidence(dp())))
		wantCategories(t, evaluate(config(only), evidence(dp(), aligned(testName, verdict(proof.VerdictMisaligned, 0.9, matrix)))), "misaligned_test")
	})
}
