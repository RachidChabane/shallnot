package proofset_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/adapters/proofset"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
	"github.com/RachidChabane/shallnot/internal/domain/proof/prooftest"
)

func write(t *testing.T, root, name string, content []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func manifestOf(t *testing.T, trust proof.Trust, statements map[string][]byte, listedDigests map[string]string) []byte {
	t.Helper()
	manifest := proof.Manifest{VerifierVersion: "0.1.0", Trust: trust, Statements: []proof.ManifestEntry{}, Pending: []proof.Pending{}}
	if trust == proof.TrustSigned {
		manifest.TrustedRootDigest = prooftest.Digest("root")
		manifest.PolicyDigest = prooftest.Digest("policy")
	}
	for name, content := range statements {
		digest := proof.SHA256Hex(content)
		if override, ok := listedDigests[name]; ok {
			digest = override
		}
		manifest.Statements = append(manifest.Statements, proof.ManifestEntry{Path: name, Digest: digest})
	}
	sortEntries(manifest.Statements)
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func sortEntries(entries []proof.ManifestEntry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j-1].Path > entries[j].Path; j-- {
			entries[j-1], entries[j] = entries[j], entries[j-1]
		}
	}
}

func source(root string) proofset.Source {
	config := proof.DefaultConfig()
	config.Code = []string{"src"}
	return proofset.Source{
		Config: config, WorkDir: root, SpecFiles: []string{filepath.Join(root, "specs", "a.md")},
		SpecGlobs: []string{"specs"}, TestGlobs: []string{"tests"}, DefaultExcludes: true,
	}
}

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "specs/a.md", []byte("spec"))
	write(t, root, "tests/t_test.py", []byte("test"))
	write(t, root, "src/a.py", []byte("code"))
	return root
}

func statementFiles() map[string][]byte {
	digest := prooftest.Digest("statement")
	return map[string][]byte{
		"dp/PARK-2~1/key.json": prooftest.Bytes(prooftest.DiscriminatingPower("PARK-2", 1, digest)),
	}
}

func TestLoad(t *testing.T) {
	t.Run("reads the manifest and the statements it lists and records their digests [verifies SN-109~1]", func(t *testing.T) {
		root := project(t)
		files := statementFiles()
		for name, content := range files {
			write(t, root, ".spec-proof/verified/"+name, content)
		}
		manifest := manifestOf(t, proof.TrustSigned, files, nil)
		write(t, root, ".spec-proof/verified/manifest.json", manifest)

		evidence, err := proofset.Load(source(root))
		if err != nil {
			t.Fatal(err)
		}
		if !evidence.Usable() || evidence.ManifestDigest != proof.SHA256Hex(manifest) || len(evidence.Statements) != 1 {
			t.Fatalf("got %+v", evidence)
		}
		loaded := evidence.Statements[0]
		if loaded.Problem != "" || loaded.Statement == nil || loaded.Digest != proof.SHA256Hex(files["dp/PARK-2~1/key.json"]) {
			t.Fatalf("got %+v", loaded)
		}
	})

	t.Run("reports a missing manifest as absent, not as an error [verifies SN-109~1]", func(t *testing.T) {
		evidence, err := proofset.Load(source(project(t)))
		if err != nil || evidence.ManifestPresent || evidence.Usable() {
			t.Fatalf("got %+v, %v", evidence, err)
		}
	})

	t.Run("reports a malformed manifest and ignores its statements [verifies SN-109~1]", func(t *testing.T) {
		for name, content := range map[string]string{
			"not JSON":       "{",
			"invalid trust":  `{"verifierVersion":"1","trust":"maybe","statements":[],"pending":[]}`,
			"unsorted":       `{"verifierVersion":"1","trust":"local","statements":[{"path":"b.json","digest":"` + prooftest.Digest("b") + `"},{"path":"a.json","digest":"` + prooftest.Digest("a") + `"}],"pending":[]}`,
			"escaping path":  `{"verifierVersion":"1","trust":"local","statements":[{"path":"../x.json","digest":"` + prooftest.Digest("b") + `"}],"pending":[]}`,
			"bad pending":    `{"verifierVersion":"1","trust":"local","statements":[],"pending":[{"module":"m","criterion":"PARK-2","reason":"awaiting_host"}]}`,
			"unknown reason": `{"verifierVersion":"1","trust":"local","statements":[],"pending":[{"module":"m","criterion":"PARK-2~1","reason":"quota_paused"}]}`,
		} {
			root := project(t)
			write(t, root, ".spec-proof/verified/manifest.json", []byte(content))
			evidence, err := proofset.Load(source(root))
			if err != nil || !evidence.ManifestPresent || evidence.ManifestProblem == "" || evidence.Usable() || len(evidence.Statements) != 0 {
				t.Errorf("%s: got %+v, %v", name, evidence, err)
			}
		}
	})

	t.Run("flags a statement that differs from the manifest, does not decode or breaks an invariant [verifies SN-109~1]", func(t *testing.T) {
		root := project(t)
		files := map[string][]byte{
			"a-tampered.json": prooftest.Bytes(prooftest.DiscriminatingPower("PARK-2", 1, prooftest.Digest("x"))),
			"b-garbage.json":  []byte("not json"),
			"c-broken.json":   []byte(strings.Replace(string(statementFiles()["dp/PARK-2~1/key.json"]), `"correctCodePasses":{"value":true,"class":"executed"}`, `"correctCodePasses":{"value":true,"class":"judged"}`, 1)),
			"d-missing.json":  nil,
		}
		for name, content := range files {
			if content != nil {
				write(t, root, ".spec-proof/verified/"+name, content)
			}
		}
		listed := map[string]string{"a-tampered.json": prooftest.Digest("other bytes")}
		manifestFiles := map[string][]byte{}
		for name, content := range files {
			manifestFiles[name] = content
		}
		write(t, root, ".spec-proof/verified/manifest.json", manifestOf(t, proof.TrustLocal, manifestFiles, listed))
		evidence, err := proofset.Load(source(root))
		if err != nil {
			t.Fatal(err)
		}
		if len(evidence.Statements) != 4 {
			t.Fatalf("got %d statements", len(evidence.Statements))
		}
		for _, loaded := range evidence.Statements {
			if loaded.Problem == "" || loaded.Statement != nil {
				t.Errorf("%s was accepted: %+v", loaded.Path, loaded)
			}
		}
	})

	t.Run("ignores a statement of another predicate type and a file the manifest does not list [verifies SN-109~1]", func(t *testing.T) {
		root := project(t)
		future := []byte(`{"_type":"` + proof.StatementType + `","subject":[],"predicateType":"https://rachid-chabane.com/attestations/future/v1","predicate":{}}`)
		write(t, root, ".spec-proof/verified/future.json", future)
		write(t, root, ".spec-proof/verified/unlisted.json", []byte("junk"))
		write(t, root, ".spec-proof/verified/manifest.json", manifestOf(t, proof.TrustLocal, map[string][]byte{"future.json": future}, nil))
		evidence, err := proofset.Load(source(root))
		if err != nil || len(evidence.Statements) != 1 || !evidence.Statements[0].Ignored || evidence.Statements[0].Problem != "" {
			t.Fatalf("got %+v, %v", evidence.Statements, err)
		}
	})
}

