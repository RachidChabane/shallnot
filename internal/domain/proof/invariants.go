package proof

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
)

// Validate checks every invariant of the format on a decoded statement: the
// Statement v1 `_type`, the subject rule of the predicate, the claim rules, the
// closed enums, the sort orders and the corroboration invariant. It returns
// every violation found; none means the statement is well formed.
func (s Statement) Validate() []Violation {
	var sink collector
	if s.Type != StatementType {
		sink.add(CodeStatementType, "_type", "%q is not %q", s.Type, StatementType)
	}
	if expected := s.Kind.PredicateType(); s.PredicateType != expected {
		sink.add(CodePredicateType, "predicateType", "%q is not %q", s.PredicateType, expected)
	}
	switch s.Kind {
	case KindAlignment:
		s.Alignment.check(&sink, s.Subject)
	case KindDiscriminatingPower:
		s.DiscriminatingPower.check(&sink, s.Subject)
	case KindCompleteness:
		s.Completeness.check(&sink, s.Subject)
	}
	return sink.violations
}

func confidenceInRange(confidence float64) bool {
	return !math.IsNaN(confidence) && confidence >= MinConfidence && confidence <= MaxConfidence
}

// check reports every violation of the claim rules at path. allowed is the
// class the predicate permits for this claim.
func (c Claim[V]) check(sink *collector, path string, allowed Class) {
	if c.Class != ClassExecuted && c.Class != ClassJudged {
		sink.add(CodeUnknownValue, path+".class", "class %q is neither %q nor %q", c.Class, ClassExecuted, ClassJudged)
		return
	}
	if c.Class != allowed {
		sink.add(CodeWrongClass, path+".class", "class %q where the predicate requires %q", c.Class, allowed)
	}
	switch {
	case c.Class == ClassJudged && c.Confidence == nil:
		sink.add(CodeJudgedWithoutConfidence, path, "a judged claim requires a confidence")
	case c.Class == ClassJudged && !confidenceInRange(*c.Confidence):
		sink.add(CodeConfidenceRange, path+".confidence", "confidence %v is outside [%v,%v]", *c.Confidence, MinConfidence, MaxConfidence)
	case c.Class == ClassExecuted && c.Confidence != nil:
		sink.add(CodeExecutedWithConfidence, path, "an executed claim forbids a confidence")
	}
	c.checkJudgmentMembers(sink, path)
}

// checkJudgmentMembers checks the members only a judged claim may carry.
func (c Claim[V]) checkJudgmentMembers(sink *collector, path string) {
	if c.Class == ClassExecuted {
		if len(c.CorroboratedBy) > 0 {
			sink.add(CodeExecutedWithCorroboration, path+".corroboratedBy", "an executed claim forbids corroboration")
		}
		if c.Executor != nil {
			sink.add(CodeExecutedWithExecutor, path+".executor", "an executed claim forbids an executor")
		}
		return
	}
	checkCorroborations(sink, path+".corroboratedBy", c.CorroboratedBy)
	if c.Executor != nil {
		c.Executor.check(sink, path+".executor")
	}
}

func checkCorroborations(sink *collector, path string, corroborations []Corroboration) {
	for index, corroboration := range corroborations {
		entry := fmt.Sprintf("%s[%d]", path, index)
		checkMember(sink, entry+".kind", corroboration.Kind, corroborationKinds)
		checkDigest(sink, entry+".digest", corroboration.Digest)
	}
	checkAscending(sink, path, corroborations, func(a, b Corroboration) int {
		return cmp.Or(strings.Compare(string(a.Kind), string(b.Kind)), strings.Compare(a.Digest, b.Digest))
	})
}

func (e ExecutorProfile) check(sink *collector, path string) {
	checkMember(sink, path+".channel", e.Channel, channels)
	checkNotEmpty(sink, path+".harness.name", e.Harness.Name)
	checkMember(sink, path+".isolation", e.Isolation, isolations)
	checkMember(sink, path+".model.source", e.Model.Source, modelSources)
	if e.Model.Source != ModelSourceUnknown {
		checkNotEmpty(sink, path+".model.name", e.Model.Name)
	}
	e.Attestation.check(sink, path+".attestation", e.Isolation == IsolationRestrictedAttested)
}

