// Package prooftest builds well-formed proofs for tests. It is test support:
// no shipped package imports it.
package prooftest

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// Defaults of the builders.
const (
	RequirementID       = "PARK-2"
	RequirementRevision = 1
	DefaultConfidence   = 0.9
	moduleVersion       = "0.1.0"
	harnessName         = "claude-code"
	harnessVersion      = "2.1.292"
	modelName           = "claude-opus-5-5"
)

// Digest returns the SHA-256 of a seed, a stand-in for any digest.
func Digest(seed string) string { return proof.SHA256Hex([]byte(seed)) }

// Executor returns a valid, blind executor of the given isolation level.
func Executor(isolation proof.Isolation) proof.Executor {
	executor := proof.Executor{
		ExecutorProfile: proof.ExecutorProfile{
			Channel:   proof.ChannelHostSubagent,
			Harness:   proof.Harness{Name: harnessName, Version: harnessVersion},
			Isolation: isolation,
			Model:     proof.ExecutorModel{Name: modelName, Source: proof.ModelSourceResolved},
		},
		PromptDigest: Digest("prompt"),
		Blind:        true,
	}
	if isolation == proof.IsolationRestrictedAttested {
		executor.Attestation = proof.Attestation{AgentID: "agent-1", CapDigest: Digest("cap"), TranscriptDigest: Digest("transcript")}
	}
	return executor
}

// Executed builds an executed claim.
func Executed[V comparable](value V) proof.Claim[V] {
	return proof.Claim[V]{Value: value, Class: proof.ClassExecuted}
}

// Judged builds a judged claim corroborated by the given kinds of executed evidence.
func Judged[V comparable](value V, confidence float64, kinds ...proof.CorroborationKind) proof.Claim[V] {
	claim := proof.Claim[V]{Value: value, Class: proof.ClassJudged, Confidence: &confidence}
	for _, kind := range kinds {
		claim.CorroboratedBy = append(claim.CorroboratedBy, proof.Corroboration{Kind: kind, Digest: Digest(string(kind))})
	}
	slices.SortFunc(claim.CorroboratedBy, func(a, b proof.Corroboration) int { return strings.Compare(string(a.Kind), string(b.Kind)) })
	return claim
}

// Input builds an input descriptor whose digest derives from its name.
func Input(role proof.InputRole, name string) proof.Input {
	return proof.Input{Role: role, Name: name, Digest: proof.Digest{SHA256: Digest(name)}}
}

// ProofInput builds a `proof` input: named by the consumed kind, carrying the criterion it is about, with the
// digest of the consumed Statement file.
func ProofInput(kind proof.Kind, criterion, statementFileDigest string) proof.Input {
	return proof.Input{
		Role: proof.RoleProof, Name: string(kind), Digest: proof.Digest{SHA256: statementFileDigest},
		Annotations: &proof.Annotations{Criterion: criterion},
	}
}

// Results builds one result per test, in test name order, with the given outcome per test.
func Results(outcomes map[string]proof.Outcome) []proof.TestResult {
	results := make([]proof.TestResult, 0, len(outcomes))
	for test, outcome := range outcomes {
		results = append(results, proof.TestResult{Test: test, Outcome: Executed(outcome)})
	}
	slices.SortFunc(results, func(a, b proof.TestResult) int { return strings.Compare(a.Test, b.Test) })
	return results
}

// Mutant builds a mutant named by a seed.
func Mutant(seed string, validity proof.Claim[proof.Validity], outcomes map[string]proof.Outcome) proof.Mutant {
	return proof.Mutant{Digest: Digest("mutant " + seed), Validity: validity, Results: Results(outcomes)}
}

func core(inputs ...proof.Input) proof.Core {
	if len(inputs) == 0 {
		inputs = []proof.Input{Input(proof.RoleModelID, modelName)}
	}
	return proof.Core{Module: proof.Module{Name: "spec-proof-module", Version: moduleVersion}, Inputs: inputs, ProducedBy: proof.ProducedByCI}
}

func requirementSubject(id string, revision int, statementDigest string) []proof.Subject {
	return []proof.Subject{{Name: proof.SubjectName(id, revision), Digest: proof.Digest{SHA256: statementDigest}}}
}

