package proof

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrUnknownPredicateType is returned by Decode for a predicate type outside the
// format. Readers ignore such statements.
var ErrUnknownPredicateType = errors.New("unknown predicate type")

// Digest is the digest set of a subject or an input. Only SHA-256 is part of the
// format; other algorithms present on the wire are ignored.
type Digest struct {
	SHA256 string `json:"sha256"`
}

// Subject is one in-toto subject.
type Subject struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}

// Requirement identifies the criterion a statement is about.
type Requirement struct {
	ID                        string `json:"id"`
	Revision                  int    `json:"revision"`
	StatementCanonicalization string `json:"statementCanonicalization"`
}

// Module identifies the module that produced a statement.
type Module struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Annotations carries the file globs a file-set input was expanded from.
type Annotations struct {
	Globs []string `json:"globs,omitempty"`
}

// Input is one resource descriptor that the statement depends on.
type Input struct {
	Role        InputRole    `json:"role"`
	Name        string       `json:"name"`
	Digest      Digest       `json:"digest"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

// Decision records how a judged claim was decided.
type Decision struct {
	Basis                DecisionBasis `json:"basis"`
	Deciders             []string      `json:"deciders"`
	RawConfidence        float64       `json:"rawConfidence"`
	CalibratedConfidence *float64      `json:"calibratedConfidence,omitempty"`
	Threshold            float64       `json:"threshold"`
	EscalationReason     string        `json:"escalationReason,omitempty"`
}

// OperatorScreen counts operator mutants that were screened rather than listed.
type OperatorScreen struct {
	Flagged     int `json:"flagged"`
	ScreenedOut int `json:"screenedOut"`
}

// Extensions is the open part of a predicate. Readers ignore unknown members.
type Extensions struct {
	Decision       *Decision       `json:"decision,omitempty"`
	OperatorScreen *OperatorScreen `json:"operatorScreen,omitempty"`
}

// Core is the part shared by the three predicates.
type Core struct {
	Module     Module     `json:"module"`
	Inputs     []Input    `json:"inputs"`
	CacheKey   string     `json:"cacheKey"`
	ProducedBy ProducedBy `json:"producedBy"`
	Extensions Extensions `json:"extensions"`
}

// Harness identifies the host harness that ran an executor.
type Harness struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Attestation is the provenance evidence of an executor.
type Attestation struct {
	AgentID          string `json:"agentId"`
	CapDigest        string `json:"capDigest"`
	TranscriptDigest string `json:"transcriptDigest"`
}

// ExecutorModel names the model of an executor and how it was learned.
type ExecutorModel struct {
	Name   string      `json:"name"`
	Source ModelSource `json:"source"`
}

// ExecutorProfile describes who answered a request.
type ExecutorProfile struct {
	Channel     Channel       `json:"channel"`
	Harness     Harness       `json:"harness"`
	Isolation   Isolation     `json:"isolation"`
	Attestation Attestation   `json:"attestation"`
	Model       ExecutorModel `json:"model"`
}

// Executor is the profile of the executor that produced a judged or generated
// result, with the digest of the prompt it answered and whether it was blind.
type Executor struct {
	ExecutorProfile
	PromptDigest string `json:"promptDigest"`
	Blind        bool   `json:"blind"`
}

// Corroboration is one piece of executed evidence behind a judged claim.
type Corroboration struct {
	Kind   CorroborationKind `json:"kind"`
	Digest string            `json:"digest"`
}

// Claim is a verdict-bearing value with the class of its evidence.
type Claim[V comparable] struct {
	Value          V               `json:"value"`
	Class          Class           `json:"class"`
	Confidence     *float64        `json:"confidence,omitempty"`
	Executor       *Executor       `json:"executor,omitempty"`
	CorroboratedBy []Corroboration `json:"corroboratedBy,omitempty"`
}

// ConfidenceOrZero returns the confidence of the claim, 0 when it carries none.
func (c Claim[V]) ConfidenceOrZero() float64 {
	if c.Confidence == nil {
		return 0
	}
	return *c.Confidence
}

// Corroborates reports whether the claim lists executed evidence of that kind.
func (c Claim[V]) Corroborates(kind CorroborationKind) bool {
	for _, evidence := range c.CorroboratedBy {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

// Alignment is the predicate of one (test, criterion) alignment judgment.
type Alignment struct {
	Requirement Requirement `json:"requirement"`
	Core
	Test     string         `json:"test"`
	Verdict  Claim[Verdict] `json:"verdict"`
	Executor Executor       `json:"executor"`
}

// MutantExtensions is the open part of one mutant entry.
type MutantExtensions struct {
	Decision     *Decision `json:"decision,omitempty"`
	Collateral   bool      `json:"collateral,omitempty"`
	OffTarget    string    `json:"offTarget,omitempty"`
	Unattributed bool      `json:"unattributed,omitempty"`
}

// TestResult is the executed outcome of one test against one mutant.
type TestResult struct {
	Test    string         `json:"test"`
	Outcome Claim[Outcome] `json:"outcome"`
}

// Mutant is one entry of the mutant pool. The mutant body is never part of the proof.
type Mutant struct {
	Digest     string            `json:"digest"`
	Validity   Claim[Validity]   `json:"validity"`
	Sentinel   bool              `json:"sentinel,omitempty"`
	Results    []TestResult      `json:"results"`
	Extensions *MutantExtensions `json:"extensions,omitempty"`
}

// ExampleRun is the executed result of one spec example on the reference code.
type ExampleRun struct {
	Index  int                  `json:"index"`
	Result Claim[ExampleResult] `json:"result"`
}

// DiscriminatingPower is the predicate of one criterion's tests-by-mutants matrix.
type DiscriminatingPower struct {
	Requirement Requirement `json:"requirement"`
	Core
	Tests             []string     `json:"tests"`
	CorrectCodePasses Claim[bool]  `json:"correctCodePasses"`
	Mutants           []Mutant     `json:"mutants,omitempty"`
	Executor          Executor     `json:"executor"`
	Examples          []ExampleRun `json:"examples,omitempty"`
}

// Question is one advisory completeness question.
type Question struct {
	Dimension      string   `json:"dimension"`
	Question       string   `json:"question"`
	RequirementIDs []string `json:"requirementIds,omitempty"`
}

// OutOfScope is one dimension the spec owner declared out of scope.
type OutOfScope struct {
	Dimension string `json:"dimension"`
	Reason    string `json:"reason"`
}

// Completeness is the advisory predicate of one spec. It has no verdict field, so the gate cannot block on it.
type Completeness struct {
	Core
	Advisory           bool         `json:"advisory"`
	Questions          []Question   `json:"questions,omitempty"`
	OutOfScopeDeclared []OutOfScope `json:"outOfScopeDeclared,omitempty"`
}

// Statement is an in-toto Statement v1 carrying the predicate of its kind. Exactly one of the predicate pointers is set.
type Statement struct {
	Type          string
	Subject       []Subject
	PredicateType string
	Kind          Kind

	Alignment           *Alignment
	DiscriminatingPower *DiscriminatingPower
	Completeness        *Completeness
}

type envelope struct {
	Type          string          `json:"_type"`
	Subject       []Subject       `json:"subject"`
	PredicateType string          `json:"predicateType"`
	Predicate     json.RawMessage `json:"predicate"`
}

// Decode reads a statement of any known predicate type. Unknown members are
// ignored, because minor versions are additive. It does not validate: call
// Validate on the result. A predicate type outside the format yields
// ErrUnknownPredicateType.
func Decode(data []byte) (Statement, error) {
	var wire envelope
	if err := json.Unmarshal(data, &wire); err != nil {
		return Statement{}, fmt.Errorf("decode statement: %w", err)
	}
	kind, known := KindOfPredicateType(wire.PredicateType)
	if !known {
		return Statement{}, fmt.Errorf("%w: %q", ErrUnknownPredicateType, wire.PredicateType)
	}
	statement := Statement{Type: wire.Type, Subject: wire.Subject, PredicateType: wire.PredicateType, Kind: kind}
	var err error
	switch kind {
	case KindAlignment:
		statement.Alignment, err = decodePredicate[Alignment](wire.Predicate)
	case KindDiscriminatingPower:
		statement.DiscriminatingPower, err = decodePredicate[DiscriminatingPower](wire.Predicate)
	default:
		statement.Completeness, err = decodePredicate[Completeness](wire.Predicate)
	}
	return statement, err
}

func decodePredicate[P any](raw json.RawMessage) (*P, error) {
	var predicate P
	if len(raw) == 0 {
		return nil, errors.New("decode statement: no predicate")
	}
	if err := json.Unmarshal(raw, &predicate); err != nil {
		return nil, fmt.Errorf("decode statement: %w", err)
	}
	return &predicate, nil
}

// Core returns the part shared by the statement's predicate.
func (s Statement) Core() Core {
	switch s.Kind {
	case KindAlignment:
		return s.Alignment.Core
	case KindDiscriminatingPower:
		return s.DiscriminatingPower.Core
	default:
		return s.Completeness.Core
	}
}

// Requirement returns the criterion the statement is about; completeness has none.
func (s Statement) Requirement() (Requirement, bool) {
	switch s.Kind {
	case KindAlignment:
		return s.Alignment.Requirement, true
	case KindDiscriminatingPower:
		return s.DiscriminatingPower.Requirement, true
	default:
		return Requirement{}, false
	}
}

// MarshalJSON writes the statement in its wire form.
func (s Statement) MarshalJSON() ([]byte, error) {
	var predicate any
	switch s.Kind {
	case KindAlignment:
		predicate = s.Alignment
	case KindDiscriminatingPower:
		predicate = s.DiscriminatingPower
	case KindCompleteness:
		predicate = s.Completeness
	default:
		return nil, fmt.Errorf("encode statement: unknown kind %q", s.Kind)
	}
	return json.Marshal(struct {
		Type          string    `json:"_type"`
		Subject       []Subject `json:"subject"`
		PredicateType string    `json:"predicateType"`
		Predicate     any       `json:"predicate"`
	}{s.Type, s.Subject, s.PredicateType, predicate})
}
