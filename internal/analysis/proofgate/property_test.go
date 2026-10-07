package proofgate_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const maxMutants = 5

var allKinds = []proof.CorroborationKind{
	proof.CorroborationExampleFailed, proof.CorroborationOtherExampleFailed, proof.CorroborationSpecificKill,
	proof.CorroborationWitness, proof.CorroborationBuilds, proof.CorroborationOtherCriteriaPass,
	proof.CorroborationNoDiffFound, proof.CorroborationSpecificMatrix,
}

func drawConfig(t *rapid.T) proof.Config {
	return config(func(c *proof.Config) {
		c.Trust = rapid.SampledFrom([]proof.Trust{proof.TrustSigned, proof.TrustLocal}).Draw(t, "trust")
		c.SignedPassBasis = rapid.SampledFrom([]proof.SignedPassBasis{proof.BasisExecutedOnly, proof.BasisCorroborated, proof.BasisJudged}).Draw(t, "basis")
		c.MinIsolation = rapid.SampledFrom(proof.Isolations()).Draw(t, "floor")
		c.MaxHostExclusions = rapid.IntRange(0, 3).Draw(t, "maxHostExclusions")
		c.Thresholds.Validity = proof.Threshold{Default: rapid.Float64Range(0, 1).Draw(t, "validityThreshold")}
		c.Thresholds.Alignment = proof.Threshold{Default: rapid.Float64Range(0, 1).Draw(t, "alignmentThreshold")}
		c.Require = []proof.Kind{proof.KindDiscriminatingPower}
	})
}

// drawMutant draws a mutant of the pool; corroborated says whether it may list executed evidence.
func drawMutant(t *rapid.T, label string, corroborated bool, outcomes ...proof.Outcome) proof.Mutant {
	validity := rapid.SampledFrom([]proof.Validity{proof.ValidityValid, proof.ValidityEquivalent, proof.ValidityNotViolating, proof.ValidityUndetermined}).Draw(t, label+" validity")
	var kinds []proof.CorroborationKind
	if corroborated {
		kinds = rapid.SliceOfNDistinct(rapid.SampledFrom(allKinds), 0, len(allKinds), func(kind proof.CorroborationKind) proof.CorroborationKind { return kind }).Draw(t, label+" evidence")
	}
	claim := prooftest.Judged(validity, rapid.Float64Range(0, 1).Draw(t, label+" confidence"), kinds...)
	claim.Executor = ptr(prooftest.Executor(rapid.SampledFrom(proof.Isolations()).Draw(t, label+" isolation")))
	outcome := rapid.SampledFrom(outcomes).Draw(t, label+" outcome")
	return prooftest.Mutant(label+rapid.StringMatching(`[a-z]{6}`).Draw(t, label+" seed"), claim, map[string]proof.Outcome{testName: outcome})
}

func drawPool(t *rapid.T, corroborated bool, outcomes ...proof.Outcome) []proof.Mutant {
	count := rapid.IntRange(0, maxMutants).Draw(t, "mutants")
	pool := make([]proof.Mutant, count)
	for index := range pool {
		pool[index] = drawMutant(t, "mutant", corroborated, outcomes...)
	}
	return pool
}

func statementOf(mutants []proof.Mutant, mutate ...func(*proof.DiscriminatingPower)) proof.Statement {
	statement := dp()
	statement.DiscriminatingPower.Mutants = mutants
	for _, change := range mutate {
		change(statement.DiscriminatingPower)
	}
	return prooftest.Seal(statement)
}

func outcomeOfPool(cfg proof.Config, statements ...proof.Statement) proofgate.Outcome {
	result := evaluate(cfg, localize(cfg, statements...))
	return result.Outcomes[criterion().Ref]
}

// localize gives local trust the evidence it needs: a locally accepted set of locally produced proofs.
func localize(cfg proof.Config, statements ...proof.Statement) proof.Evidence {
	if cfg.Trust == proof.TrustLocal {
		for _, statement := range statements {
			if statement.DiscriminatingPower != nil {
				statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
			}
		}
	}
	ev := evidence(statements...)
	if cfg.Trust == proof.TrustLocal {
		ev.Manifest.Trust = proof.TrustLocal
	}
	return ev
}

func TestMonotonicity(t *testing.T) {
	t.Run("adding a response that blocks or escalates never makes an outcome weaker [verifies SN-129~1]", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := drawConfig(t)
			pool := drawPool(t, true, proof.OutcomeKilled, proof.OutcomeSurvived, proof.OutcomeError)
			if rapid.Bool().Draw(t, "starts from a passing pool") {
				pool = append([]proof.Mutant{killed("credit", valid(highConfidence, proof.CorroborationExampleFailed))}, pool...)
			}
			before := outcomeOfPool(cfg, statementOf(pool))

			kind := rapid.IntRange(0, 4).Draw(t, "addition")
			var after proof.Statement
			switch kind {
			case 0:
				added := drawMutant(t, "added", true, proof.OutcomeSurvived)
				after = statementOf(append(append([]proof.Mutant{}, pool...), added))
			case 1:
				after = statementOf(pool, func(p *proof.DiscriminatingPower) { p.CorrectCodePasses = prooftest.Executed(false) })
			case 2:
				after = statementOf(pool, func(p *proof.DiscriminatingPower) {
					p.Examples = []proof.ExampleRun{{Index: 0, Result: prooftest.Executed(proof.ExampleFailed)}}
				})
			case 3:
				sentinel := prooftest.Mutant("sentinel", prooftest.Judged(proof.ValidityEquivalent, 1), map[string]proof.Outcome{testName: proof.OutcomeKilled})
				sentinel.Sentinel = true
				after = statementOf(append(append([]proof.Mutant{}, pool...), sentinel))
			default:
				after = statementOf(pool)
			}
			result := outcomeOfPool(cfg, after)
			if kind == 4 {
				// An alignment proof that does not clear its test only adds an escalation.
				verdict := verdict(proof.VerdictMisaligned, 1)
				alignment := aligned(testName, verdict)
				ev := localize(cfg, after)
				if cfg.Trust == proof.TrustLocal {
					alignment.Alignment.ProducedBy = proof.ProducedByLocal
					alignment = prooftest.Seal(alignment)
				}
				ev.Statements = loaded(after, alignment)
				result = evaluate(cfg, ev).Outcomes[criterion().Ref]
			}
			if result < before {
				t.Fatalf("addition %d turned outcome %d into %d", kind, before, result)
			}
		})
	})
}

func TestNoJudgedOnlyBlock(t *testing.T) {
	t.Run("a proof whose judged claims list no executed evidence never blocks, and never passes under a signed basis other than judged [verifies SN-130~1]", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := drawConfig(t)
			pool := drawPool(t, false, proof.OutcomeKilled, proof.OutcomeSurvived, proof.OutcomeError)
			statement := statementOf(pool)
			got := outcomeOfPool(cfg, statement)
			if got == proofgate.OutcomeBlock {
				t.Fatalf("a judged signal alone blocked: %v", messages(evaluate(cfg, localize(cfg, statement))))
			}
			if cfg.Trust == proof.TrustSigned && cfg.SignedPassBasis != proof.BasisJudged && got == proofgate.OutcomePass {
				t.Fatalf("a judged signal alone passed under %s", cfg.SignedPassBasis)
			}
		})
	})
}
