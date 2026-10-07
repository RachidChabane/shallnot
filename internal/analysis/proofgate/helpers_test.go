package proofgate_test

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	"github.com/RachidChabane/shallnot/internal/analysis/proofgate"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const (
	requirementID = "PARK-2"
	revision      = 1
	testName      = "test_one"
	specPath      = "specs/parkfee.md"
)

var (
	statementDigest = prooftest.Digest("PARK-2 statement")
	specSetDigest   = prooftest.Digest("spec set")
	testSetDigest   = prooftest.Digest("test set")
	codeSetDigest   = prooftest.Digest("code set")
	specFileDigest  = prooftest.Digest("spec file")
)

func derived() proof.DerivedInputs {
	return proof.DerivedInputs{
		Spec:       &proof.FileSet{Globs: []string{"specs"}, Digest: specSetDigest},
		TestSource: &proof.FileSet{Globs: []string{"tests"}, Digest: testSetDigest},
		CodeTree:   &proof.FileSet{Globs: []string{"src"}, Digest: codeSetDigest},
		SpecFiles:  map[string]string{specPath: specFileDigest},
	}
}

// currentInputs are the inputs of a proof produced from the current tree.
func currentInputs(extra ...proof.Input) []proof.Input {
	inputs := []proof.Input{
		{Role: proof.RoleSpec, Name: specPath, Digest: proof.Digest{SHA256: specSetDigest}},
		{Role: proof.RoleTestSource, Name: "tests", Digest: proof.Digest{SHA256: testSetDigest}},
		{Role: proof.RoleCodeTree, Name: "src", Digest: proof.Digest{SHA256: codeSetDigest}},
	}
	return append(inputs, extra...)
}

func criterion(tests ...string) proofgate.Criterion {
	if len(tests) == 0 {
		tests = []string{testName}
	}
	names := make([]proofgate.TestName, len(tests))
	for index, name := range tests {
		names[index] = proofgate.TestName{Name: name}
	}
	return proofgate.Criterion{
		Ref: domain.Ref{ID: requirementID, Revision: revision}, Digest: statementDigest,
		Location: domain.Location{File: specPath, Line: 3}, PassingTests: names,
	}
}

func known() map[domain.RequirementID]proofgate.KnownRequirement {
	return map[domain.RequirementID]proofgate.KnownRequirement{requirementID: {Revision: revision, Digest: statementDigest}}
}

func config(mutate ...func(*proof.Config)) proof.Config {
	config := proof.DefaultConfig()
	config.Require = []proof.Kind{proof.KindDiscriminatingPower}
	config.Code = []string{"src"}
	for _, change := range mutate {
		change(&config)
	}
	return config
}

func local(config *proof.Config) { config.Trust = proof.TrustLocal }
func basis(value proof.SignedPassBasis) func(*proof.Config) {
	return func(config *proof.Config) { config.SignedPassBasis = value }
}
func requireBoth(config *proof.Config) {
	config.Require = []proof.Kind{proof.KindAlignment, proof.KindDiscriminatingPower}
}

// dp builds a discriminating-power statement over the current tree whose pool is the given mutants, or, with none,
// a pool of one VALID mutant that an example of the criterion shows and a test killed.
func dp(mutants ...proof.Mutant) proof.Statement {
	statement := prooftest.DiscriminatingPower(requirementID, revision, statementDigest, currentInputs()...)
	if len(mutants) > 0 {
		statement.DiscriminatingPower.Mutants = mutants
	}
	return prooftest.Seal(statement)
}

func aligned(test string, claim proof.Claim[proof.Verdict], extra ...proof.Input) proof.Statement {
	statement := prooftest.Alignment(requirementID, revision, statementDigest, test, currentInputs(extra...)...)
	statement.Alignment.Verdict = claim
	return prooftest.Seal(statement)
}

func killed(seed string, validity proof.Claim[proof.Validity]) proof.Mutant {
	return prooftest.Mutant(seed, validity, map[string]proof.Outcome{testName: proof.OutcomeKilled})
}

func survivor(seed string, validity proof.Claim[proof.Validity]) proof.Mutant {
	return prooftest.Mutant(seed, validity, map[string]proof.Outcome{testName: proof.OutcomeSurvived})
}

func valid(confidence float64, kinds ...proof.CorroborationKind) proof.Claim[proof.Validity] {
	return prooftest.Judged(proof.ValidityValid, confidence, kinds...)
}

func judged(value proof.Validity, confidence float64, kinds ...proof.CorroborationKind) proof.Claim[proof.Validity] {
	return prooftest.Judged(value, confidence, kinds...)
}

// loaded wraps statements as the adapter would hand them over.
func loaded(statements ...proof.Statement) []proof.LoadedStatement {
	files := make([]proof.LoadedStatement, len(statements))
	for index, statement := range statements {
		statement := statement
		files[index] = proof.LoadedStatement{
			Path: fmt.Sprintf("verified/%02d.json", index), Digest: statementFileDigest(statement), Statement: &statement,
		}
	}
	return files
}

func statementFileDigest(statement proof.Statement) string {
	return proof.SHA256Hex(prooftest.Bytes(statement))
}

func evidence(statements ...proof.Statement) proof.Evidence {
	return proof.Evidence{
		ManifestPath: "verified/manifest.json", ManifestPresent: true, ManifestDigest: prooftest.Digest("manifest"),
		Manifest:   proof.Manifest{VerifierVersion: "0.1.0", Trust: proof.TrustSigned, Statements: []proof.ManifestEntry{}, Pending: []proof.Pending{}},
		Statements: loaded(statements...), Derived: derived(),
	}
}

func evaluate(config proof.Config, evidence proof.Evidence, tests ...string) proofgate.Result {
	return proofgate.Evaluate(proofgate.Input{Config: config, Known: known(), Criteria: []proofgate.Criterion{criterion(tests...)}, Evidence: evidence})
}

func categories(result proofgate.Result) []string {
	names := []string{}
	for _, finding := range result.Findings {
		names = append(names, finding.Category.String())
	}
	sort.Strings(names)
	return names
}

func wantCategories(t *testing.T, result proofgate.Result, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	sort.Strings(want)
	if got := categories(result); !slices.Equal(got, want) {
		t.Fatalf("categories = %v, want %v\n%v", got, want, messages(result))
	}
}

func messages(result proofgate.Result) []string {
	lines := make([]string, len(result.Findings))
	for index, finding := range result.Findings {
		lines[index] = fmt.Sprintf("%s: %s", finding.Category, finding.Message)
	}
	return lines
}

func outcome(t *testing.T, result proofgate.Result) proofgate.Outcome {
	t.Helper()
	return result.Outcomes[domain.Ref{ID: requirementID, Revision: revision}]
}
