package proofgate

import (
	"fmt"
	"sort"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// disposition is what the claim-power table makes of one mutant.
type disposition uint8

const (
	dispositionNone disposition = iota
	// dispositionBlock: a valid mutant survived every test.
	dispositionBlock
	// dispositionEscalate: a survivor the evidence does not settle.
	dispositionEscalate
	// dispositionExcludedExecuted: a survivor excluded on executed evidence.
	dispositionExcludedExecuted
	// dispositionExcludedHost: a survivor excluded on a judgment alone.
	dispositionExcludedHost
	// dispositionCredit: a valid mutant killed by a test that earns kill credit.
	dispositionCredit
	// dispositionNoCredit: a valid mutant killed, but not by evidence that earns credit.
	dispositionNoCredit
)

// verdictOnMutant is a disposition and the reason it was reached.
type verdictOnMutant struct {
	disposition disposition
	reason      string
}

// dpEvaluation is what a discriminating-power proof concluded.
type dpEvaluation struct {
	findings         []Finding
	credited         int
	validMutants     int
	hostExclusions   int
	screenExclusions int
	// tainted holds the tests that killed a sentinel.
	tainted   map[string]bool
	channels  map[proof.Channel]int
	isolation map[proof.Isolation]int
}

type mutantFacts struct {
	survived bool
	killedBy []string
}

func factsOf(mutant proof.Mutant) mutantFacts {
	facts := mutantFacts{survived: len(mutant.Results) > 0}
	for _, result := range mutant.Results {
		switch result.Outcome.Value {
		case proof.OutcomeKilled:
			facts.killedBy = append(facts.killedBy, result.Test)
			facts.survived = false
		case proof.OutcomeError:
			facts.survived = false
		}
	}
	return facts
}

// evaluateDiscriminatingPower applies the claim-power table to the matrix of one criterion.
func (p policy) evaluateDiscriminatingPower(criterion Criterion, predicate *proof.DiscriminatingPower) dpEvaluation {
	evaluation := dpEvaluation{tainted: map[string]bool{}, channels: map[proof.Channel]int{}, isolation: map[proof.Isolation]int{}}
	emit := func(category domain.Category, effect Effect, message string) {
		evaluation.findings = append(evaluation.findings, Finding{
			Category: category, Effect: effect, Module: proof.KindDiscriminatingPower.ModuleName(),
			RequirementID: criterion.Ref.ID, Location: criterion.Location, Message: message,
		})
	}
	ref := criterion.Ref.String()
	if predicate.Extensions.OperatorScreen != nil {
		evaluation.screenExclusions = predicate.Extensions.OperatorScreen.ScreenedOut
	}
	if !predicate.CorrectCodePasses.Value {
		emit(domain.CategoryCorrectCodeFails, EffectBlock, fmt.Sprintf("requirement %s: its tests do not pass on the reference code", ref))
	}
	for _, example := range predicate.Examples {
		if example.Result.Value == proof.ExampleFailed {
			emit(domain.CategoryExampleFailed, EffectBlock, fmt.Sprintf("requirement %s: example %d fails on the reference code", ref, example.Index))
		}
	}

	mutants := append([]proof.Mutant(nil), predicate.Mutants...)
	sort.SliceStable(mutants, func(i, j int) bool { return mutants[i].Digest < mutants[j].Digest })
	p.taintSentinelKillers(&evaluation, mutants, emit, ref)

	blocked := false
	for _, mutant := range mutants {
		if mutant.Sentinel {
			continue
		}
		executor := executorOf(mutant.Validity, predicate.Executor)
		evaluation.channels[executor.Channel]++
		evaluation.isolation[executor.Isolation]++
		if mutant.Validity.Value == proof.ValidityValid {
			evaluation.validMutants++
		}
		verdict := p.judgeMutant(mutant, predicate.Executor, &evaluation)
		name := abbreviate(mutant.Digest)
		switch verdict.disposition {
		case dispositionBlock:
			blocked = true
			emit(domain.CategorySurvivingValidMutant, EffectBlock, fmt.Sprintf("requirement %s: mutant %s violates it and survived every test (%s)", ref, name, verdict.reason))
		case dispositionEscalate:
			emit(domain.CategoryEscalatedMutant, EffectEscalate, fmt.Sprintf("requirement %s: mutant %s survived every test and the evidence does not settle it (%s)", ref, name, verdict.reason))
		case dispositionCredit:
			evaluation.credited++
		}
	}
	switch {
	case evaluation.validMutants == 0:
		emit(domain.CategoryNoValidMutant, EffectEscalate, fmt.Sprintf("requirement %s: no mutant was judged to violate it, so the tests were not shown to discriminate", ref))
	case evaluation.credited == 0 && !blocked:
		emit(domain.CategoryNoKilledValidMutant, EffectEscalate, fmt.Sprintf("requirement %s: no test killed a valid mutant on evidence that earns pass credit", ref))
	}
	return evaluation
}

// taintSentinelKillers records the tests that killed a sentinel mutant, which is equivalent to the reference by construction.
func (p policy) taintSentinelKillers(evaluation *dpEvaluation, mutants []proof.Mutant, emit func(domain.Category, Effect, string), ref string) {
	for _, mutant := range mutants {
		if !mutant.Sentinel {
			continue
		}
		for _, test := range factsOf(mutant).killedBy {
			if evaluation.tainted[test] {
				continue
			}
			evaluation.tainted[test] = true
			emit(domain.CategoryMutationSensitiveTest, EffectEscalate,
				fmt.Sprintf("requirement %s: test %q killed a sentinel mutant that is equivalent to the reference, so it earns no kill credit and its alignment is undetermined", ref, test))
		}
	}
}

// judgeMutant is the claim-power table of the proof format, one case per row.
func (p policy) judgeMutant(mutant proof.Mutant, predicateExecutor proof.Executor, evaluation *dpEvaluation) verdictOnMutant {
	facts := factsOf(mutant)
	validity := mutant.Validity
	executor := executorOf(validity, predicateExecutor)
	switch {
	case facts.survived:
		return p.judgeSurvivor(mutant, executor, evaluation)
	case len(facts.killedBy) > 0 && validity.Value == proof.ValidityValid:
		return p.judgeKill(mutant, facts, executor, evaluation.tainted)
	}
	return verdictOnMutant{}
}

func (p policy) judgeSurvivor(mutant proof.Mutant, executor proof.Executor, evaluation *dpEvaluation) verdictOnMutant {
	validity := mutant.Validity
	confidence := validity.ConfidenceOrZero()
	switch validity.Value {
	case proof.ValidityValid:
		switch {
		case has(validity, ownerExample...):
			return verdictOnMutant{dispositionBlock, "it fails an example of the criterion"}
		case p.confidentValidity(confidence) && has(validity, blockWitness...):
			return verdictOnMutant{dispositionBlock, "judged VALID with a confirmed witness, a build and the other criteria's tests passing"}
		case !p.confidentValidity(confidence):
			return verdictOnMutant{dispositionEscalate, fmt.Sprintf("judged VALID with confidence %.2f, below %.2f", confidence, p.config.Thresholds.Validity)}
		default:
			return verdictOnMutant{dispositionEscalate, "judged VALID without a confirmed witness, a build and the other criteria's tests passing"}
		}
	case proof.ValidityEquivalent:
		if has(validity, equivalentProof...) {
			return verdictOnMutant{dispositionExcludedExecuted, ""}
		}
		return verdictOnMutant{dispositionEscalate, "judged EQUIVALENT without a differential search that found no difference"}
	case proof.ValidityNotViolating:
		return p.judgeNotViolatingSurvivor(validity, executor, evaluation)
	default:
		return verdictOnMutant{dispositionEscalate, "its validity is UNDETERMINED"}
	}
}

// judgeNotViolatingSurvivor decides on a survivor judged not to violate the criterion, which covers off-target and
// unattributed mutants: an executed example of another criterion excludes it; a judgment alone does so only where the
// basis allows and the budget of host exclusions is not spent.
func (p policy) judgeNotViolatingSurvivor(validity proof.Claim[proof.Validity], executor proof.Executor, evaluation *dpEvaluation) verdictOnMutant {
	switch {
	case has(validity, exclusionExample...):
		return verdictOnMutant{dispositionExcludedExecuted, ""}
	case p.basis() != proof.BasisJudged:
		return verdictOnMutant{dispositionEscalate, fmt.Sprintf("judged NOT_VIOLATING with no example of another criterion failing; signed_pass_basis %s excludes no survivor on a judgment", p.basis())}
	case !p.confidentValidity(validity.ConfidenceOrZero()):
		return verdictOnMutant{dispositionEscalate, fmt.Sprintf("judged NOT_VIOLATING with confidence %.2f, below %.2f", validity.ConfidenceOrZero(), p.config.Thresholds.Validity)}
	case !p.aboveFloor(executor):
		return verdictOnMutant{dispositionEscalate, fmt.Sprintf("judged NOT_VIOLATING by an executor of isolation %s, below the floor %s", executor.Isolation, p.config.MinIsolation)}
	case evaluation.hostExclusions >= p.config.MaxHostExclusions:
		return verdictOnMutant{dispositionEscalate, fmt.Sprintf("judged NOT_VIOLATING, but %d survivors are already excluded on a judgment alone", p.config.MaxHostExclusions)}
	}
	evaluation.hostExclusions++
	return verdictOnMutant{dispositionExcludedHost, ""}
}

// judgeKill decides whether a valid mutant that a test killed earns pass credit. Tests that killed a sentinel earn none.
func (p policy) judgeKill(mutant proof.Mutant, facts mutantFacts, executor proof.Executor, tainted map[string]bool) verdictOnMutant {
	validity := mutant.Validity
	earning := 0
	for _, test := range facts.killedBy {
		if !tainted[test] {
			earning++
		}
	}
	switch {
	case earning == 0:
		return verdictOnMutant{dispositionNoCredit, "every test that killed it also killed a sentinel"}
	case has(validity, ownerExample...):
		return verdictOnMutant{dispositionCredit, ""}
	case p.basis() == proof.BasisExecutedOnly:
		return verdictOnMutant{dispositionNoCredit, "signed_pass_basis executed-only gives no credit to a judged validity"}
	case !p.confidentValidity(validity.ConfidenceOrZero()):
		return verdictOnMutant{dispositionNoCredit, "the validity is judged below the threshold"}
	case !p.aboveFloor(executor):
		return verdictOnMutant{dispositionNoCredit, "the executor's isolation is below the floor"}
	case p.basis() == proof.BasisCorroborated && !has(validity, creditEvidence...):
		return verdictOnMutant{dispositionNoCredit, "the validity lacks a specific kill, a build, the other criteria's tests passing and a confirmed witness"}
	}
	return verdictOnMutant{dispositionCredit, ""}
}