func (a Attestation) check(sink *collector, path string, required bool) {
	if required {
		checkNotEmpty(sink, path+".agentId", a.AgentID)
		checkDigest(sink, path+".capDigest", a.CapDigest)
		checkDigest(sink, path+".transcriptDigest", a.TranscriptDigest)
		return
	}
	if a.CapDigest != "" {
		checkDigest(sink, path+".capDigest", a.CapDigest)
	}
	if a.TranscriptDigest != "" {
		checkDigest(sink, path+".transcriptDigest", a.TranscriptDigest)
	}
}

func (e Executor) check(sink *collector, path string) {
	e.ExecutorProfile.check(sink, path)
	checkDigest(sink, path+".promptDigest", e.PromptDigest)
}

func (p *Alignment) check(sink *collector, subject []Subject) {
	checkRequirement(sink, "predicate.requirement", p.Requirement)
	checkRequirementSubject(sink, subject, p.Requirement)
	checkCore(sink, "predicate", p.Core, subject)
	checkNotEmpty(sink, "predicate.test", p.Test)
	p.Verdict.check(sink, "predicate.verdict", ClassJudged)
	checkMember(sink, "predicate.verdict.value", p.Verdict.Value, verdicts)
	p.Executor.check(sink, "predicate.executor")
	if !p.Executor.Blind {
		sink.add(CodeNotBlind, "predicate.executor.blind", "the judge must have been blind")
	}
}

func (p *DiscriminatingPower) check(sink *collector, subject []Subject) {
	checkRequirement(sink, "predicate.requirement", p.Requirement)
	checkRequirementSubject(sink, subject, p.Requirement)
	checkCore(sink, "predicate", p.Core, subject)
	checkNotEmptyList(sink, "predicate.tests", p.Tests)
	checkAscending(sink, "predicate.tests", p.Tests, strings.Compare)
	p.CorrectCodePasses.check(sink, "predicate.correctCodePasses", ClassExecuted)
	p.Executor.check(sink, "predicate.executor")
	checkAscending(sink, "predicate.mutants", p.Mutants, func(a, b Mutant) int { return strings.Compare(a.Digest, b.Digest) })
	for index, mutant := range p.Mutants {
		checkMutant(sink, fmt.Sprintf("predicate.mutants[%d]", index), mutant)
	}
	checkAscending(sink, "predicate.examples", p.Examples, func(a, b ExampleRun) int { return cmp.Compare(a.Index, b.Index) })
	for index, example := range p.Examples {
		path := fmt.Sprintf("predicate.examples[%d]", index)
		example.Result.check(sink, path+".result", ClassExecuted)
		checkMember(sink, path+".result.value", example.Result.Value, exampleResults)
	}
}

func (p *Completeness) check(sink *collector, subject []Subject) {
	checkSpecSubject(sink, subject)
	checkCore(sink, "predicate", p.Core, subject)
	if !p.Advisory {
		sink.add(CodeNotAdvisory, "predicate.advisory", "a completeness proof is advisory")
	}
	checkAscending(sink, "predicate.questions", p.Questions, func(a, b Question) int {
		return cmp.Or(strings.Compare(a.Dimension, b.Dimension), strings.Compare(a.Question, b.Question))
	})
	for index, question := range p.Questions {
		path := fmt.Sprintf("predicate.questions[%d]", index)
		checkNotEmpty(sink, path+".dimension", question.Dimension)
		checkNotEmpty(sink, path+".question", question.Question)
		checkAscending(sink, path+".requirementIds", question.RequirementIDs, strings.Compare)
	}
	checkAscending(sink, "predicate.outOfScopeDeclared", p.OutOfScopeDeclared, func(a, b OutOfScope) int {
		return cmp.Or(strings.Compare(a.Dimension, b.Dimension), strings.Compare(a.Reason, b.Reason))
	})
	for index, declared := range p.OutOfScopeDeclared {
		path := fmt.Sprintf("predicate.outOfScopeDeclared[%d]", index)
		checkNotEmpty(sink, path+".dimension", declared.Dimension)
		checkNotEmpty(sink, path+".reason", declared.Reason)
	}
}

