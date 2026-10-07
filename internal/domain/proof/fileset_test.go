package proof_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

func TestFileSetDigest(t *testing.T) {
	files := []proof.File{{Path: "b.txt", Content: []byte("two")}, {Path: "a/c.txt", Content: []byte("one")}}

	t.Run("hashes path NUL content NUL over the files sorted by path [verifies SN-107~1]", func(t *testing.T) {
		expected := sha256.Sum256([]byte("a/c.txt\x00one\x00b.txt\x00two\x00"))
		got, err := proof.FileSetDigest(files)
		if err != nil {
			t.Fatal(err)
		}
		if got != hex.EncodeToString(expected[:]) {
			t.Fatalf("digest %s", got)
		}
	})

	t.Run("does not depend on the order of the files [verifies SN-107~1]", func(t *testing.T) {
		forward, _ := proof.FileSetDigest(files)
		backward, _ := proof.FileSetDigest([]proof.File{files[1], files[0]})
		if forward != backward {
			t.Fatalf("%s != %s", forward, backward)
		}
	})

	t.Run("changes with any path or content [verifies SN-107~1]", func(t *testing.T) {
		base, _ := proof.FileSetDigest(files)
		renamed, _ := proof.FileSetDigest([]proof.File{{Path: "b2.txt", Content: []byte("two")}, files[1]})
		edited, _ := proof.FileSetDigest([]proof.File{{Path: "b.txt", Content: []byte("tw0")}, files[1]})
		if base == renamed || base == edited {
			t.Fatal("a change went unnoticed")
		}
	})

	t.Run("refuses a path that is not relative POSIX or that occurs twice [verifies SN-107~1]", func(t *testing.T) {
		for _, path := range []string{"/abs.txt", "../up.txt", "a//b.txt", "a/./b.txt", `a\b.txt`, ""} {
			if _, err := proof.FileSetDigest([]proof.File{{Path: path}}); err == nil {
				t.Errorf("accepted %q", path)
			}
		}
		if _, err := proof.FileSetDigest([]proof.File{files[0], files[0]}); err == nil {
			t.Error("accepted a duplicate path")
		}
	})
}
