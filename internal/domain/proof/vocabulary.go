// Package proof is the pure model of the proofs the gate consumes: the in-toto
// Statement v1 envelope, the three predicates of the proof format, the
// invariants every statement must hold, the digests the gate recomputes and the
// policy that turns valid proofs into a per-criterion outcome. It performs no
// I/O and links no schema library.
package proof

import "strconv"

// Statement and predicate identifiers of the proof format.
const (
	// StatementType is the in-toto Statement v1 `_type`.
	StatementType = "https://in-toto.io/Statement/v1"

	PredicateTypeAlignment           = "https://rachid-chabane.com/attestations/alignment/v1"
	PredicateTypeDiscriminatingPower = "https://rachid-chabane.com/attestations/discriminating-power/v1"
	PredicateTypeCompleteness        = "https://rachid-chabane.com/attestations/completeness/v1"

	// Sha256HexLength is the length of a hex-encoded SHA-256 digest.
	Sha256HexLength = 64

	// StatementCanonicalizationC1 names the statement canonicalization the digests of subjects are computed with.
	StatementCanonicalizationC1 = "c1"

	// RevisionSeparator separates a requirement ID from its revision in `ID~REV`.
	RevisionSeparator = "~"
	// MinimumRevision is the first revision of a requirement.
	MinimumRevision = 1

	// MinConfidence and MaxConfidence bound the confidence of a judged claim.
	MinConfidence = 0.0
	MaxConfidence = 1.0
)

// Kind names one proof module and the predicate it produces.
type Kind string

// The three proof kinds.
const (
	KindAlignment           Kind = "alignment"
	KindDiscriminatingPower Kind = "discriminating-power"
	KindCompleteness        Kind = "completeness"
)

var predicateTypes = map[Kind]string{
	KindAlignment:           PredicateTypeAlignment,
	KindDiscriminatingPower: PredicateTypeDiscriminatingPower,
	KindCompleteness:        PredicateTypeCompleteness,
}

// Kinds lists every proof kind in a stable order.
func Kinds() []Kind {
	return []Kind{KindAlignment, KindDiscriminatingPower, KindCompleteness}
}

// PredicateType returns the predicate type URI of the kind.
func (k Kind) PredicateType() string { return predicateTypes[k] }

// KindOfPredicateType resolves a predicate type URI to its kind.
func KindOfPredicateType(predicateType string) (Kind, bool) {
	for _, kind := range Kinds() {
		if predicateTypes[kind] == predicateType {
			return kind, true
		}
	}
	return "", false
}

// KindOfName resolves the wire name of a kind, the name a `proof` input carries.
func KindOfName(name string) (Kind, bool) {
	for _, kind := range Kinds() {
		if string(kind) == name {
			return kind, true
		}
	}
	return "", false
}

// Class is the evidence class of a claim.
type Class string

// Evidence classes.
const (
	ClassExecuted Class = "executed"
	ClassJudged   Class = "judged"
)

// Verdict is the judged alignment verdict.
type Verdict string

// Alignment verdicts.
const (
	VerdictAligned      Verdict = "ALIGNED"
	VerdictMisaligned   Verdict = "MISALIGNED"
	VerdictUndetermined Verdict = "UNDETERMINED"
)

// Validity is the judged validity of a mutant.
type Validity string

// Mutant validity values.
const (
	ValidityValid        Validity = "VALID"
	ValidityEquivalent   Validity = "EQUIVALENT"
	ValidityNotViolating Validity = "NOT_VIOLATING"
	ValidityUndetermined Validity = "UNDETERMINED"
)

// Outcome is the executed outcome of one test against one mutant.
type Outcome string

// Mutant outcomes.
const (
	OutcomeKilled   Outcome = "killed"
	OutcomeSurvived Outcome = "survived"
	OutcomeError    Outcome = "error"
)

// ExampleResult is the executed result of one spec example on the reference.
type ExampleResult string

// Example results.
const (
	ExamplePassed ExampleResult = "passed"
	ExampleFailed ExampleResult = "failed"
)

// ProducedBy tells where a statement was produced.
type ProducedBy string

// Producers.
const (
	ProducedByLocal ProducedBy = "local"
	ProducedByCI    ProducedBy = "ci"
)

