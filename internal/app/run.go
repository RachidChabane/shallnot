package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/RachidChabane/shallnot/internal/adapters/junitxml"
	"github.com/RachidChabane/shallnot/internal/adapters/scan"
	"github.com/RachidChabane/shallnot/internal/adapters/spec"
	"github.com/RachidChabane/shallnot/internal/analysis"
	"github.com/RachidChabane/shallnot/internal/domain"
)

const resultsExtension = ".xml"

// Run executes one traceability check. An error means the tool could not
// produce a verdict; it is never a statement about coverage.
func Run(settings Settings) (Outcome, error) {
	ids, err := domain.NewIDPattern(settings.IDPattern)
	if err != nil {
		return Outcome{}, err
	}
	grammar := domain.TagGrammar{IDs: ids}
	focusMatchers, err := validateFocusIDs(settings.FocusIDs)
	if err != nil {
		return Outcome{}, err
	}

	specFiles, focusFiles, err := resolveSpecs(settings)
	if err != nil {
		return Outcome{}, err
	}
	resultsFiles, err := resolveResults(settings)
	if err != nil {
		return Outcome{}, err
	}

	input := analysis.Input{Policy: settings.Policy}
	focus := focusRule{files: absoluteSet(focusFiles), idPatterns: focusMatchers}
	loader := spec.NewLoader(ids)
	for _, file := range specFiles {
		document, err := loader.LoadFile(file.path, file.displayPath)
		if err != nil {
			return Outcome{}, err
		}
		input.SpecProblems = append(input.SpecProblems, document.Problems...)
		for _, requirement := range document.Requirements {
			input.Requirements = append(input.Requirements, analysis.FocusedRequirement{
				Requirement: requirement,
				InFocus:     focus.includes(file, requirement.ID),
			})
		}
	}

	reader := junitxml.Reader{Grammar: grammar}
	for _, file := range resultsFiles {
		testCases, err := reader.ReadFile(file.path, file.displayPath)
		if err != nil {
			return Outcome{}, err
		}
		input.TestCases = append(input.TestCases, testCases...)
	}

	scanner := scan.Scanner{
		Extractors: scan.DefaultExtractors(grammar),
		Excludes:   excludesOf(settings),
		Skip:       absoluteSet(append(append([]inputFile(nil), specFiles...), resultsFiles...)),
	}
	testRoots := cleanRoots(settings.Tests)
	for _, root := range testRoots {
		tags, err := scanner.ScanRoot(root)
		if err != nil {
			return Outcome{}, fmt.Errorf("test root %q: %w", root, err)
		}
		input.SourceTags = append(input.SourceTags, tags...)
	}

	return Outcome{
		Inputs: Inputs{
			IDPattern:    ids.Source(),
			SpecFiles:    displayPaths(specFiles),
			FocusFiles:   displayPaths(focusFiles),
			FocusIDs:     append([]string(nil), settings.FocusIDs...),
			TestRoots:    testRoots,
			ResultsFiles: displayPaths(resultsFiles),
		},
		Policy:   settings.Policy,
		Analysis: analysis.Analyze(input),
	}, nil
}

// resolveSpecs returns the known spec files (specs and focus together) and the focus files.
func resolveSpecs(settings Settings) (known, focus []inputFile, err error) {
	if len(settings.Specs) == 0 && len(settings.Focus) == 0 {
		return nil, nil, errors.New("no spec given: pass --specs or --focus, or set `specs` in the config file")
	}
	focus, err = expand("focus", settings.Focus, spec.Extensions())
	if err != nil {
		return nil, nil, err
	}
	known, err = expand("spec", append(append([]string(nil), settings.Specs...), settings.Focus...), spec.Extensions())
	return known, focus, err
}

func resolveResults(settings Settings) ([]inputFile, error) {
	if len(settings.Results) == 0 {
		return nil, errors.New("no results file given: pass --results or set `results` in the config file")
	}
	return expand("results", settings.Results, []string{resultsExtension})
}

func validateFocusIDs(patterns []string) ([]string, error) {
	for _, pattern := range patterns {
		if !doublestar.ValidatePattern(pattern) {
			return nil, fmt.Errorf("focus id pattern %q is not a valid glob", pattern)
		}
	}
	return patterns, nil
}

func excludesOf(settings Settings) []string {
	var excludes []string
	if settings.UseDefaultExcludes {
		excludes = append(excludes, scan.DefaultExcludes...)
	}
	return append(excludes, settings.Exclude...)
}

func cleanRoots(roots []string) []string {
	seen := map[string]bool{}
	var cleaned []string
	for _, root := range roots {
		display := filepath.ToSlash(filepath.Clean(root))
		if !seen[display] {
			seen[display] = true
			cleaned = append(cleaned, display)
		}
	}
	sort.Strings(cleaned)
	return cleaned
}

func absoluteSet(files []inputFile) map[string]bool {
	set := make(map[string]bool, len(files))
	for _, file := range files {
		set[file.absolute] = true
	}
	return set
}

// focusRule decides which known requirements a run demands coverage for.
type focusRule struct {
	files      map[string]bool
	idPatterns []string
}

func (f focusRule) includes(file inputFile, id domain.RequirementID) bool {
	if len(f.files) == 0 && len(f.idPatterns) == 0 {
		return true
	}
	if f.files[file.absolute] {
		return true
	}
	for _, pattern := range f.idPatterns {
		if matched, _ := doublestar.Match(pattern, string(id)); matched {
			return true
		}
	}
	return false
}
