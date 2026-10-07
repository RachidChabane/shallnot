// Package proofgate is the gate policy over the verified proofs: it classifies
// each proof as valid, stale or malformed, applies the claim-power table of the
// proof format to the valid ones and composes the result per criterion with
// traceability. It performs no I/O and consults no model.
package proofgate

import (
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// Effect is the power of a finding over the verdict.
type Effect uint8

const (
	// EffectBlock is evidence that the criterion is not met, or executed evidence that is absent.
	EffectBlock Effect = iota
	// EffectEscalate is a judged signal that cannot block alone: it fails the verdict only where the policy escalates to error.
	EffectEscalate
	// EffectInform never fails the verdict by default.
	EffectInform
)

// Finding is an observation of the proof policy before its severity is settled.
type Finding struct {
	Category      domain.Category
	Effect        Effect
	Module        string
	RequirementID domain.RequirementID
	Location      domain.Location
	Message       string
}

// KnownRequirement is a requirement the specs declare, with the digest of its statement.
type KnownRequirement struct {
	Revision domain.Revision
	Digest   string
}

// TestName names a passing test bound to a criterion.
type TestName struct {
	Name      string
	ClassName string
}

// Criterion is a requirement in focus that traceability found covered: the only ones the proofs are asked about.
type Criterion struct {
	Ref          domain.Ref
	Digest       string
	Location     domain.Location
	PassingTests []TestName
}

// Input is everything the policy needs.
type Input struct {
	Config proof.Config
	// Known holds every requirement the specs declare.
	Known    map[domain.RequirementID]KnownRequirement
	Criteria []Criterion
	Evidence proof.Evidence
}

// Outcome is what the proofs concluded for a criterion.
type Outcome uint8

const (
	OutcomePass Outcome = iota
	OutcomeEscalate
	OutcomeBlock
)

// Result is the findings, the report extension and the outcome of each criterion.
type Result struct {
	Findings []Finding
	Report   proof.Report
	Outcomes map[domain.Ref]Outcome
}

// outcomeOf is the strongest effect among findings.
func outcomeOf(findings []Finding) Outcome {
	outcome := OutcomePass
	for _, finding := range findings {
		switch finding.Effect {
		case EffectBlock:
			return OutcomeBlock
		case EffectEscalate:
			outcome = OutcomeEscalate
		}
	}
	return outcome
}