func checkMutant(sink *collector, path string, mutant Mutant) {
	checkDigest(sink, path+".digest", mutant.Digest)
	mutant.Validity.check(sink, path+".validity", ClassJudged)
	checkMember(sink, path+".validity.value", mutant.Validity.Value, validities)
	if mutant.Sentinel && mutant.Validity.Value != ValidityEquivalent {
		sink.add(CodeSentinelNotEquivalent, path+".validity.value", "a sentinel mutant is EQUIVALENT by construction, not %q", mutant.Validity.Value)
	}
	checkNotEmptyList(sink, path+".results", mutant.Results)
	checkAscending(sink, path+".results", mutant.Results, func(a, b TestResult) int { return strings.Compare(a.Test, b.Test) })
	for index, result := range mutant.Results {
		resultPath := fmt.Sprintf("%s.results[%d]", path, index)
		checkNotEmpty(sink, resultPath+".test", result.Test)
		result.Outcome.check(sink, resultPath+".outcome", ClassExecuted)
		checkMember(sink, resultPath+".outcome.value", result.Outcome.Value, outcomes)
	}
	checkCorroborationRequired(sink, path+".validity", mutant.Validity)
	if mutant.Extensions != nil {
		checkDecision(sink, path+".extensions.decision", mutant.Extensions.Decision)
		checkAttribution(sink, path, mutant)
	}
}

// checkCorroborationRequired enforces that a judged claim with the power to
// block or to earn credit rests on executed evidence.
func checkCorroborationRequired(sink *collector, path string, validity Claim[Validity]) {
	if validity.Class == ClassJudged && validity.Value == ValidityValid && len(validity.CorroboratedBy) == 0 {
		sink.add(CodeUncorroboratedClaim, path+".corroboratedBy", "a judged %s claim needs executed corroboration", validity.Value)
	}
}

// checkAttribution enforces that an unattributed or off-target mutant is judged
// NOT_VIOLATING, and that no mutant is both.
func checkAttribution(sink *collector, path string, mutant Mutant) {
	extensions := mutant.Extensions
	if (extensions.Unattributed || extensions.OffTarget != "") && mutant.Validity.Value != ValidityNotViolating {
		sink.add(CodeInconsistentAttribution, path+".validity.value", "an unattributed or off-target mutant is %q, not %q", ValidityNotViolating, mutant.Validity.Value)
	}
	if extensions.Unattributed && extensions.OffTarget != "" {
		sink.add(CodeInconsistentAttribution, path+".extensions", "a mutant is unattributed or off-target, not both")
	}
}

func checkCore(sink *collector, path string, core Core, subject []Subject) {
	checkNotEmpty(sink, path+".module.name", core.Module.Name)
	checkNotEmpty(sink, path+".module.version", core.Module.Version)
	checkNotEmptyList(sink, path+".inputs", core.Inputs)
	checkAscending(sink, path+".inputs", core.Inputs, func(a, b Input) int {
		return cmp.Or(strings.Compare(string(a.Role), string(b.Role)), strings.Compare(a.Name, b.Name))
	})
	for index, input := range core.Inputs {
		checkInput(sink, fmt.Sprintf("%s.inputs[%d]", path, index), input)
	}
	checkDigest(sink, path+".cacheKey", core.CacheKey)
	checkCacheKey(sink, path+".cacheKey", core, subject)
	checkMember(sink, path+".producedBy", core.ProducedBy, producers)
	checkDecision(sink, path+".extensions.decision", core.Extensions.Decision)
}

func checkInput(sink *collector, path string, input Input) {
	checkMember(sink, path+".role", input.Role, inputRoles)
	checkNotEmpty(sink, path+".name", input.Name)
	checkDigest(sink, path+".digest.sha256", input.Digest.SHA256)
	if input.Annotations != nil {
		checkAscending(sink, path+".annotations.globs", input.Annotations.Globs, strings.Compare)
	}
}

func checkCacheKey(sink *collector, path string, core Core, subject []Subject) {
	if !IsSHA256Hex(core.CacheKey) {
		return
	}
	expected, err := CacheKey(subject, core.Inputs)
	if err != nil {
		sink.add(CodeCacheKeyMismatch, path, "cache key cannot be recomputed: %v", err)
		return
	}
	if expected != core.CacheKey {
		sink.add(CodeCacheKeyMismatch, path, "cache key %s is not the digest of subject and inputs %s", core.CacheKey, expected)
	}
}