func requirement(id string, revision int) proof.Requirement {
	return proof.Requirement{ID: id, Revision: revision, StatementCanonicalization: proof.StatementCanonicalizationC1}
}

func envelope(kind proof.Kind, subject []proof.Subject) proof.Statement {
	return proof.Statement{Type: proof.StatementType, Subject: subject, PredicateType: kind.PredicateType(), Kind: kind}
}

// DiscriminatingPower builds a sealed discriminating-power statement of one criterion: it passes on the
// reference code and carries one test that kills one VALID mutant by an example.
func DiscriminatingPower(id string, revision int, statementDigest string, inputs ...proof.Input) proof.Statement {
	statement := envelope(proof.KindDiscriminatingPower, requirementSubject(id, revision, statementDigest))
	statement.DiscriminatingPower = &proof.DiscriminatingPower{
		Requirement:       requirement(id, revision),
		Core:              core(inputs...),
		Tests:             []string{"test_one"},
		CorrectCodePasses: Executed(true),
		Executor:          Executor(proof.IsolationRestricted),
		Mutants: []proof.Mutant{Mutant("killed", Judged(proof.ValidityValid, DefaultConfidence, proof.CorroborationExampleFailed),
			map[string]proof.Outcome{"test_one": proof.OutcomeKilled})},
	}
	return Seal(statement)
}

// Alignment builds a sealed alignment statement of one test of one criterion, judged ALIGNED.
func Alignment(id string, revision int, statementDigest, test string, inputs ...proof.Input) proof.Statement {
	statement := envelope(proof.KindAlignment, requirementSubject(id, revision, statementDigest))
	statement.Alignment = &proof.Alignment{
		Requirement: requirement(id, revision),
		Core:        core(inputs...),
		Test:        test,
		Verdict:     Judged(proof.VerdictAligned, DefaultConfidence, proof.CorroborationSpecificMatrix),
		Executor:    Executor(proof.IsolationRestricted),
	}
	return Seal(statement)
}

// Completeness builds a sealed advisory completeness statement of one spec file.
func Completeness(specPath, fileDigest string, inputs ...proof.Input) proof.Statement {
	statement := envelope(proof.KindCompleteness, []proof.Subject{{Name: specPath, Digest: proof.Digest{SHA256: fileDigest}}})
	statement.Completeness = &proof.Completeness{Core: core(inputs...), Advisory: true}
	return Seal(statement)
}

// Seal sorts every array of the statement into its canonical order and recomputes the cache key.
func Seal(statement proof.Statement) proof.Statement {
	statementCore := coreOf(&statement)
	slices.SortFunc(statementCore.Inputs, func(a, b proof.Input) int {
		return cmp.Or(strings.Compare(string(a.Role), string(b.Role)), strings.Compare(a.Name, b.Name), strings.Compare(criterionOf(a), criterionOf(b)))
	})
	if predicate := statement.DiscriminatingPower; predicate != nil {
		slices.Sort(predicate.Tests)
		slices.SortFunc(predicate.Mutants, func(a, b proof.Mutant) int { return strings.Compare(a.Digest, b.Digest) })
		slices.SortFunc(predicate.Examples, func(a, b proof.ExampleRun) int { return cmp.Compare(a.Index, b.Index) })
		for index := range predicate.Mutants {
			slices.SortFunc(predicate.Mutants[index].Results, func(a, b proof.TestResult) int { return strings.Compare(a.Test, b.Test) })
		}
	}
	key, err := proof.CacheKey(statement.Subject, statementCore.Inputs)
	if err != nil {
		panic(err)
	}
	statementCore.CacheKey = key
	return statement
}

func coreOf(statement *proof.Statement) *proof.Core {
	switch statement.Kind {
	case proof.KindAlignment:
		return &statement.Alignment.Core
	case proof.KindDiscriminatingPower:
		return &statement.DiscriminatingPower.Core
	default:
		return &statement.Completeness.Core
	}
}

// Bytes returns the wire form of the statement.
func Bytes(statement proof.Statement) []byte {
	encoded, err := json.Marshal(statement)
	if err != nil {
		panic(err)
	}
	return encoded
}

func criterionOf(input proof.Input) string {
	if input.Annotations == nil {
		return ""
	}
	return input.Annotations.Criterion
}
