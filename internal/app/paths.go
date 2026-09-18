package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// inputFile is a resolved input with the path shown in reports.
type inputFile struct {
	path        string
	displayPath string
	absolute    string
}

// expand resolves files, directories and globs to the matching files, sorted
// by display path. Directories contribute their files with a given extension.
// A pattern that matches nothing is an error: an absent input is never a verdict.
func expand(kind string, patterns, extensions []string) ([]inputFile, error) {
	seen := map[string]bool{}
	var files []inputFile
	add := func(path string) error {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if !seen[absolute] {
			seen[absolute] = true
			files = append(files, inputFile{path: path, displayPath: filepath.ToSlash(filepath.Clean(path)), absolute: absolute})
		}
		return nil
	}
	for _, pattern := range patterns {
		matches, err := resolve(pattern, extensions)
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", kind, pattern, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s %q matches no file", kind, pattern)
		}
		for _, match := range matches {
			if err := add(match); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].displayPath < files[j].displayPath })
	return files, nil
}

func resolve(pattern string, extensions []string) ([]string, error) {
	info, err := os.Stat(pattern)
	switch {
	case err == nil && info.IsDir():
		return filesUnder(pattern, extensions)
	case err == nil:
		return []string{pattern}, nil
	}
	matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly())
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func filesUnder(directory string, extensions []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() && hasExtension(path, extensions) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func hasExtension(path string, extensions []string) bool {
	actual := strings.ToLower(filepath.Ext(path))
	for _, extension := range extensions {
		if actual == extension {
			return true
		}
	}
	return false
}

func displayPaths(files []inputFile) []string {
	paths := make([]string, len(files))
	for index, file := range files {
		paths[index] = file.displayPath
	}
	return paths
}
