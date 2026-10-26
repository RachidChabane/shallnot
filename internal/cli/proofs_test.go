package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/agenthook"
	"github.com/RachidChabane/shallnot/internal/app"
	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
	"github.com/RachidChabane/shallnot/internal/testkit"
)

const (
	proofSpec   = "- **REQ-1~1**: THE SYSTEM SHALL work.\n"
	proofTests  = "tests/t.py"
	proofCode   = "src/a.py"
	proofConfig = "version: 1\nspecs: [spec.md]\ntests: [tests]\nresults: [junit.xml]\nproofs:\n  trust: local\n  code: [src]\n  require: [discriminating-power]\n"
)

// proofProject is a gated project with a spec, tests, code and results, and no proof yet.
func proofProject(t *testing.T, config string) string {
	t.Helper()
	directory := t.TempDir()
	for name, content := range map[string]string{
		"spec.md":       proofSpec,
		proofTests:      "def works(record_property):\n    record_property(\"" + domain.TagKeyword + "\", \"REQ-1~1\")\n",
		proofCode:       "def work(): pass\n",
		"junit.xml":     `<testsuite name="s"><testcase classname="c" name="works ` + testkit.Tag("REQ-1~1") + `"/></testsuite>`,
		"shallnot.yaml": config,
	} {
		path := filepath.Join(directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func fileDigestOf(t *testing.T, directory, path string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return proof.SHA256Hex(content)
}

// writeProofs writes a verified set holding one discriminating-power proof of REQ-1~1 produced from the current tree.
func writeProofs(t *testing.T, directory string) {
	t.Helper()
	statementDigest := domain.Requirement{Statement: strings.TrimSpace("THE SYSTEM SHALL work.")}.StatementDigest()
	inputs := []proof.Input{
		{Role: proof.RoleCodeTree, Name: proofCode, Digest: proof.Digest{SHA256: fileDigestOf(t, directory, proofCode)}},
		{Role: proof.RoleSpec, Name: "REQ-1~1", Digest: proof.Digest{SHA256: statementDigest}},
		{Role: proof.RoleTestSource, Name: proofTests, Digest: proof.Digest{SHA256: fileDigestOf(t, directory, proofTests)}},
	}
	statement := prooftest.DiscriminatingPower("REQ-1", 1, statementDigest, inputs...)
	statement.DiscriminatingPower.ProducedBy = proof.ProducedByLocal
	statement = prooftest.Seal(statement)
	content := prooftest.Bytes(statement)
	manifest, err := json.Marshal(proof.Manifest{
		VerifierVersion: "0.1.0", Trust: proof.TrustLocal, Pending: []proof.Pending{},
		Statements: []proof.ManifestEntry{{Path: "discriminating-power/REQ-1~1/key.json", Digest: proof.SHA256Hex(content)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, ".musthold", "verified")
	if err := os.MkdirAll(filepath.Join(root, "discriminating-power", "REQ-1~1"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"manifest.json": manifest, "discriminating-power/REQ-1~1/key.json": content} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func checkProject(t *testing.T, directory string, extra ...string) run {
	t.Helper()
	return shallnot(t, append([]string{"check", "--config", filepath.Join(directory, "shallnot.yaml"), "--format", "json"}, extra...)...)
}

func TestProofsEndToEnd(t *testing.T) {
	t.Run("passes a covered requirement whose required proof is current [verifies SN-113~1]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		result := checkProject(t, directory)
		document := decode(t, result.stdout)
		if result.exit != app.ExitClean || document.Verdict != domain.VerdictPass {
			t.Fatalf("exit %d, findings %+v, stderr %s", result.exit, document.Findings, result.stderr)
		}
	})

	t.Run("reads the files the proof names from the directory of the configuration whatever the working directory is [verifies SN-110~3]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		t.Chdir(t.TempDir())
		result := checkProject(t, directory)
		if result.exit != app.ExitClean || decode(t, result.stdout).Verdict != domain.VerdictPass {
			t.Fatalf("exit %d, stdout %s, stderr %s", result.exit, result.stdout, result.stderr)
		}
	})

	t.Run("reads them from the directory of a configuration given as a relative path [verifies SN-110~3]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		t.Chdir(filepath.Dir(directory))
		result := shallnot(t, "check", "--config", filepath.Join(filepath.Base(directory), "shallnot.yaml"), "--format", "json")
		if result.exit != app.ExitClean || decode(t, result.stdout).Verdict != domain.VerdictPass {
			t.Fatalf("exit %d, stdout %s, stderr %s", result.exit, result.stdout, result.stderr)
		}
	})

	t.Run("blocks with stale_proof when the proofs are missing [verifies SN-113~1]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		result := checkProject(t, directory)
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || len(document.Findings) != 1 || document.Findings[0].Category != domain.CategoryStaleProof ||
			document.Findings[0].Module != "musthold-dp" {
			t.Fatalf("exit %d, findings %+v", result.exit, document.Findings)
		}
	})

	t.Run("blocks with stale_proof once the code changed after the proof [verifies SN-111~2]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		if err := os.WriteFile(filepath.Join(directory, proofCode), []byte("def work(): return 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result := checkProject(t, directory)
		document := decode(t, result.stdout)
		if result.exit != app.ExitBlocked || document.Findings[0].Category != domain.CategoryStaleProof ||
			!strings.Contains(string(result.stdout), "code_tree file src/a.py changed") {
			t.Fatalf("exit %d, stdout %s", result.exit, result.stdout)
		}
	})

	t.Run("keeps the proof valid when a file outside every descriptor changes [verifies SN-132~1]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		for name, content := range map[string]string{"src/unrelated.py": "def other(): pass\n", "tests/helpers.py": "VALUE = 1\n"} {
			if err := os.WriteFile(filepath.Join(directory, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		result := checkProject(t, directory)
		if document := decode(t, result.stdout); result.exit != app.ExitClean || document.Verdict != domain.VerdictPass {
			t.Fatalf("exit %d, findings %+v", result.exit, document.Findings)
		}
	})

	t.Run("blocks with stale_proof once a test file bound to the criterion is missing from the proof [verifies SN-134~2]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		moved := "def works_too(record_property):\n    record_property(\"" + domain.TagKeyword + "\", \"REQ-1~1\")\n"
		if err := os.WriteFile(filepath.Join(directory, "tests", "t2.py"), []byte(moved), 0o644); err != nil {
			t.Fatal(err)
		}
		result := checkProject(t, directory)
		if !strings.Contains(string(result.stdout), "tests/t2.py holds a test bound to REQ-1~1") {
			t.Fatalf("exit %d, stdout %s", result.exit, result.stdout)
		}
	})

	t.Run("blocks with stale_proof when the proof lists a code file outside proofs.code [verifies SN-133~2]", func(t *testing.T) {
		directory := proofProject(t, strings.Replace(proofConfig, "code: [src]", "code: [lib]", 1))
		writeProofs(t, directory)
		result := checkProject(t, directory)
		if result.exit != app.ExitBlocked || !strings.Contains(string(result.stdout), "outside the configured proofs.code globs") {
			t.Fatalf("exit %d, stdout %s", result.exit, result.stdout)
		}
	})

	t.Run("blocks with stale_proof when the spec input is not the statement digest of the criterion [verifies SN-135~2]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		if err := os.WriteFile(filepath.Join(directory, "spec.md"), []byte("- **REQ-1~1**: THE SYSTEM SHALL work differently.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result := checkProject(t, directory)
		if result.exit != app.ExitBlocked || decode(t, result.stdout).Findings[0].Category != domain.CategoryStaleProof {
			t.Fatalf("exit %d, stdout %s", result.exit, result.stdout)
		}
	})

	t.Run("blocks with stale_proof once the statement changed without a revision bump [verifies SN-111~2]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		if err := os.WriteFile(filepath.Join(directory, "spec.md"), []byte("- **REQ-1~1**: THE SYSTEM SHALL work differently.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result := checkProject(t, directory)
		if result.exit != app.ExitBlocked || decode(t, result.stdout).Findings[0].Category != domain.CategoryStaleProof {
			t.Fatalf("exit %d, stdout %s", result.exit, result.stdout)
		}
	})

	t.Run("records the consumed proofs in extensions.proofs and validates against the report schema [verifies SN-125~1]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		result := checkProject(t, directory)
		var generic any
		if err := json.Unmarshal([]byte(result.stdout), &generic); err != nil {
			t.Fatal(err)
		}
		if err := reportSchema(t).Validate(generic); err != nil {
			t.Fatal(err)
		}
		var extension proof.Report
		if err := json.Unmarshal(decode(t, result.stdout).Extensions["proofs"], &extension); err != nil {
			t.Fatal(err)
		}
		if extension.Trust != proof.TrustLocal || extension.Manifest == nil || extension.Manifest.Digest == "" || len(extension.Consumed) != 1 ||
			extension.Consumed[0].State != proof.StateValid || len(extension.Criteria) != 1 || extension.Criteria[0].Ref != "REQ-1~1" ||
			extension.Criteria[0].Modules[proof.KindDiscriminatingPower].Outcome != proof.OutcomePass {
			t.Fatalf("extension %+v", extension)
		}
	})

	t.Run("writes the same report bytes for the same commit and the same proof files [verifies SN-125~1]", func(t *testing.T) {
		directory := proofProject(t, proofConfig)
		writeProofs(t, directory)
		first, second := checkProject(t, directory), checkProject(t, directory)
		if first.stdout != second.stdout {
			t.Fatal("two runs on the same inputs differ")
		}
	})

	t.Run("adds no extension and reads no proof without a proofs block [verifies SN-125~1]", func(t *testing.T) {
		directory := proofProject(t, "version: 1\nspecs: [spec.md]\nresults: [junit.xml]\n")
		result := checkProject(t, directory)
		if document := decode(t, result.stdout); len(document.Extensions) != 0 || result.exit != app.ExitClean {
			t.Fatalf("extensions %v", document.Extensions)
		}
	})
}

func TestEscalationSeverity(t *testing.T) {
	alignmentRequired := strings.Replace(proofConfig, "require: [discriminating-power]", "require: [alignment]", 1)

	severityOf := func(t *testing.T, config string) (domain.Severity, app.ExitCode) {
		t.Helper()
		directory := proofProject(t, config)
		result := checkProject(t, directory)
		document := decode(t, result.stdout)
		if len(document.Findings) != 1 || document.Findings[0].Category != domain.CategoryStaleProof {
			t.Fatalf("findings %+v", document.Findings)
		}
		return document.Findings[0].Severity, result.exit
	}

	t.Run("fails the verdict on an escalating finding by default [verifies SN-122~1]", func(t *testing.T) {
		if severity, exit := severityOf(t, alignmentRequired); severity != domain.SeverityError || exit != app.ExitBlocked {
			t.Fatalf("severity %s, exit %d", severity, exit)
		}
	})

	t.Run("only reports an escalating finding when the escalation is warning [verifies SN-122~1]", func(t *testing.T) {
		if severity, exit := severityOf(t, alignmentRequired+"  escalation: warning\n"); severity != domain.SeverityWarning || exit != app.ExitClean {
			t.Fatalf("severity %s, exit %d", severity, exit)
		}
	})

	t.Run("lets a severity set for the category win over the escalation [verifies SN-122~1]", func(t *testing.T) {
		config := alignmentRequired + "  escalation: warning\nseverities: {stale_proof: error}\n"
		if severity, exit := severityOf(t, config); severity != domain.SeverityError || exit != app.ExitBlocked {
			t.Fatalf("severity %s, exit %d", severity, exit)
		}
	})

	t.Run("never lowers a blocking finding with the escalation [verifies SN-122~1]", func(t *testing.T) {
		config := proofConfig + "  escalation: warning\n"
		if severity, exit := severityOf(t, config); severity != domain.SeverityError || exit != app.ExitBlocked {
			t.Fatalf("severity %s, exit %d", severity, exit)
		}
	})
}

// writePending writes a verified set that lists no statement and the given pending entries.
func writePending(t *testing.T, directory string, pending ...proof.Pending) {
	t.Helper()
	if pending == nil {
		pending = []proof.Pending{}
	}
	manifest, err := json.Marshal(proof.Manifest{VerifierVersion: "0.1.0", Trust: proof.TrustLocal, Statements: []proof.ManifestEntry{}, Pending: pending})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, ".musthold", "verified")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
}

