package proof_test

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

var statementDigest = prooftest.Digest("PARK-2 statement")

func discriminatingPower() proof.Statement {
	return prooftest.DiscriminatingPower(prooftest.RequirementID, prooftest.RequirementRevision, statementDigest)
}

func alignment() proof.Statement {
	return prooftest.Alignment(prooftest.RequirementID, prooftest.RequirementRevision, statementDigest, "test_one")
}

func completeness() proof.Statement {
	return prooftest.Completeness("specs/parkfee.md", prooftest.Digest("spec file"))
}

func roundTrip(t *testing.T, statement proof.Statement) proof.Statement {
	t.Helper()
	decoded, err := proof.Decode(prooftest.Bytes(statement))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return decoded
}

func codesOf(violations []proof.Violation) []proof.ViolationCode {
	return proof.Codes(violations)
}

func wantCodes(t *testing.T, statement proof.Statement, want ...proof.ViolationCode) {
	t.Helper()
	got := codesOf(roundTrip(t, statement).Validate())
	slices.Sort(want)
	if len(want) == 0 {
		want = []proof.ViolationCode{}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("violations = %v, want %v\n%s", got, want, proof.Describe(roundTrip(t, statement).Validate()))
	}
}

func TestEnvelope(t *testing.T) {
	t.Run("reads the envelope and the predicate of each of the three kinds [verifies SN-101~1]", func(t *testing.T) {
		for name, statement := range map[string]proof.Statement{
			"alignment": alignment(), "discriminating-power": discriminatingPower(), "completeness": completeness(),
		} {
			decoded := roundTrip(t, statement)
			if decoded.Kind != proof.Kind(name) {
				t.Errorf("%s decoded as %s", name, decoded.Kind)
			}
			if violations := decoded.Validate(); len(violations) != 0 {
				t.Errorf("%s: %s", name, proof.Describe(violations))
			}
		}
		decoded := roundTrip(t, discriminatingPower())
		if got := decoded.DiscriminatingPower.Requirement; got.ID != prooftest.RequirementID || got.Revision != prooftest.RequirementRevision {
			t.Errorf("requirement = %+v", got)
		}
		if !decoded.DiscriminatingPower.CorrectCodePasses.Value {
			t.Error("correctCodePasses lost")
		}
	})

	t.Run("ignores a member the format does not define [verifies SN-101~1]", func(t *testing.T) {
		var generic map[string]any
		if err := json.Unmarshal(prooftest.Bytes(alignment()), &generic); err != nil {
			t.Fatal(err)
		}
		generic["futureMember"] = "added by a later minor version"
		generic["predicate"].(map[string]any)["futureMember"] = map[string]any{"nested": true}
		document, err := json.Marshal(generic)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := proof.Decode(document)
		if err != nil {
			t.Fatal(err)
		}
		if violations := decoded.Validate(); len(violations) != 0 {
			t.Fatal(proof.Describe(violations))
		}
	})

	t.Run("ignores a statement of another predicate type [verifies SN-101~1]", func(t *testing.T) {
		document := strings.Replace(string(prooftest.Bytes(alignment())), proof.PredicateTypeAlignment, "https://rachid-chabane.com/attestations/future/v1", 1)
		if _, err := proof.Decode([]byte(document)); !errors.Is(err, proof.ErrUnknownPredicateType) {
			t.Fatalf("err = %v, want ErrUnknownPredicateType", err)
		}
	})

	t.Run("reports a _type other than Statement v1 and a predicate type that differs from the statement's own [verifies SN-101~1]", func(t *testing.T) {
		wrongType := alignment()
		wrongType.Type = "https://in-toto.io/Statement/v0.1"
		wantCodes(t, wrongType, proof.CodeStatementType)

		decoded := roundTrip(t, alignment())
		decoded.PredicateType = proof.PredicateTypeDiscriminatingPower
		if got := codesOf(decoded.Validate()); !slices.Equal(got, []proof.ViolationCode{proof.CodePredicateType}) {
			t.Fatalf("violations = %v", got)
		}
	})

	t.Run("refuses a document that is not JSON or holds a mistyped member [verifies SN-101~1]", func(t *testing.T) {
		for _, document := range []string{"", "not json", `{"predicateType": 3}`,
			strings.Replace(string(prooftest.Bytes(alignment())), `"test":"test_one"`, `"test":7`, 1)} {
			if _, err := proof.Decode([]byte(document)); err == nil {
				t.Errorf("accepted %q", document)
			}
		}
	})
}

