package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

const (
	exampleSpec      = "- **REQ-1~1**: THE SYSTEM SHALL work.\n  - Example: work(n=1) => 2\n"
	exampleStatement = "THE SYSTEM SHALL work. - Example: work(n=1) => 2"
	survivorTest     = "test_one"
)

// writeSurvivorProof writes a verified set whose discriminating-power proof of
// REQ-1~1 holds a valid mutant every test passes on, with a counterexample that
// names the criterion's first example.
func writeSurvivorProof(t *testing.T, directory string) {
	t.Helper()
	statementDigest := domain.Requirement{Statement: exampleStatement}.StatementDigest()
	inputs := []proof.Input{
		{Role: proof.RoleCodeTree, Name: proofCode, Digest: proof.Digest{SHA256: fileDigestOf(t, directory, proofCode)}},
		{Role: proof.RoleSpec, Name: "REQ-1~1", Digest: proof.Digest{SHA256: statementDigest}},
		{Role: proof.RoleTestSource, Name: proofTests, Digest: proof.Digest{SHA256: fileDigestOf(t, directory, proofTests)}},
	}
	statement := prooftest.DiscriminatingPower("REQ-1", 1, statementDigest, inputs...)
	statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
	example := 1
	survivor := prooftest.Mutant("survivor", prooftest.Judged(proof.ValidityValid, prooftest.DefaultConfidence, proof.CorroborationExampleFailed),
		map[string]proof.Outcome{survivorTest: proof.OutcomeSurvived})
	survivor.Extensions = &proof.MutantExtensions{Counterexample: &proof.Counterexample{Origin: proof.OriginTargeted, Slot: "S2", Example: &example}}
	statement.DiscriminatingPower.Mutants = append(statement.DiscriminatingPower.Mutants, survivor)
	statement = prooftest.Seal(statement)
	content := prooftest.Bytes(statement)
	manifest, err := json.Marshal(proof.Manifest{
		VerifierVersion: "0.1.0", Trust: proof.TrustLocal, Pending: []proof.Pending{},
		Statements: []proof.ManifestEntry{{Path: "discriminating-power/REQ-1~1/key.json", Digest: proof.SHA256Hex(content)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, ".spec-proof", "verified")
	if err := os.MkdirAll(filepath.Join(root, "discriminating-power", "REQ-1~1"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"manifest.json": manifest, "discriminating-power/REQ-1~1/key.json": content} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHookOnASurvivor(t *testing.T) {
	t.Run("the end-of-turn block names the survivor's slot and the example of the criterion it fails, as the spec writes it [verifies SN-139~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		directory := proofProject(t, proofConfig)
		if err := os.WriteFile(filepath.Join(directory, "spec.md"), []byte(exampleSpec), 0o644); err != nil {
			t.Fatal(err)
		}
		writeSurvivorProof(t, directory)
		t.Setenv("CLAUDE_PROJECT_DIR", directory)
		result := shallnotWithInput(t, "", "hook", "claude-stop")
		if result.exit != claudeExitBlock {
			t.Fatalf("got %+v", result)
		}
		want := "surviving_valid_mutant at spec.md:1: requirement REQ-1~1: mutant "
		tail := "; every test passes on a targeted mutant of slot S2 (boundary) that fails the criterion's example 1: work(n=1) => 2"
		if !strings.Contains(result.stderr, want) || !strings.Contains(result.stderr, tail) {
			t.Fatalf("the block does not name the counterexample:\n%s", result.stderr)
		}
	})
}