func checkDecision(sink *collector, path string, decision *Decision) {
	if decision == nil {
		return
	}
	checkMember(sink, path+".basis", decision.Basis, bases)
	checkNotEmptyList(sink, path+".deciders", decision.Deciders)
	checkAscending(sink, path+".deciders", decision.Deciders, strings.Compare)
	checkConfidence(sink, path+".rawConfidence", decision.RawConfidence)
	if decision.CalibratedConfidence != nil {
		checkConfidence(sink, path+".calibratedConfidence", *decision.CalibratedConfidence)
	}
	checkConfidence(sink, path+".threshold", decision.Threshold)
}

func checkConfidence(sink *collector, path string, value float64) {
	if !confidenceInRange(value) {
		sink.add(CodeConfidenceRange, path, "%v is outside [%v,%v]", value, MinConfidence, MaxConfidence)
	}
}

func checkRequirement(sink *collector, path string, requirement Requirement) {
	if requirement.ID == "" || strings.Contains(requirement.ID, RevisionSeparator) {
		sink.add(CodeRequirement, path+".id", "%q is not a requirement ID", requirement.ID)
	}
	if requirement.Revision < MinimumRevision {
		sink.add(CodeRequirement, path+".revision", "revision %d is below %d", requirement.Revision, MinimumRevision)
	}
	if requirement.StatementCanonicalization != StatementCanonicalizationC1 {
		sink.add(CodeRequirement, path+".statementCanonicalization", "%q is not %q", requirement.StatementCanonicalization, StatementCanonicalizationC1)
	}
}

// checkRequirementSubject enforces the subject rule of alignment and
// discriminating power: the criterion `ID~REV` and its statement digest.
func checkRequirementSubject(sink *collector, subject []Subject, requirement Requirement) {
	if !checkSingleSubject(sink, subject) {
		return
	}
	expected := SubjectName(requirement.ID, requirement.Revision)
	if subject[0].Name != expected {
		sink.add(CodeSubject, "subject[0].name", "%q is not the requirement %q", subject[0].Name, expected)
	}
}

// checkSpecSubject enforces the subject rule of completeness: a spec path and the hash of the file.
func checkSpecSubject(sink *collector, subject []Subject) {
	if !checkSingleSubject(sink, subject) {
		return
	}
	if !IsRelativePOSIXPath(subject[0].Name) {
		sink.add(CodeSubject, "subject[0].name", "%q is not a relative POSIX spec path", subject[0].Name)
	}
}

func checkSingleSubject(sink *collector, subject []Subject) bool {
	if len(subject) != 1 {
		sink.add(CodeSubject, "subject", "a statement has exactly one subject, found %d", len(subject))
		return false
	}
	if !IsSHA256Hex(subject[0].Digest.SHA256) {
		sink.add(CodeBadDigest, "subject[0].digest.sha256", "%q is not a SHA-256 hex digest", subject[0].Digest.SHA256)
	}
	return true
}

func checkDigest(sink *collector, path, value string) {
	if !IsSHA256Hex(value) {
		sink.add(CodeBadDigest, path, "%q is not a SHA-256 hex digest", value)
	}
}

func checkNotEmpty(sink *collector, path, value string) {
	if value == "" {
		sink.add(CodeMissingField, path, "must not be empty")
	}
}

func checkNotEmptyList[T any](sink *collector, path string, items []T) {
	if len(items) == 0 {
		sink.add(CodeMissingField, path, "must not be empty")
	}
}

func checkMember[T comparable](sink *collector, path string, value T, allowed []T) {
	if !slices.Contains(allowed, value) {
		sink.add(CodeUnknownValue, path, "%v is not one of %v", value, allowed)
	}
}

// checkAscending reports an array that is not strictly ascending under compare.
// Equal neighbours count as a violation, so duplicates are rejected too.
func checkAscending[T any](sink *collector, path string, items []T, compare func(a, b T) int) {
	for index := 1; index < len(items); index++ {
		if compare(items[index-1], items[index]) >= 0 {
			sink.add(CodeUnsorted, path, "element %d is not strictly after element %d", index, index-1)
			return
		}
	}
}
