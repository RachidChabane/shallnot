package proofgate

import "github.com/RachidChabane/shallnot/internal/domain/proof"

// policy applies the thresholds, the isolation floor and the pass basis of the configuration.
type policy struct{ config proof.Config }

// basis is the evidence a pass may rest on. Local proofs are feedback for the agent, never proof, so they take the
// most permissive basis: the floor and the thresholds still apply.
func (p policy) basis() proof.SignedPassBasis {
	if p.config.Trust == proof.TrustLocal {
		return proof.BasisJudged
	}
	return p.config.SignedPassBasis
}

// validityThreshold is the confidence a judged validity must reach, for the confidence source that produced it.
func (p policy) validityThreshold(source proof.ConfidenceSource) float64 {
	return p.config.Thresholds.Validity.For(source)
}

// alignmentThreshold is the confidence an ALIGNED verdict must reach, for the confidence source that produced it.
func (p policy) alignmentThreshold(source proof.ConfidenceSource) float64 {
	return p.config.Thresholds.Alignment.For(source)
}

// answerer is who answered a judged claim: the executor and the confidence source of the decision.
type answerer struct {
	executor proof.Executor
	source   proof.ConfidenceSource
}

// decisionSource is the confidence source a decision records; a claim without a decision has none, which takes the
// default threshold.
func decisionSource(decision *proof.Decision) proof.ConfidenceSource {
	if decision == nil {
		return ""
	}
	return decision.ConfidenceSource
}

// validityAnswerer is who answered the validity of a mutant: its executor and the source of the decision recorded on the
// mutant, else of the one recorded on the predicate.
func validityAnswerer(mutant proof.Mutant, predicate *proof.DiscriminatingPower) answerer {
	decision := predicate.Extensions.Decision
	if mutant.Extensions != nil && mutant.Extensions.Decision != nil {
		decision = mutant.Extensions.Decision
	}
	return answerer{executor: executorOf(mutant.Validity, predicate.Executor), source: decisionSource(decision)}
}

func (p policy) aboveFloor(executor proof.Executor) bool {
	return executor.Isolation.AtLeast(p.config.MinIsolation)
}

// has reports whether a claim lists executed evidence of every one of the kinds.
func has[V comparable](claim proof.Claim[V], kinds ...proof.CorroborationKind) bool {
	for _, kind := range kinds {
		if !claim.Corroborates(kind) {
			return false
		}
	}
	return true
}

// executorOf is the executor that answered a claim: its own, else the predicate's.
func executorOf[V comparable](claim proof.Claim[V], predicate proof.Executor) proof.Executor {
	if claim.Executor != nil {
		return *claim.Executor
	}
	return predicate
}

// The executed evidence the claim-power table names for each judged effect.
var (
	blockWitness     = []proof.CorroborationKind{proof.CorroborationWitness, proof.CorroborationBuilds, proof.CorroborationOtherCriteriaPass}
	creditEvidence   = []proof.CorroborationKind{proof.CorroborationSpecificKill, proof.CorroborationBuilds, proof.CorroborationOtherCriteriaPass, proof.CorroborationWitness}
	alignmentMatrix  = []proof.CorroborationKind{proof.CorroborationSpecificMatrix}
	equivalentProof  = []proof.CorroborationKind{proof.CorroborationNoDiffFound}
	exclusionExample = []proof.CorroborationKind{proof.CorroborationOtherExampleFailed}
	ownerExample     = []proof.CorroborationKind{proof.CorroborationExampleFailed}
)
