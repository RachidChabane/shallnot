// Package proofset reads the verified proof set that the signature verifier
// writes: its manifest, the statement files it lists, and the digests of
// the files its descriptors name, to tell a current proof from a stale one. It uses
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
	evidence.Derived.SpecFiles = map[string]string{}
	evidence.Derived.Files = map[string]string{}

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
	derived, err := source.derive(evidence.Statements)
	if err != nil {
		return evidence, err
	}
	evidence.Derived = derived
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

// derive reads what the checks compare against: the digest of each known spec file, and the digest of each file a
// code_tree or test_source descriptor of a well-formed statement names.
func (s Source) derive(statements []proof.LoadedStatement) (proof.DerivedInputs, error) {
	derived := proof.DerivedInputs{SpecFiles: map[string]string{}, Files: map[string]string{}}
	for _, path := range s.SpecFiles {
		relative, err := s.relative(path)
		if err != nil {
			return derived, err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return derived, err
		}
		derived.SpecFiles[relative] = proof.SHA256Hex(content)
	}
	for _, name := range descriptorNames(statements) {
		digest, present, err := s.digestOf(name)
		if err != nil {
			return derived, err
		}
		if present {
			derived.Files[name] = digest
		}
	}
	return derived, nil
}

// descriptorNames lists the distinct paths named by the file descriptors of the statements, in a stable order.
func descriptorNames(statements []proof.LoadedStatement) []string {
	seen := map[string]bool{}
	for _, loaded := range statements {
		if loaded.Statement == nil {
			continue
		}
		for _, input := range loaded.Statement.Core().Inputs {
			if input.Role.NamesFile() && proof.IsRelativePOSIXPath(input.Name) {
				seen[input.Name] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// digestOf hashes a file of the tree; a file that does not exist, or is not a regular file, is not present.
func (s Source) digestOf(name string) (digest string, present bool, err error) {
	path := s.resolve(filepath.FromSlash(name))
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", false, nil
	case err != nil:
		return "", false, err
	case !info.Mode().IsRegular():
		return "", false, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return proof.SHA256Hex(content), true, nil
}