// InputRole is the role of one input descriptor.
type InputRole string

// Input roles.
const (
	RoleSpec           InputRole = "spec"
	RoleTestSource     InputRole = "test_source"
	RoleCodeTree       InputRole = "code_tree"
	RoleJUnitResult    InputRole = "junit_result"
	RoleMutantSet      InputRole = "mutant_set"
	RoleModelID        InputRole = "model_id"
	RolePromptTemplate InputRole = "prompt_template"
	RoleDecisionPolicy InputRole = "decision_policy"
	RoleExampleBinding InputRole = "example_binding"
	RoleHostResponse   InputRole = "host_response"
	RoleProof          InputRole = "proof"
)

// DecisionBasis tells how a judged claim was decided.
type DecisionBasis string

// Decision bases.
const (
	BasisConstruction DecisionBasis = "construction"
	BasisSpecExample  DecisionBasis = "spec_example"
	BasisJudges       DecisionBasis = "judges"
	BasisSystemOne    DecisionBasis = "system_one"
	BasisExecution    DecisionBasis = "execution"
)

// Channel tells how a response reached the module that consumed it.
type Channel string

// Channels.
const (
	ChannelHostSubagent   Channel = "host-subagent"
	ChannelHostSampled    Channel = "host-sampled"
	ChannelHostInline     Channel = "host-inline"
	ChannelSystemOneRelay Channel = "systemone-relay"
)

// ModelSource tells how the model of an executor was learned.
type ModelSource string

// Model sources.
const (
	ModelSourceResolved ModelSource = "resolved"
	ModelSourceDeclared ModelSource = "declared"
	ModelSourceUnknown  ModelSource = "unknown"
)

// CorroborationKind names the executed evidence a judged claim rests on.
type CorroborationKind string

// Corroboration kinds, one per row of the claim-power table.
const (
	CorroborationExampleFailed      CorroborationKind = "example_failed"
	CorroborationOtherExampleFailed CorroborationKind = "other_example_failed"
	CorroborationSpecificKill       CorroborationKind = "specific_kill"
	CorroborationWitness            CorroborationKind = "f4_witness"
	CorroborationBuilds             CorroborationKind = "builds"
	CorroborationOtherCriteriaPass  CorroborationKind = "other_criteria_pass"
	CorroborationNoDiffFound        CorroborationKind = "no_diff_found"
	CorroborationSpecificMatrix     CorroborationKind = "specific_matrix"
)

var (
	verdicts       = []Verdict{VerdictAligned, VerdictMisaligned, VerdictUndetermined}
	validities     = []Validity{ValidityValid, ValidityEquivalent, ValidityNotViolating, ValidityUndetermined}
	outcomes       = []Outcome{OutcomeKilled, OutcomeSurvived, OutcomeError}
	exampleResults = []ExampleResult{ExamplePassed, ExampleFailed}
	producers      = []ProducedBy{ProducedByLocal, ProducedByCI}
	inputRoles     = []InputRole{
		RoleSpec, RoleTestSource, RoleCodeTree, RoleJUnitResult, RoleMutantSet,
		RoleModelID, RolePromptTemplate, RoleDecisionPolicy, RoleExampleBinding,
		RoleHostResponse, RoleProof,
	}
	bases        = []DecisionBasis{BasisConstruction, BasisSpecExample, BasisJudges, BasisSystemOne, BasisExecution}
	channels     = []Channel{ChannelHostSubagent, ChannelHostSampled, ChannelHostInline, ChannelSystemOneRelay}
	modelSources = []ModelSource{ModelSourceResolved, ModelSourceDeclared, ModelSourceUnknown}

	corroborationKinds = []CorroborationKind{
		CorroborationExampleFailed, CorroborationOtherExampleFailed, CorroborationSpecificKill,
		CorroborationWitness, CorroborationBuilds, CorroborationOtherCriteriaPass,
		CorroborationNoDiffFound, CorroborationSpecificMatrix,
	}
)

// SubjectName builds the `ID~REV` subject name of a requirement.
func SubjectName(requirementID string, revision int) string {
	return requirementID + RevisionSeparator + strconv.Itoa(revision)
}