func TestDerivedDigests(t *testing.T) {
	expected := func(t *testing.T, files ...proof.File) string {
		t.Helper()
		digest, err := proof.FileSetDigest(files)
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}

	t.Run("recomputes the spec, test source and code digests with the file-set rule [verifies SN-110~1]", func(t *testing.T) {
		root := project(t)
		derived, err := proofset.Load(source(root))
		if err != nil {
			t.Fatal(err)
		}
		got := derived.Derived
		if got.Spec.Digest != expected(t, proof.File{Path: "specs/a.md", Content: []byte("spec")}) ||
			got.TestSource.Digest != expected(t, proof.File{Path: "tests/t_test.py", Content: []byte("test")}) ||
			got.CodeTree.Digest != expected(t, proof.File{Path: "src/a.py", Content: []byte("code")}) {
			t.Fatalf("got %+v %+v %+v", got.Spec, got.TestSource, got.CodeTree)
		}
		if got.SpecFiles["specs/a.md"] != proof.SHA256Hex([]byte("spec")) || got.CodeTree.Globs[0] != "src" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("changes a digest when a file changes and leaves the others alone [verifies SN-110~1]", func(t *testing.T) {
		root := project(t)
		before, _ := proofset.Load(source(root))
		write(t, root, "src/a.py", []byte("edited"))
		after, _ := proofset.Load(source(root))
		if before.Derived.CodeTree.Digest == after.Derived.CodeTree.Digest || before.Derived.Spec.Digest != after.Derived.Spec.Digest {
			t.Fatal("digests did not follow the edit")
		}
	})

	t.Run("leaves out excluded paths and dependency directories and expands globs [verifies SN-110~1]", func(t *testing.T) {
		root := project(t)
		write(t, root, "src/node_modules/dep/x.js", []byte("dependency"))
		write(t, root, "src/generated.py", []byte("generated"))
		configured := source(root)
		configured.Exclude = []string{"src/generated.py"}
		configured.Config.Code = []string{"src/**/*.py"}
		got, err := proofset.Load(configured)
		if err != nil {
			t.Fatal(err)
		}
		if got.Derived.CodeTree.Digest != expected(t, proof.File{Path: "src/a.py", Content: []byte("code")}) {
			t.Fatal("excluded or dependency files entered the set")
		}
	})

	t.Run("cannot derive the code set without proofs.code [verifies SN-110~1]", func(t *testing.T) {
		configured := source(project(t))
		configured.Config.Code = nil
		got, err := proofset.Load(configured)
		if err != nil || got.Derived.CodeTree != nil {
			t.Fatalf("got %+v, %v", got.Derived.CodeTree, err)
		}
	})
}
