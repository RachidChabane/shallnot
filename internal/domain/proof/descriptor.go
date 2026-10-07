package proof

import (
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const pathSeparator = "/"

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

// NamesFile reports whether an input of the role is a per-file descriptor: its name is a path and its digest the
// SHA-256 of the file's exact bytes.
func (r InputRole) NamesFile() bool { return r == RoleCodeTree || r == RoleTestSource }

// Scope is where the configuration lets a proof's file descriptors point.
type Scope struct {
	// Code are the `proofs.code` entries: files, directories or globs.
	Code []string
	// Tests are the `tests` entries: files, directories or globs.
	Tests []string
	// Exclude are the excluded globs, the default dependency and build directories included.
	Exclude []string
}

// Contains reports whether the file descriptor of the role names a file the configuration covers: under one of the
// entries of its role and in no excluded path. A role that is not a file descriptor covers nothing.
func (s Scope) Contains(role InputRole, file string) bool {
	var entries []string
	switch role {
	case RoleCodeTree:
		entries = s.Code
	case RoleTestSource:
		entries = s.Tests
	default:
		return false
	}
	if !IsRelativePOSIXPath(file) || matchesAny(s.Exclude, file) {
		return false
	}
	for _, entry := range entries {
		if covers(path.Clean(entry), file) {
			return true
		}
	}
	return false
}

// covers tells whether an entry names the file itself, a directory above it, or a glob that matches it.
func covers(entry, file string) bool {
	if entry == file || strings.HasPrefix(file, entry+pathSeparator) {
		return true
	}
	matched, _ := doublestar.Match(entry, file)
	if matched {
		return true
	}
	matched, _ = doublestar.Match(entry+pathSeparator+"**", file)
	return matched
}

func matchesAny(patterns []string, file string) bool {
	for _, pattern := range patterns {
		if matched, _ := doublestar.Match(pattern, file); matched {
			return true
		}
	}
	return false
}