func TestClaimRules(t *testing.T) {
	t.Run("reports a judged claim without a confidence [verifies SN-102~1]", func(t *testing.T) {
		statement := alignment()
		statement.Alignment.Verdict.Confidence = nil
		wantCodes(t, prooftest.Seal(statement), proof.CodeJudgedWithoutConfidence)
	})

	t.Run("reports an executed claim with a confidence [verifies SN-102~1]", func(t *testing.T) {
		statement := discriminatingPower()
		confidence := 0.5
		statement.DiscriminatingPower.CorrectCodePasses.Confidence = &confidence
		wantCodes(t, statement, proof.CodeExecutedWithConfidence)
	})

	t.Run("reports a confidence outside [0,1] and accepts the bounds [verifies SN-102~1]", func(t *testing.T) {
		for _, outside := range []float64{-0.01, 1.01} {
			statement := alignment()
			statement.Alignment.Verdict.Confidence = &outside
			wantCodes(t, statement, proof.CodeConfidenceRange)
		}
		for _, inside := range []float64{0, 1} {
			statement := alignment()
			statement.Alignment.Verdict.Confidence = &inside
			wantCodes(t, statement)
		}
	})

	t.Run("reports a claim whose class is not the one its field allows [verifies SN-102~1]", func(t *testing.T) {
		judgedVerdictAsExecuted := alignment()
		judgedVerdictAsExecuted.Alignment.Verdict = prooftest.Executed(proof.VerdictAligned)
		wantCodes(t, judgedVerdictAsExecuted, proof.CodeWrongClass)

		judgedExecution := discriminatingPower()
		judgedExecution.DiscriminatingPower.CorrectCodePasses = prooftest.Judged(true, prooftest.DefaultConfidence)
		wantCodes(t, judgedExecution, proof.CodeWrongClass)

		unknownClass := alignment()
		unknownClass.Alignment.Verdict.Class = "guessed"
		wantCodes(t, unknownClass, proof.CodeUnknownValue)
	})
}

func TestSubjectAndCacheKey(t *testing.T) {
	t.Run("reports a subject that is not the requirement of the predicate [verifies SN-103~1]", func(t *testing.T) {
		statement := alignment()
		statement.Subject[0].Name = "PARK-3~1"
		wantCodes(t, statement, proof.CodeSubject, proof.CodeCacheKeyMismatch)
	})

	t.Run("reports a subject count other than one and a digest that is not SHA-256 [verifies SN-103~1]", func(t *testing.T) {
		twice := discriminatingPower()
		twice.Subject = append(twice.Subject, twice.Subject[0])
		wantCodes(t, twice, proof.CodeSubject, proof.CodeCacheKeyMismatch)

		badDigest := discriminatingPower()
		badDigest.Subject[0].Digest.SHA256 = "XYZ"
		wantCodes(t, badDigest, proof.CodeBadDigest, proof.CodeCacheKeyMismatch)
	})

	t.Run("requires a relative POSIX spec path for a completeness statement [verifies SN-103~1]", func(t *testing.T) {
		for _, path := range []string{"/etc/specs/parkfee.md", "../parkfee.md", `specs\parkfee.md`, ""} {
			statement := prooftest.Completeness(path, prooftest.Digest("file"))
			wantCodes(t, statement, proof.CodeSubject)
		}
		wantCodes(t, completeness())
	})

	t.Run("reports a cache key that is not the digest of the subject and the inputs [verifies SN-103~1]", func(t *testing.T) {
		statement := alignment()
		statement.Alignment.CacheKey = prooftest.Digest("another key")
		wantCodes(t, statement, proof.CodeCacheKeyMismatch)
	})

	t.Run("computes the same cache key for any order of the inputs and another for any other input [verifies SN-103~1]", func(t *testing.T) {
		subject := alignment().Subject
		inputs := []proof.Input{prooftest.Input(proof.RoleSpec, "a"), prooftest.Input(proof.RoleTestSource, "b")}
		reversed := []proof.Input{inputs[1], inputs[0]}
		first, err := proof.CacheKey(subject, inputs)
		if err != nil {
			t.Fatal(err)
		}
		second, err := proof.CacheKey(subject, reversed)
		if err != nil {
			t.Fatal(err)
		}
		changed, err := proof.CacheKey(subject, append(slices.Clone(inputs), prooftest.Input(proof.RoleCodeTree, "c")))
		if err != nil {
			t.Fatal(err)
		}
		if first != second || first == changed {
			t.Fatalf("keys: %s %s %s", first, second, changed)
		}
	})
}

