package analysis

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// ProofInput is the verified proof set a run consumes and the policy it is consumed under.
type ProofInput struct {
	Config proof.Config
	// Scope is where the configuration lets the file descriptors of a proof point.
	Scope proof.Scope
	// BaseDir is the directory of the configuration file, as the tags' paths are written; descriptors are relative to it.
	BaseDir  string
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
				Ref: requirement.Ref(), Digest: requirement.StatementDigest(), Location: requirement.Location, PassingTests: a.passingTests(row), TestFiles: a.testFiles(requirement.Ref()),
				Examples: row.Examples,
			})
		}
	}
	result := proofgate.Evaluate(proofgate.Input{Config: a.input.Proofs.Config, Known: known, Criteria: criteria, Scope: a.input.Proofs.Scope, Evidence: a.input.Proofs.Evidence})
	for _, finding := range result.Findings {
		a.reportProof(finding)
	}
	return &result.Report
}

// testFiles are the files holding a source tag that cites the criterion, relative to the configuration file's directory.
func (a *analyzer) testFiles(ref domain.Ref) []string {
	seen := map[string]bool{}
	var files []string
	for _, tag := range a.input.SourceTags {
		if !citesRef(tag, ref) {
			continue
		}
		file := relativeToBase(a.input.Proofs.BaseDir, tag.Location.File)
		if !seen[file] {
			seen[file] = true
			files = append(files, file)
		}
	}
	sort.Strings(files)
	return files
}

func relativeToBase(baseDir, file string) string {
	file = filepath.ToSlash(filepath.Clean(file))
	if baseDir == "" {
		return file
	}
	return strings.TrimPrefix(file, filepath.ToSlash(filepath.Clean(baseDir))+"/")
}

func (a *analyzer) passingTests(row domain.Row) []proofgate.TestName {
	var names []proofgate.TestName
	for _, test := range row.Tests {
		if test.Outcome == domain.OutcomePassed {
			name := proofgate.TestName{Name: test.Name, ClassName: test.ClassName}
			if test.Source != nil {
				name.File = relativeToBase(a.input.Proofs.BaseDir, test.Source.File)
			}
			names = append(names, name)
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
