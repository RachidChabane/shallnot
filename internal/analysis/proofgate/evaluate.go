package proofgate

import (
	"fmt"
	"sort"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// Evaluate classifies the verified proofs, applies the claim-power table to the valid ones and composes the outcome
// of each criterion in focus: traceability and every required module must pass, and a required module with no valid
// proof is never a pass.
func Evaluate(input Input) Result {
	p := policy{config: input.Config}
	statements := classify(input)
	run := &evaluation{input: input, policy: p, statements: statements, outcomes: map[domain.Ref]Outcome{}}
	run.reportUnusableSet()
	run.reportMalformed()
	criteria := append([]Criterion(nil), input.Criteria...)
	sort.SliceStable(criteria, func(i, j int) bool { return criteria[i].Ref.String() < criteria[j].Ref.String() })
	for _, criterion := range criteria {
		run.evaluateCriterion(criterion)
	}
	run.reportCompleteness()
	return Result{Findings: run.findings, Report: run.report(), Outcomes: run.outcomes}
}

type evaluation struct {
	input      Input
	policy     policy
	statements []classified
	findings   []Finding
	reports    []proof.CriterionReport
	outcomes   map[domain.Ref]Outcome
}

func (e *evaluation) add(finding Finding) { e.findings = append(e.findings, finding) }

func (e *evaluation) reportUnusableSet() {
	evidence := e.input.Evidence
	if evidence.ManifestPresent && evidence.ManifestProblem != "" {
		e.add(Finding{
			Category: domain.CategoryMalformedProof, Effect: EffectBlock, Location: domain.Location{File: evidence.ManifestPath},
			Message: fmt.Sprintf("the manifest of the verified proof set cannot be used, so no proof counts: %s", evidence.ManifestProblem),
		})
	}
}

func (e *evaluation) reportMalformed() {
	for _, statement := range e.statements {
		if statement.State != proof.StateMalformed {
			continue
		}
		e.add(Finding{
			Category: domain.CategoryMalformedProof, Effect: EffectBlock, Location: domain.Location{File: statement.Path},
			Message: fmt.Sprintf("the proof %s is malformed and counts as missing: %s", statement.Path, statement.Reason),
		})
	}
}

// valid returns the valid statements of a kind for a criterion.
func (e *evaluation) valid(kind proof.Kind, criterion Criterion) []proof.Statement {
	var found []proof.Statement
	for _, statement := range e.statements {
		if !statement.valid() || statement.Statement.Kind != kind {
			continue
		}
		requirement, _ := statement.Statement.Requirement()
		if requirement.ID == string(criterion.Ref.ID) && requirement.Revision == int(criterion.Ref.Revision) {
			found = append(found, *statement.Statement)
		}
	}
	return found
}

// staleReasons returns why the statements of a kind for a criterion are not valid.
func (e *evaluation) staleReasons(kind proof.Kind, criterion Criterion) []string {
	var reasons []string
	for _, statement := range e.statements {
		if statement.State != proof.StateStale || statement.Statement.Kind != kind {
			continue
		}
		if requirement, _ := statement.Statement.Requirement(); requirement.ID == string(criterion.Ref.ID) {
			reasons = append(reasons, statement.Reason)
		}
	}
	return uniqueSorted(reasons)
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	unique := values[:0]
	for index, value := range values {
		if index == 0 || value != values[index-1] {
			unique = append(unique, value)
		}
	}
	return unique
}

// pendingFor finds the manifest's pending entry of a module for a criterion.
func (e *evaluation) pendingFor(kind proof.Kind, criterion Criterion) (proof.Pending, bool) {
	for _, pending := range e.input.Evidence.Manifest.Pending {
		if moduleKind, known := proof.KindOfModule(pending.Module); known && moduleKind == kind && pending.Criterion == criterion.Ref.String() {
			return pending, true
		}
	}
	return proof.Pending{}, false
}

// consumedKinds are the kinds whose proofs a module reads: alignment cross-checks its verdicts against the matrix of
// discriminating power.
var consumedKinds = map[proof.Kind][]proof.Kind{proof.KindAlignment: {proof.KindDiscriminatingPower}}

// pendingConsumed finds a pending entry, for the criterion, of a module whose proof the kind consumes.
func (e *evaluation) pendingConsumed(kind proof.Kind, criterion Criterion) (proof.Pending, bool) {
	for _, consumed := range consumedKinds[kind] {
		if pending, isPending := e.pendingFor(consumed, criterion); isPending {
			return pending, true
		}
	}
	return proof.Pending{}, false
}

var pendingCategories = map[proof.PendingReason]domain.Category{
	proof.PendingAwaitingHost:         domain.CategoryAwaitingHost,
	proof.PendingAwaitingProof:        domain.CategoryAwaitingProof,
	proof.PendingAwaitingHostJudgment: domain.CategoryAwaitingHostJudgment,
	proof.PendingHostUnresponsive:     domain.CategoryHostUnresponsive,
}

// absenceEffect is the power of a missing proof: executed evidence that is absent blocks, a judged signal escalates.
func absenceEffect(kind proof.Kind) Effect {
	if kind == proof.KindDiscriminatingPower {
		return EffectBlock
	}
	return EffectEscalate
}

// reportAbsence says that a required module has no valid proof: as the pending reason the manifest gives, else as a stale proof.
func (e *evaluation) reportAbsence(kind proof.Kind, criterion Criterion, scope string, reasons []string) proof.State {
	ref := criterion.Ref.String()
	if pending, isPending := e.pendingFor(kind, criterion); isPending {
		effect := absenceEffect(kind)
		if pending.Reason == proof.PendingAwaitingHostJudgment || pending.Reason == proof.PendingHostUnresponsive {
			effect = EffectEscalate
		}
		e.add(Finding{
			Category: pendingCategories[pending.Reason], Effect: effect, Module: pending.Module, RequirementID: criterion.Ref.ID,
			Location: criterion.Location, Message: pendingMessage(pending.Reason, ref, pending.Module),
		})
		return proof.StatePending
	}
	if waiting, isWaiting := e.pendingConsumed(kind, criterion); isWaiting {
		// A module that consumes the proof of another cannot have its own while the other is pending.
		e.add(Finding{
			Category: domain.CategoryAwaitingProof, Effect: absenceEffect(kind), Module: waiting.Module, RequirementID: criterion.Ref.ID,
			Location: criterion.Location, Message: pendingMessage(proof.PendingAwaitingProof, ref, waiting.Module),
		})
		return proof.StatePending
	}
	e.add(Finding{
		Category: domain.CategoryStaleProof, Effect: absenceEffect(kind), Module: kind.ModuleName(), RequirementID: criterion.Ref.ID,
		Location: criterion.Location, Message: staleMessage(ref, kind, scope, reasons),
	})
	if len(reasons) > 0 {
		return proof.StateStale
	}
	return proof.StateMissing
}

func (e *evaluation) evaluateCriterion(criterion Criterion) {
	report := proof.CriterionReport{
		Ref: criterion.Ref.String(), Modules: map[proof.Kind]proof.ModuleReport{}, Absent: []proof.Kind{},
		Channels: map[proof.Channel]int{}, Isolation: map[proof.Isolation]int{},
	}
	first := len(e.findings)
	tainted := map[string]bool{}

	dp := e.evaluateDP(criterion, &report)
	if dp != nil {
		tainted = dp.tainted
	}
	e.evaluateAlignment(criterion, tainted, &report)

	e.outcomes[criterion.Ref] = outcomeOf(e.findings[first:])
	e.reports = append(e.reports, report)
}

func (e *evaluation) evaluateDP(criterion Criterion, report *proof.CriterionReport) *dpEvaluation {
	kind := proof.KindDiscriminatingPower
	required := e.input.Config.Requires(kind)
	found := e.valid(kind, criterion)
	switch {
	case len(found) > 1:
		e.add(e.ambiguity(kind, criterion, ""))
		found = nil
	case len(found) == 0 && !required:
		return nil
	}
	if len(found) == 0 {
		state := e.reportAbsence(kind, criterion, "", e.staleReasons(kind, criterion))
		report.Modules[kind] = proof.ModuleReport{State: state, Outcome: proof.OutcomeAbsent}
		report.Absent = append(report.Absent, kind)
		return nil
	}
	evaluation := e.policy.evaluateDiscriminatingPower(criterion, found[0].DiscriminatingPower)
	e.findings = append(e.findings, evaluation.findings...)
	report.Modules[kind] = proof.ModuleReport{State: proof.StateValid, Outcome: moduleOutcome(evaluation.findings, evaluation.credited > 0)}
	report.HostExclusions += evaluation.hostExclusions
	report.ScreenExclusions += evaluation.screenExclusions
	mergeCounts(report, evaluation.channels, evaluation.isolation)
	return &evaluation
}

func (e *evaluation) ambiguity(kind proof.Kind, criterion Criterion, scope string) Finding {
	return Finding{
		Category: domain.CategoryMalformedProof, Effect: EffectBlock, Module: kind.ModuleName(), RequirementID: criterion.Ref.ID, Location: criterion.Location,
		Message: fmt.Sprintf("requirement %s holds several valid %s proofs%s: which one counts is undecidable, so none does", criterion.Ref, kind, scope),
	}
}

func (e *evaluation) evaluateAlignment(criterion Criterion, tainted map[string]bool, report *proof.CriterionReport) {
	kind := proof.KindAlignment
	required := e.input.Config.Requires(kind)
	found := e.valid(kind, criterion)
	if len(found) == 0 && !required {
		return
	}
	first := len(e.findings)
	var absent []string
	for _, test := range criterion.PassingTests {
		var matching []*proof.Alignment
		for _, statement := range found {
			if matches(statement.Alignment.Test, test) {
				matching = append(matching, statement.Alignment)
			}
		}
		switch len(matching) {
		case 0:
			absent = append(absent, test.Name)
		case 1:
			e.settleAlignment(criterion, test, matching[0], tainted, report)
		default:
			e.add(e.ambiguity(kind, criterion, forTest(test.Name)))
		}
	}
	state := proof.StateValid
	outcome := moduleOutcome(e.findings[first:], len(absent) == 0)
	if len(absent) > 0 && required {
		state = e.reportAbsence(kind, criterion, forTest(joinNames(absent)), e.staleReasons(kind, criterion))
		outcome = proof.OutcomeAbsent
		report.Absent = append(report.Absent, kind)
	} else if len(found) == 0 {
		return
	}
	if len(absent) > 0 && !required {
		state = proof.StateMissing
	}
	report.Modules[kind] = proof.ModuleReport{State: state, Outcome: outcome}
}

func joinNames(names []string) string {
	joined := ""
	for index, name := range names {
		if index > 0 {
			joined += listSeparator
		}
		joined += name
	}
	return joined
}

func (e *evaluation) settleAlignment(criterion Criterion, test TestName, predicate *proof.Alignment, tainted map[string]bool, report *proof.CriterionReport) {
	executor := executorOf(predicate.Verdict, predicate.Executor)
	report.Channels[executor.Channel]++
	report.Isolation[executor.Isolation]++
	if clearance := e.policy.clearance(predicate, test, tainted); !clearance.cleared {
		e.add(e.policy.alignmentFinding(criterion, test, clearance.reason))
	}
}

func mergeCounts(report *proof.CriterionReport, channels map[proof.Channel]int, isolation map[proof.Isolation]int) {
	for channel, count := range channels {
		report.Channels[channel] += count
	}
	for level, count := range isolation {
		report.Isolation[level] += count
	}
}

// moduleOutcome is the strongest effect among a module's findings, and a pass only when nothing was found and the module earned its credit.
func moduleOutcome(findings []Finding, earned bool) proof.ModuleOutcome {
	switch outcomeOf(findings) {
	case OutcomeBlock:
		return proof.OutcomeBlock
	case OutcomeEscalate:
		return proof.OutcomeEscalate
	}
	if !earned {
		return proof.OutcomeEscalate
	}
	return proof.OutcomePass
}

// reportCompleteness turns the questions of each valid completeness proof into informational findings.
func (e *evaluation) reportCompleteness() {
	for _, statement := range e.statements {
		if !statement.valid() || statement.Statement.Kind != proof.KindCompleteness {
			continue
		}
		spec := statement.Statement.Subject[0].Name
		for _, question := range statement.Statement.Completeness.Questions {
			message := fmt.Sprintf("completeness (%s): %s", question.Dimension, question.Question)
			if len(question.RequirementIDs) > 0 {
				message += fmt.Sprintf(" [%s]", joinNames(question.RequirementIDs))
			}
			e.add(Finding{Category: domain.CategoryCompletenessQuestion, Effect: EffectInform, Module: proof.KindCompleteness.ModuleName(), Location: domain.Location{File: spec}, Message: message})
		}
	}
}

func (e *evaluation) report() proof.Report {
	config := e.input.Config
	evidence := e.input.Evidence
	report := proof.Report{
		Trust: config.Trust,
		Policy: proof.PolicyReport{
			SignedPassBasis: config.SignedPassBasis, MinIsolation: config.MinIsolation, MaxHostExclusions: config.MaxHostExclusions,
			ValidityThreshold: config.Thresholds.Validity, AlignmentThreshold: config.Thresholds.Alignment, Escalation: config.Escalation,
			IsolationFloorLowered: config.MinIsolation.Strength() < proof.DefaultMinIsolation.Strength(),
		},
		Consumed: consumedReport(e.statements),
		Pending:  append([]proof.Pending{}, evidence.Manifest.Pending...),
		Criteria: append([]proof.CriterionReport{}, e.reports...),
	}
	if evidence.ManifestPresent {
		report.Manifest = &proof.ManifestReport{
			Path: evidence.ManifestPath, Digest: evidence.ManifestDigest, VerifierVersion: evidence.Manifest.VerifierVersion,
			TrustedRootDigest: evidence.Manifest.TrustedRootDigest, PolicyDigest: evidence.Manifest.PolicyDigest,
		}
	}
	return report
}
