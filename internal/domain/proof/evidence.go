package proof

// DerivedInputs are the digests the gate reads from the tree: the files the proofs name and the known spec files.
type DerivedInputs struct {
	// SpecFiles maps the relative path of each known spec file to the SHA-256 of its content.
	SpecFiles map[string]string
	// Files maps the relative POSIX path of each file a code_tree or test_source descriptor names to the SHA-256 of
	// its bytes; a file that is absent or unreadable as a regular file has no entry.
	Files map[string]string
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
