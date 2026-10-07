package proof

// FileSet is a set of files as the gate derives it from its own configuration, with the digest of the set.
type FileSet struct {
	// Globs are the configured patterns the set was expanded from, sorted.
	Globs  []string
	Digest string
}

// DerivedInputs are the input digests the gate recomputes from the tree. A nil set means the configuration does
// not name that set, so the input cannot be checked.
type DerivedInputs struct {
	Spec       *FileSet
	TestSource *FileSet
	CodeTree   *FileSet
	// SpecFiles maps the relative path of each known spec file to the SHA-256 of its content.
	SpecFiles map[string]string
}

// Set returns the derived set of an input role, and false for a role the gate cannot derive.
func (d DerivedInputs) Set(role InputRole) (*FileSet, bool) {
	switch role {
	case RoleSpec:
		return d.Spec, true
	case RoleTestSource:
		return d.TestSource, true
	case RoleCodeTree:
		return d.CodeTree, true
	default:
		return nil, false
	}
}

// LoadedStatement is one statement file of the verified set, as the adapter read it.
type LoadedStatement struct {
	// Path is the path of the file as shown in reports.
	Path string
	// Digest is the SHA-256 of the file's bytes.
	Digest string
	// Statement is set unless the file was ignored or malformed.
	Statement *Statement
	// Ignored is true for a statement of a predicate type outside the format.
	Ignored bool
	// Problem is why the file cannot be used: it is unreadable, does not match the manifest, does not decode or breaks an invariant.
	Problem string
}

// Evidence is everything the gate read from the verified proof set, before any judgment.
type Evidence struct {
	// ManifestPath is where the manifest is, or would be.
	ManifestPath string
	// ManifestPresent is false when the verified set holds no manifest.
	ManifestPresent bool
	// ManifestDigest is the SHA-256 of the manifest's bytes, empty when absent.
	ManifestDigest string
	// ManifestProblem is why a present manifest cannot be used.
	ManifestProblem string
	Manifest        Manifest
	Statements      []LoadedStatement
	Derived         DerivedInputs
}

// Usable reports whether the manifest is present and well formed.
func (e Evidence) Usable() bool { return e.ManifestPresent && e.ManifestProblem == "" }
