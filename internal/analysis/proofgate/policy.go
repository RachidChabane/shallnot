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

func (p policy) confidentValidity(confidence float64) bool {
	return confidence >= p.config.Thresholds.Validity
}

func (p policy) confidentAlignment(confidence float64) bool {
	return confidence >= p.config.Thresholds.Alignment
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
