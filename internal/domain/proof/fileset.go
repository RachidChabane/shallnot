package proof

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
)

const (
	pathSeparator = "/"
	// fieldTerminator ends the path and the content of one file in the file-set digest stream.
	fieldTerminator = 0x00
)

// File is one file of a file set: a relative POSIX path and its content.
type File struct {
	Path    string
	Content []byte
}

// IsRelativePOSIXPath reports whether p is a non-empty relative path with "/"
// separators and no empty, "." or ".." segment.
func IsRelativePOSIXPath(p string) bool {
	if p == "" || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	for _, segment := range strings.Split(p, pathSeparator) {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// FileSetDigest returns the SHA-256 over the files sorted by path, each written
// as `path NUL content NUL`. The order of the files does not matter. A path that
// is not a relative POSIX path, or that occurs twice, is an error.
func FileSetDigest(files []File) (string, error) {
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b File) int { return strings.Compare(a.Path, b.Path) })

	hash := sha256.New()
	for index, file := range sorted {
		if !IsRelativePOSIXPath(file.Path) {
			return "", fmt.Errorf("file set: %q is not a relative POSIX path", file.Path)
		}
		if index > 0 && sorted[index-1].Path == file.Path {
			return "", fmt.Errorf("file set: path %q occurs twice", file.Path)
		}
		hash.Write([]byte(file.Path))
		hash.Write([]byte{fieldTerminator})
		hash.Write(file.Content)
		hash.Write([]byte{fieldTerminator})
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
