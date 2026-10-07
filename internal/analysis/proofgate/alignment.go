package proofgate

import (
	"fmt"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// matches reports whether a proof's test names the bound test, by its name or by its class name and name.
func matches(proofTest string, test TestName) bool {
	return proofTest == test.Name || (test.ClassName != "" && proofTest == test.ClassName+"."+test.Name)
}

// alignmentClearance is whether an alignment proof clears one test, and why not when it does not.
type alignmentClearance struct {
	cleared bool
	reason  string
}

// clearance applies the ALIGNED and MISALIGNED rows of the claim-power table. Alignment never blocks: a verdict that
// does not clear a test escalates.
func (p policy) clearance(predicate *proof.Alignment, test TestName, tainted map[string]bool) alignmentClearance {
	verdict := predicate.Verdict
	executor := executorOf(verdict, predicate.Executor)
	threshold := p.alignmentThreshold(decisionSource(predicate.Extensions.Decision))
	switch {
	case tainted[test.Name] || (test.ClassName != "" && tainted[test.ClassName+"."+test.Name]):
		return alignmentClearance{reason: "the test killed a sentinel mutant, so its alignment is capped at UNDETERMINED"}
	case verdict.Value == proof.VerdictMisaligned:
		return alignmentClearance{reason: "judged MISALIGNED: its assertions do not check the statement"}
	case verdict.Value == proof.VerdictUndetermined:
		return alignmentClearance{reason: "judged UNDETERMINED"}
	case verdict.ConfidenceOrZero() < threshold:
		return alignmentClearance{reason: fmt.Sprintf("judged ALIGNED with confidence %.2f, below %.2f", verdict.ConfidenceOrZero(), threshold)}
	case !p.aboveFloor(executor):
		return alignmentClearance{reason: fmt.Sprintf("judged ALIGNED by an executor of isolation %s, below the floor %s", executor.Isolation, p.config.MinIsolation)}
	case p.basis() != proof.BasisJudged && !has(verdict, alignmentMatrix...):
		return alignmentClearance{reason: fmt.Sprintf("judged ALIGNED with no specific matrix evidence: uncorroborated, and signed_pass_basis %s does not let a judgment alone clear alignment", p.basis())}
	}
	return alignmentClearance{cleared: true}
}

func (p policy) alignmentFinding(criterion Criterion, test TestName, reason string) Finding {
	return Finding{
		Category: domain.CategoryMisalignedTest, Effect: EffectEscalate, Module: proof.KindAlignment.ModuleName(),
		RequirementID: criterion.Ref.ID, Location: criterion.Location,
		Message: fmt.Sprintf("requirement %s: test %q is not shown to check its statement (%s)", criterion.Ref, test.Name, reason),
	}
}
