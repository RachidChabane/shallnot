package proof_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

const (
	goldenDirectory = "testdata/golden"
	manifestName    = "manifest.json"
	checksumsName   = "testdata/sha256.manifest"
	decodeOK        = "ok"
	decodeUnknown   = "unknown_predicate_type"
)

type goldenEntry struct {
	File       string                `json:"file"`
	Decode     string                `json:"decode"`
	Violations []proof.ViolationCode `json:"violations"`
}

func readGoldenManifest(t *testing.T) []goldenEntry {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(goldenDirectory, manifestName))
	if err != nil {
		t.Fatal(err)
	}
	var entries []goldenEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("the golden manifest lists no vector")
	}
	return entries
}

func TestSyncedGoldens(t *testing.T) {
	t.Run("gives every golden proof the decode result and the violation codes its manifest states [verifies SN-106~1]", func(t *testing.T) {
		for _, entry := range readGoldenManifest(t) {
			if entry.File == manifestName {
				continue
			}
			t.Run(entry.File, func(t *testing.T) {
				document, err := os.ReadFile(filepath.Join(goldenDirectory, entry.File))
				if err != nil {
					t.Fatal(err)
				}
				statement, err := proof.Decode(document)
				if entry.Decode == decodeUnknown {
					if !errors.Is(err, proof.ErrUnknownPredicateType) {
						t.Fatalf("err = %v, want ErrUnknownPredicateType", err)
					}
					return
				}
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				want := slices.Clone(entry.Violations)
				slices.Sort(want)
				got := proof.Codes(statement.Validate())
				if len(got) != len(want) || (len(want) > 0 && !slices.Equal(got, want)) {
					t.Fatalf("violations = %v, want %v\n%s", got, want, proof.Describe(statement.Validate()))
				}
			})
		}
	})

	t.Run("covers valid statements of each kind and the claim rules the gate enforces [verifies SN-106~1]", func(t *testing.T) {
		valid := map[proof.Kind]int{}
		seen := map[proof.ViolationCode]bool{}
		for _, entry := range readGoldenManifest(t) {
			for _, code := range entry.Violations {
				seen[code] = true
			}
			if entry.Decode != decodeOK || len(entry.Violations) > 0 {
				continue
			}
			document, err := os.ReadFile(filepath.Join(goldenDirectory, entry.File))
			if err != nil {
				t.Fatal(err)
			}
			statement, err := proof.Decode(document)
			if err != nil {
				t.Fatal(err)
			}
			valid[statement.Kind]++
		}
		for _, kind := range proof.Kinds() {
			if valid[kind] == 0 {
				t.Errorf("no valid golden of kind %s", kind)
			}
		}
		for _, code := range []proof.ViolationCode{
			proof.CodeJudgedWithoutConfidence, proof.CodeExecutedWithConfidence, proof.CodeWrongClass, proof.CodeSubject,
			proof.CodeUnsorted, proof.CodeCacheKeyMismatch, proof.CodeUncorroboratedClaim, proof.CodeExecutedWithCorroboration,
			proof.CodeExecutedWithExecutor, proof.CodeInconsistentAttribution, proof.CodeInputDescriptor,
		} {
			if !seen[code] {
				t.Errorf("no golden raises %s", code)
			}
		}
	})

	t.Run("recomputes the cache key every well-formed golden records [verifies SN-103~1]", func(t *testing.T) {
		recomputed := 0
		for _, entry := range readGoldenManifest(t) {
			if entry.Decode != decodeOK || slices.Contains(entry.Violations, proof.CodeCacheKeyMismatch) {
				continue
			}
			document, err := os.ReadFile(filepath.Join(goldenDirectory, entry.File))
			if err != nil {
				t.Fatal(err)
			}
			statement, err := proof.Decode(document)
			if err != nil {
				t.Fatal(err)
			}
			core := statement.Core()
			key, err := proof.CacheKey(statement.Subject, core.Inputs)
			if err != nil {
				t.Fatalf("%s: %v", entry.File, err)
			}
			if key != core.CacheKey {
				t.Errorf("%s: recomputed %s, recorded %s", entry.File, key, core.CacheKey)
			}
			recomputed++
		}
		if recomputed == 0 {
			t.Fatal("no golden recomputed")
		}
	})

	t.Run("matches the SHA-256 of every copied file against the manifest of the copy [verifies SN-106~1]", func(t *testing.T) {
		listed := map[string]bool{}
		file, err := os.Open(checksumsName)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			digest, name, found := strings.Cut(scanner.Text(), "  ")
			if !found {
				t.Fatalf("malformed manifest line %q", scanner.Text())
			}
			content, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != digest {
				t.Errorf("%s differs from the digest in the manifest of the copy", name)
			}
			listed[name] = true
		}
		for _, directory := range []string{"golden", "schemas"} {
			entries, err := os.ReadDir(filepath.Join("testdata", directory))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if name := directory + "/" + entry.Name(); !listed[name] {
					t.Errorf("%s is copied but not in the manifest", name)
				}
			}
		}
	})
}

func TestSyncScript(t *testing.T) {
	t.Run("the sync script refuses a call without a spec-proof checkout and a checkout without goldens [verifies SN-106~1]", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the script needs bash")
		}
		script, err := filepath.Abs("../../../script/sync-proof-fixtures")
		if err != nil {
			t.Fatal(err)
		}
		for name, args := range map[string][]string{"no argument": nil, "an empty checkout": {t.TempDir()}} {
			output, err := exec.Command("bash", append([]string{script}, args...)...).CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 2 {
				t.Errorf("%s: err = %v, output %s", name, err, output)
			}
		}
	})
}
