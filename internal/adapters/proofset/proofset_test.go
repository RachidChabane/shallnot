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
	return proofset.Source{Config: proof.DefaultConfig(), WorkDir: root, SpecFiles: []string{filepath.Join(root, "specs", "a.md")}}
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

// writeProofNaming writes a verified set holding one proof whose inputs are the given descriptors.
func writeProofNaming(t *testing.T, root string, inputs ...proof.Input) {
	t.Helper()
	statement := prooftest.Seal(prooftest.DiscriminatingPower("PARK-2", 1, prooftest.Digest("statement"), inputs...))
	content := prooftest.Bytes(statement)
	write(t, root, ".spec-proof/verified/dp/PARK-2~1/key.json", content)
	write(t, root, ".spec-proof/verified/manifest.json", manifestOf(t, proof.TrustLocal, map[string][]byte{"dp/PARK-2~1/key.json": content}, nil))
}

func fileInput(role proof.InputRole, name, content string) proof.Input {
	return proof.Input{Role: role, Name: name, Digest: proof.Digest{SHA256: proof.SHA256Hex([]byte(content))}}
}

func TestDerivedDigests(t *testing.T) {
	t.Run("reads the digest of each file a descriptor names and of each known spec file [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"), fileInput(proof.RoleTestSource, "tests/t_test.py", "test"))
		got, err := proofset.Load(source(root))
		if err != nil {
			t.Fatal(err)
		}
		files := got.Derived.Files
		if len(files) != 2 || files["src/a.py"] != proof.SHA256Hex([]byte("code")) || files["tests/t_test.py"] != proof.SHA256Hex([]byte("test")) {
			t.Fatalf("got %+v", files)
		}
		if got.Derived.SpecFiles["specs/a.md"] != proof.SHA256Hex([]byte("spec")) {
			t.Fatalf("got %+v", got.Derived.SpecFiles)
		}
	})

	t.Run("reads no file that no descriptor names [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		write(t, root, "src/unlisted.py", []byte("unlisted"))
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"))
		got, err := proofset.Load(source(root))
		if err != nil || len(got.Derived.Files) != 1 {
			t.Fatalf("got %+v, %v", got.Derived.Files, err)
		}
	})

	t.Run("follows an edit of a named file and leaves the others alone [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"), fileInput(proof.RoleTestSource, "tests/t_test.py", "test"))
		before, _ := proofset.Load(source(root))
		write(t, root, "src/a.py", []byte("edited"))
		after, _ := proofset.Load(source(root))
		if before.Derived.Files["src/a.py"] == after.Derived.Files["src/a.py"] || before.Derived.Files["tests/t_test.py"] != after.Derived.Files["tests/t_test.py"] {
			t.Fatal("digests did not follow the edit")
		}
	})

	t.Run("records no entry for a file that is absent or a directory [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		write(t, root, "src/dir/x.py", []byte("x"))
		writeProofNaming(t, root,
			fileInput(proof.RoleCodeTree, "src/gone.py", "code"), fileInput(proof.RoleCodeTree, "src/dir", "code"),
			fileInput(proof.RoleTestSource, "tests/t_test.py", "test"))
		got, err := proofset.Load(source(root))
		if err != nil || len(got.Derived.Files) != 1 || got.Derived.Files["tests/t_test.py"] == "" {
			t.Fatalf("got %+v, %v", got.Derived.Files, err)
		}
	})

	t.Run("resolves descriptors against the configuration directory [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"))
		configured := source(root)
		configured.WorkDir = root
		got, err := proofset.Load(configured)
		if err != nil || got.Derived.Files["src/a.py"] == "" {
			t.Fatalf("got %+v, %v", got.Derived.Files, err)
		}
	})

	t.Run("resolves descriptors against the configuration directory whatever the working directory is [verifies SN-110~3]", func(t *testing.T) {
		root := project(t)
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"))
		elsewhere := t.TempDir()
		write(t, elsewhere, "src/a.py", []byte("another file with the same relative path"))
		t.Chdir(elsewhere)
		configured := source(root)
		configured.WorkDir = root
		got, err := proofset.Load(configured)
		if err != nil || got.Derived.Files["src/a.py"] != proof.SHA256Hex([]byte("code")) {
			t.Fatalf("got %+v, %v", got.Derived.Files, err)
		}
	})

	t.Run("resolves descriptors against a configuration directory given relative to the working directory [verifies SN-110~3]", func(t *testing.T) {
		parent := t.TempDir()
		root := filepath.Join(parent, "project")
		write(t, root, "specs/a.md", []byte("spec"))
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "src/a.py", "code"))
		write(t, root, "src/a.py", []byte("code"))
		t.Chdir(parent)
		configured := proofset.Source{Config: proof.DefaultConfig(), WorkDir: "project", SpecFiles: []string{filepath.Join("project", "specs", "a.md")}}
		got, err := proofset.Load(configured)
		if err != nil || got.Derived.Files["src/a.py"] != proof.SHA256Hex([]byte("code")) || got.Derived.SpecFiles["specs/a.md"] == "" {
			t.Fatalf("got %+v %+v, %v", got.Derived.Files, got.Derived.SpecFiles, err)
		}
	})

	t.Run("treats a proof whose descriptor names a path outside the tree as malformed and reads no file for it [verifies SN-136~1]", func(t *testing.T) {
		root := project(t)
		writeProofNaming(t, root, fileInput(proof.RoleCodeTree, "../outside.py", "code"), fileInput(proof.RoleTestSource, "tests/t_test.py", "test"))
		got, err := proofset.Load(source(root))
		if err != nil || len(got.Statements) != 1 || got.Statements[0].Problem == "" || len(got.Derived.Files) != 0 {
			t.Fatalf("got %+v %+v, %v", got.Statements, got.Derived.Files, err)
		}
	})
}
