package analysis

import (
	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// ProofInput is the verified proof set a run consumes and the policy it is consumed under.
type ProofInput struct {
	Config   proof.Config
	Evidence proof.Evidence
}

// settleProofs applies the proof policy to the criteria that traceability found covered, reports its findings with
// the severity the configuration gives them, and returns the report extension.
func (a *analyzer) settleProofs(rows []domain.Row) *proof.Report {
	if a.input.Proofs == nil {
		return nil
	}
	known := map[domain.RequirementID]proofgate.KnownRequirement{}
	var criteria []proofgate.Criterion
	for _, row := range rows {
		requirement := row.Requirement
		known[requirement.ID] = proofgate.KnownRequirement{Revision: requirement.Revision, Digest: requirement.StatementDigest()}
		if row.InFocus && row.Coverage == domain.CoverageCovered {
			criteria = append(criteria, proofgate.Criterion{
				Ref: requirement.Ref(), Digest: requirement.StatementDigest(), Location: requirement.Location, PassingTests: passingTests(row),
			})
		}
	}
	result := proofgate.Evaluate(proofgate.Input{Config: a.input.Proofs.Config, Known: known, Criteria: criteria, Evidence: a.input.Proofs.Evidence})
	for _, finding := range result.Findings {
		a.reportProof(finding)
	}
	return &result.Report
}

func passingTests(row domain.Row) []proofgate.TestName {
	var names []proofgate.TestName
	for _, test := range row.Tests {
		if test.Outcome == domain.OutcomePassed {
			names = append(names, proofgate.TestName{Name: test.Name, ClassName: test.ClassName})
		}
	}
	return names
}

// reportProof settles the severity of a proof finding: an explicit setting for its category wins; otherwise a finding
// that escalates a judged signal takes the escalation mode of the configuration.
func (a *analyzer) reportProof(finding proofgate.Finding) {
	severity := finding.Category.DefaultSeverity()
	if override, overridden := a.input.Policy.Overrides(finding.Category); overridden {
		severity = override
	} else if finding.Effect == proofgate.EffectEscalate && a.input.Proofs.Config.Escalation == proof.EscalationWarning {
		severity = domain.SeverityWarning
	}
	if severity == domain.SeverityOff {
		return
	}
	a.findings = append(a.findings, domain.Finding{
		Category: finding.Category, Severity: severity, Blocking: a.input.Policy.Blocks(severity), Message: finding.Message,
		RequirementID: finding.RequirementID, Location: finding.Location, Module: finding.Module,
	})
}