func TestClosedSetsAndOrders(t *testing.T) {
	t.Run("reports a value outside a closed set [verifies SN-104~1]", func(t *testing.T) {
		mutations := map[string]func(*proof.Statement){
			"input role":   func(s *proof.Statement) { s.Alignment.Inputs[0].Role = "weird_role" },
			"verdict":      func(s *proof.Statement) { s.Alignment.Verdict.Value = "MAYBE" },
			"producer":     func(s *proof.Statement) { s.Alignment.ProducedBy = "somewhere" },
			"channel":      func(s *proof.Statement) { s.Alignment.Executor.Channel = "carrier-pigeon" },
			"isolation":    func(s *proof.Statement) { s.Alignment.Executor.Isolation = "moat" },
			"model source": func(s *proof.Statement) { s.Alignment.Executor.Model.Source = "rumour" },
			"decision basis": func(s *proof.Statement) {
				s.Alignment.Extensions.Decision = &proof.Decision{Basis: "hunch", Deciders: []string{"a"}}
			},
			"corroboration":   func(s *proof.Statement) { s.Alignment.Verdict.CorroboratedBy[0].Kind = "vibes" },
			"digest of input": func(s *proof.Statement) { s.Alignment.Inputs[0].Digest.SHA256 = "abc" },
		}
		for name, mutate := range mutations {
			statement := alignment()
			mutate(&statement)
			got := codesOf(roundTrip(t, statement).Validate())
			if len(got) == 0 {
				t.Errorf("%s: accepted", name)
			}
		}

		power := discriminatingPower()
		power.DiscriminatingPower.Mutants[0].Validity.Value = "SORT_OF"
		power.DiscriminatingPower.Mutants[0].Results[0].Outcome.Value = "maybe"
		power.DiscriminatingPower.Examples = []proof.ExampleRun{{Index: 0, Result: prooftest.Executed(proof.ExampleResult("shrug"))}}
		got := codesOf(roundTrip(t, power).Validate())
		if !slices.Contains(got, proof.CodeUnknownValue) {
			t.Errorf("violations = %v", got)
		}
	})

	t.Run("reports an array that is not strictly ascending [verifies SN-104~1]", func(t *testing.T) {
		unsortedTests := discriminatingPower()
		unsortedTests.DiscriminatingPower.Tests = []string{"test_b", "test_a"}
		wantCodes(t, unsortedTests, proof.CodeUnsorted)

		duplicateTests := discriminatingPower()
		duplicateTests.DiscriminatingPower.Tests = []string{"test_a", "test_a"}
		wantCodes(t, duplicateTests, proof.CodeUnsorted)

		unsortedInputs := alignment()
		inputs := unsortedInputs.Alignment.Inputs
		unsortedInputs.Alignment.Inputs = append([]proof.Input{prooftest.Input(proof.RoleTestSource, "t")}, append(inputs, prooftest.Input(proof.RoleHostResponse, "s"))...)
		unsortedInputs.Alignment.CacheKey, _ = proof.CacheKey(unsortedInputs.Subject, unsortedInputs.Alignment.Inputs)
		wantCodes(t, unsortedInputs, proof.CodeUnsorted)

		unsortedExamples := discriminatingPower()
		unsortedExamples.DiscriminatingPower.Examples = []proof.ExampleRun{
			{Index: 1, Result: prooftest.Executed(proof.ExamplePassed)}, {Index: 0, Result: prooftest.Executed(proof.ExamplePassed)},
		}
		wantCodes(t, unsortedExamples, proof.CodeUnsorted)

		unsortedMutants := discriminatingPower()
		first := unsortedMutants.DiscriminatingPower.Mutants[0]
		second := prooftest.Mutant("other", prooftest.Judged(proof.ValidityUndetermined, 0.5), map[string]proof.Outcome{"test_one": proof.OutcomeSurvived})
		pair := []proof.Mutant{first, second}
		slices.SortFunc(pair, func(a, b proof.Mutant) int { return -strings.Compare(a.Digest, b.Digest) })
		unsortedMutants.DiscriminatingPower.Mutants = pair
		wantCodes(t, unsortedMutants, proof.CodeUnsorted)
	})

	t.Run("sorts byte-wise, so that PARK-10 comes before PARK-2 [verifies SN-104~1]", func(t *testing.T) {
		statement := completeness()
		statement.Completeness.Questions = []proof.Question{
			{Dimension: "time", Question: "q", RequirementIDs: []string{"PARK-10", "PARK-2"}},
		}
		wantCodes(t, statement)
		statement.Completeness.Questions[0].RequirementIDs = []string{"PARK-2", "PARK-10"}
		wantCodes(t, statement, proof.CodeUnsorted)
	})

	t.Run("reports a bad digest in a corroboration and an unsorted corroboration list [verifies SN-104~1]", func(t *testing.T) {
		badDigest := discriminatingPower()
		badDigest.DiscriminatingPower.Mutants[0].Validity.CorroboratedBy[0].Digest = "short"
		wantCodes(t, badDigest, proof.CodeBadDigest)

		unsorted := discriminatingPower()
		unsorted.DiscriminatingPower.Mutants[0].Validity = prooftest.Judged(proof.ValidityValid, 0.9, proof.CorroborationExampleFailed, proof.CorroborationBuilds)
		unsorted.DiscriminatingPower.Mutants[0].Validity.CorroboratedBy = []proof.Corroboration{
			{Kind: proof.CorroborationExampleFailed, Digest: prooftest.Digest("a")}, {Kind: proof.CorroborationBuilds, Digest: prooftest.Digest("b")},
		}
		wantCodes(t, unsorted, proof.CodeUnsorted)
	})
}

