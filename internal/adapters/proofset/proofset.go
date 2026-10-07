// Package proofset reads the verified proof set that the signature verifier
// writes: its manifest, the statement files it lists, and the digests of the
// tree the gate recomputes to tell a current proof from a stale one. It uses
// the operating system and the standard JSON library only.
package proofset

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/RachidChabane/shallnot/internal/adapters/scan"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// ManifestName is the file that lists the verified statements.
const ManifestName = "manifest.json"

const problemSeparator = "; "

// Source says where the proof set is and how the gate derives the input digests.
type Source struct {
	Config proof.Config
	// WorkDir is the directory of the configuration file; "" means the working directory.
	WorkDir string
	// SpecFiles are the known spec files, as paths openable from the working directory.
	SpecFiles []string
	// SpecGlobs and TestGlobs are the configured `specs` (with `focus`) and `tests` entries, relative to WorkDir.
	SpecGlobs []string
	TestGlobs []string
	// Exclude are the configured exclude globs; DefaultExcludes adds the dependency and build directories.
	Exclude         []string
	DefaultExcludes bool
}

func (s Source) base() string {
	if s.WorkDir == "" {
		return "."
	}
	return s.WorkDir
}

func (s Source) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.base(), path)
}

func (s Source) relative(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(s.base())
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(base, absolute)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

// Load reads the verified set. A missing manifest, an unreadable statement or a file that breaks an invariant is
// reported in the evidence, never as an error: such a proof counts as absent. An error means the tree could not be read.
func Load(source Source) (proof.Evidence, error) {
	directory := source.resolve(source.Config.VerifiedDir)
	evidence := proof.Evidence{ManifestPath: filepath.ToSlash(filepath.Join(directory, ManifestName))}
	derived, err := source.derive()
	if err != nil {
		return evidence, err
	}
	evidence.Derived = derived

	raw, err := os.ReadFile(filepath.Join(directory, ManifestName))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return evidence, nil
	case err != nil:
		evidence.ManifestPresent = true
		evidence.ManifestProblem = err.Error()
		return evidence, nil
	}
	evidence.ManifestPresent = true
	evidence.ManifestDigest = proof.SHA256Hex(raw)
	manifest, violations, err := proof.ParseManifest(raw)
	switch {
	case err != nil:
		evidence.ManifestProblem = err.Error()
		return evidence, nil
	case len(violations) > 0:
		evidence.ManifestProblem = proof.Describe(violations)
		return evidence, nil
	}
	evidence.Manifest = manifest
	for _, entry := range manifest.Statements {
		evidence.Statements = append(evidence.Statements, loadStatement(directory, entry))
	}
	return evidence, nil
}

func loadStatement(directory string, entry proof.ManifestEntry) proof.LoadedStatement {
	loaded := proof.LoadedStatement{Path: filepath.ToSlash(filepath.Join(directory, filepath.FromSlash(entry.Path)))}
	raw, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(entry.Path)))
	if err != nil {
		loaded.Problem = err.Error()
		return loaded
	}
	loaded.Digest = proof.SHA256Hex(raw)
	if loaded.Digest != entry.Digest {
		loaded.Problem = fmt.Sprintf("its SHA-256 %s is not the %s the manifest lists", loaded.Digest, entry.Digest)
		return loaded
	}
	statement, err := proof.Decode(raw)
	if errors.Is(err, proof.ErrUnknownPredicateType) {
		loaded.Ignored = true
		return loaded
	}
	if err != nil {
		loaded.Problem = err.Error()
		return loaded
	}
	if violations := statement.Validate(); len(violations) > 0 {
		lines := make([]string, len(violations))
		for index, violation := range violations {
			lines[index] = violation.String()
		}
		loaded.Problem = strings.Join(lines, problemSeparator)
		return loaded
	}
	loaded.Statement = &statement
	return loaded
}

// derive recomputes the digests of the spec, test source and code sets from the tree.
func (s Source) derive() (proof.DerivedInputs, error) {
	var derived proof.DerivedInputs
	specs, err := s.readFiles(s.SpecFiles)
	if err != nil {
		return derived, err
	}
	derived.SpecFiles = map[string]string{}
	for _, file := range specs {
		derived.SpecFiles[file.Path] = proof.SHA256Hex(file.Content)
	}
	derived.Spec = fileSet(specs, s.SpecGlobs)
	tests, err := s.collect(s.TestGlobs)
	if err != nil {
		return derived, err
	}
	derived.TestSource = fileSet(tests, s.TestGlobs)
	if len(s.Config.Code) > 0 {
		code, err := s.collect(s.Config.Code)
		if err != nil {
			return derived, err
		}
		derived.CodeTree = fileSet(code, s.Config.Code)
	}
	return derived, nil
}

// fileSet digests the files; a set the format cannot express (a path outside the tree) is not derivable.
func fileSet(files []proof.File, globs []string) *proof.FileSet {
	if len(globs) == 0 {
		return nil
	}
	digest, err := proof.FileSetDigest(files)
	if err != nil {
		return nil
	}
	sorted := append([]string(nil), globs...)
	sort.Strings(sorted)
	return &proof.FileSet{Globs: sorted, Digest: digest}
}

func (s Source) readFiles(paths []string) ([]proof.File, error) {
	files := make([]proof.File, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		relative, err := s.relative(path)
		if err != nil {
			return nil, err
		}
		if seen[relative] {
			continue
		}
		seen[relative] = true
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, proof.File{Path: relative, Content: content})
	}
	return files, nil
}

// collect expands patterns: a directory contributes every regular file under it, a glob every file it matches.
// Excluded paths and, by default, dependency and build directories are left out.
func (s Source) collect(patterns []string) ([]proof.File, error) {
	var paths []string
	for _, pattern := range patterns {
		matched, err := s.expand(s.resolve(pattern))
		if err != nil {
			return nil, err
		}
		paths = append(paths, matched...)
	}
	kept := paths[:0]
	for _, path := range paths {
		relative, err := s.relative(path)
		if err != nil {
			return nil, err
		}
		if !s.excluded(relative) {
			kept = append(kept, path)
		}
	}
	return s.readFiles(kept)
}

func (s Source) expand(pattern string) ([]string, error) {
	info, err := os.Stat(pattern)
	if err == nil && info.IsDir() {
		var files []string
		err := filepath.WalkDir(pattern, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type().IsRegular() {
				files = append(files, path)
			}
			return nil
		})
		return files, err
	}
	if err == nil {
		return []string{pattern}, nil
	}
	return doublestar.FilepathGlob(filepath.ToSlash(pattern), doublestar.WithFilesOnly())
}

func (s Source) excluded(relative string) bool {
	patterns := append([]string(nil), s.Exclude...)
	if s.DefaultExcludes {
		patterns = append(patterns, scan.DefaultExcludes...)
	}
	for _, pattern := range patterns {
		if matched, _ := doublestar.Match(pattern, relative); matched {
			return true
		}
	}
	return false
}