func awaitingHost(reason proof.PendingReason) proof.Pending {
	return proof.Pending{Module: "musthold-dp", Criterion: "REQ-1~1", Reason: reason}
}

func TestHookOnPendingProofs(t *testing.T) {
	hook := func(t *testing.T, directory string) run {
		t.Helper()
		t.Setenv("CLAUDE_PROJECT_DIR", directory)
		return shallnotWithInput(t, "", "hook", "claude-stop")
	}
	ready := func(t *testing.T) string {
		t.Helper()
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		return proofProject(t, proofConfig)
	}

	t.Run("tells the agent which module waits, which skill to run and not to answer the requests itself [verifies SN-128~1]", func(t *testing.T) {
		directory := ready(t)
		writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
		result := hook(t, directory)
		if result.exit != claudeExitBlock {
			t.Fatalf("got %+v", result)
		}
		for _, expected := range []string{"musthold-dp", "musthold-dp-fulfil", "Never answer the requests yourself", "awaiting_host"} {
			if !strings.Contains(result.stderr, expected) {
				t.Errorf("no %q in:\n%s", expected, result.stderr)
			}
		}
		if strings.Contains(result.stderr, "Never delete or alter a tag") {
			t.Errorf("the tag prohibitions do not apply to a wait:\n%s", result.stderr)
		}
	})

	t.Run("tells the agent to report to the user and not to loop when the host is unresponsive [verifies SN-128~1]", func(t *testing.T) {
		directory := ready(t)
		writePending(t, directory, awaitingHost(proof.PendingHostUnresponsive))
		result := hook(t, directory)
		if result.exit != claudeExitBlock || !strings.Contains(result.stderr, "report this to the user, do not loop") || strings.Contains(result.stderr, "Run the fulfil skill") {
			t.Fatalf("got %+v", result)
		}
	})

	t.Run("lets the turn end after MaxHostWaits blocks on pending proofs and resets both counters [verifies SN-127~1]", func(t *testing.T) {
		directory := ready(t)
		writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
		for attempt := 1; attempt <= agenthook.MaxHostWaits; attempt++ {
			if result := hook(t, directory); result.exit != claudeExitBlock {
				t.Fatalf("attempt %d: %+v", attempt, result)
			}
		}
		given := hook(t, directory)
		if given.exit != claudeExitNotice || !strings.Contains(given.stderr, "still blocked") {
			t.Fatalf("got %+v", given)
		}
		if again := hook(t, directory); again.exit != claudeExitBlock {
			t.Fatalf("the counters were not reset: %+v", again)
		}
	})

	t.Run("counts blocks that are not pending apart, and neither resets the other [verifies SN-127~1]", func(t *testing.T) {
		directory := ready(t)
		statuses := []string{}
		step := func(pending bool) run {
			if pending {
				writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
			} else {
				writePending(t, directory)
			}
			result := hook(t, directory)
			statuses = append(statuses, map[app.ExitCode]string{claudeExitBlock: "block", claudeExitNotice: "give-up", app.ExitClean: "pass"}[result.exit])
			return result
		}
		// Three waits and three stale proofs, interleaved: all six block, the seventh gives up.
		for _, pending := range []bool{true, false, true, false, true, false} {
			if result := step(pending); result.exit != claudeExitBlock {
				t.Fatalf("sequence %v: %+v", statuses, result)
			}
		}
		if result := step(true); result.exit != claudeExitNotice {
			t.Fatalf("sequence %v: %+v", statuses, result)
		}
	})

	t.Run("counts as waits the blocks of a project that requires alignment and discriminating power while one module is pending [verifies SN-127~1]", func(t *testing.T) {
		isolateAttempts(t)
		t.Chdir(t.TempDir())
		both := strings.Replace(proofConfig, "require: [discriminating-power]", "require: [alignment, discriminating-power]", 1)
		directory := proofProject(t, both)
		writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
		first := hook(t, directory)
		if first.exit != claudeExitBlock || strings.Contains(first.stderr, "stale_proof") || !strings.Contains(first.stderr, "awaiting_proof") {
			t.Fatalf("got %+v", first)
		}
		for attempt := 2; attempt <= agenthook.MaxHostWaits; attempt++ {
			hook(t, directory)
		}
		if given := hook(t, directory); given.exit != claudeExitNotice {
			t.Fatalf("got %+v", given)
		}
	})

	t.Run("a pass resets both counters [verifies SN-127~1]", func(t *testing.T) {
		directory := ready(t)
		writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
		hook(t, directory)
		hook(t, directory)
		writeProofs(t, directory)
		if result := hook(t, directory); result.exit != app.ExitClean {
			t.Fatalf("got %+v", result)
		}
		writePending(t, directory, awaitingHost(proof.PendingAwaitingHost))
		for attempt := 1; attempt <= agenthook.MaxHostWaits; attempt++ {
			if result := hook(t, directory); result.exit != claudeExitBlock {
				t.Fatalf("attempt %d after the pass: %+v", attempt, result)
			}
		}
	})
}