func TestCounterexampleDecoding(t *testing.T) {
	t.Run("reads the counterexample a module records on a mutant [verifies SN-139~1]", func(t *testing.T) {
		statement := discriminatingPower()
		example := 2
		statement.DiscriminatingPower.Mutants[0].Extensions = &proof.MutantExtensions{Counterexample: &proof.Counterexample{
			Origin: proof.OriginOperator, Family: "relational", Example: &example,
			Witness: &proof.WitnessExample{Name: "w", Clause: "c", Code: "def w(): pass", Truncated: true},
		}}
		decoded := roundTrip(t, statement)
		got := decoded.DiscriminatingPower.Mutants[0].Extensions.Counterexample
		if got == nil || got.Origin != proof.OriginOperator || got.Family != "relational" || got.Example == nil || *got.Example != example ||
			got.Witness == nil || got.Witness.Code != "def w(): pass" || !got.Witness.Truncated {
			t.Errorf("counterexample = %+v", got)
		}
	})
}

func TestStructuralInvariants(t *testing.T) {
	t.Run("reports a judged VALID mutant without executed corroboration [verifies SN-105~1]", func(t *testing.T) {
		statement := discriminatingPower()
		statement.DiscriminatingPower.Mutants[0].Validity = prooftest.Judged(proof.ValidityValid, 0.9)
		wantCodes(t, statement, proof.CodeUncorroboratedClaim)

		for _, validity := range []proof.Validity{proof.ValidityEquivalent, proof.ValidityNotViolating, proof.ValidityUndetermined} {
			statement.DiscriminatingPower.Mutants[0].Validity = prooftest.Judged(validity, 0.9)
			wantCodes(t, statement)
		}
	})

	t.Run("reports an executed claim that carries a corroboration or an executor [verifies SN-105~1]", func(t *testing.T) {
		corroborated := discriminatingPower()
		corroborated.DiscriminatingPower.CorrectCodePasses.CorroboratedBy = prooftest.Judged(true, 1, proof.CorroborationBuilds).CorroboratedBy
		wantCodes(t, corroborated, proof.CodeExecutedWithCorroboration)

		withExecutor := discriminatingPower()
		executor := prooftest.Executor(proof.IsolationRestricted)
		withExecutor.DiscriminatingPower.Mutants[0].Results[0].Outcome.Executor = &executor
		wantCodes(t, withExecutor, proof.CodeExecutedWithExecutor)
	})

	t.Run("requires of an executor the fields its isolation level needs [verifies SN-105~1]", func(t *testing.T) {
		attestedWithout := alignment()
		attestedWithout.Alignment.Executor = prooftest.Executor(proof.IsolationRestrictedAttested)
		attestedWithout.Alignment.Executor.Attestation = proof.Attestation{}
		wantCodes(t, attestedWithout, proof.CodeMissingField, proof.CodeBadDigest)

		attested := alignment()
		attested.Alignment.Executor = prooftest.Executor(proof.IsolationRestrictedAttested)
		wantCodes(t, attested)

		unnamedModel := alignment()
		unnamedModel.Alignment.Executor.Model = proof.ExecutorModel{Source: proof.ModelSourceDeclared}
		wantCodes(t, unnamedModel, proof.CodeMissingField)

		unknownModel := alignment()
		unknownModel.Alignment.Executor.Model = proof.ExecutorModel{Source: proof.ModelSourceUnknown}
		wantCodes(t, unknownModel)

		badPrompt := alignment()
		badPrompt.Alignment.Executor.PromptDigest = "x"
		wantCodes(t, badPrompt, proof.CodeBadDigest)
	})

	t.Run("reports a sentinel that is not EQUIVALENT [verifies SN-105~1]", func(t *testing.T) {
		statement := discriminatingPower()
		statement.DiscriminatingPower.Mutants[0].Sentinel = true
		wantCodes(t, statement, proof.CodeSentinelNotEquivalent)
		statement.DiscriminatingPower.Mutants[0].Validity = prooftest.Judged(proof.ValidityEquivalent, 1)
		wantCodes(t, statement)
	})

	t.Run("reports a judge that was not blind and a completeness proof that is not advisory [verifies SN-105~1]", func(t *testing.T) {
		notBlind := alignment()
		notBlind.Alignment.Executor.Blind = false
		wantCodes(t, notBlind, proof.CodeNotBlind)

		notAdvisory := completeness()
		notAdvisory.Completeness.Advisory = false
		wantCodes(t, notAdvisory, proof.CodeNotAdvisory)
	})

	t.Run("reports a mutant that is both unattributed and off-target, or attributed as something other than NOT_VIOLATING [verifies SN-105~1]", func(t *testing.T) {
		both := discriminatingPower()
		both.DiscriminatingPower.Mutants[0].Validity = prooftest.Judged(proof.ValidityNotViolating, 0.9)
		both.DiscriminatingPower.Mutants[0].Extensions = &proof.MutantExtensions{Unattributed: true, OffTarget: "PARK-3"}
		wantCodes(t, both, proof.CodeInconsistentAttribution)

		valid := discriminatingPower()
		valid.DiscriminatingPower.Mutants[0].Extensions = &proof.MutantExtensions{OffTarget: "PARK-3"}
		wantCodes(t, valid, proof.CodeInconsistentAttribution)
	})
}

func FuzzDecode(f *testing.F) {
	for _, statement := range []proof.Statement{alignment(), discriminatingPower(), completeness()} {
		f.Add(prooftest.Bytes(statement))
	}
	f.Add([]byte(`{"predicateType":"https://rachid-chabane.com/attestations/alignment/v1","predicate":null}`))
	f.Fuzz(func(t *testing.T, document []byte) {
		decoded, err := proof.Decode(document)
		if err != nil {
			return
		}
		decoded.Validate()
	})
}

func FuzzParseManifest(f *testing.F) {
	f.Add([]byte(`{"verifierVersion":"1","trust":"local","statements":[],"pending":[]}`))
	f.Add([]byte(`{"statements":[{"path":"../x","digest":"y"}],"pending":[{"criterion":"A~0"}]}`))
	f.Fuzz(func(t *testing.T, document []byte) {
		if _, violations, err := proof.ParseManifest(document); err == nil {
			proof.Describe(violations)
		}
	})
}
