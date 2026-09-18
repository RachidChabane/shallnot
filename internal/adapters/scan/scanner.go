package scan

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	// maxSourceFileSize bounds the files read by the scan; larger files are generated artefacts.
	maxSourceFileSize = 2 * 1024 * 1024
	binaryProbeSize   = 8 * 1024
	// directoryProbe stands for any file of a directory when testing exclude globs against it.
	directoryProbe = "/_"
)

// DefaultExcludes are the directories never scanned for tags.
var DefaultExcludes = []string{
	"**/.git/**", "**/node_modules/**", "**/target/**", "**/build/**", "**/dist/**",
	"**/.venv/**", "**/venv/**", "**/__pycache__/**", "**/.gradle/**",
	"**/.pytest_cache/**", "**/.mypy_cache/**", "**/coverage/**",
}

// Scanner walks test roots and applies every tag extractor to each text file.
type Scanner struct {
	Extractors []TagExtractor
	Excludes   []string
	// Skip holds the absolute paths of files that are inputs of another kind (specs, results).
	Skip map[string]bool
}

// ScanRoot returns the tags found under one test root, ordered by location.
func (s Scanner) ScanRoot(root string) ([]domain.SourceTag, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: a test root must be a directory", root)
	}
	var tags []domain.SourceTag
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		displayPath := filepath.ToSlash(filepath.Clean(path))
		if entry.IsDir() {
			if s.excluded(displayPath + directoryProbe) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || s.excluded(displayPath) || s.skipped(path) {
			return nil
		}
		found, err := s.scanFile(path, displayPath)
		tags = append(tags, found...)
		return err
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].Location.File != tags[j].Location.File {
			return tags[i].Location.File < tags[j].Location.File
		}
		return tags[i].Location.Line < tags[j].Location.Line
	})
	return tags, nil
}

func (s Scanner) excluded(displayPath string) bool {
	for _, pattern := range s.Excludes {
		if matched, _ := doublestar.Match(pattern, displayPath); matched {
			return true
		}
	}
	return false
}

func (s Scanner) skipped(path string) bool {
	absolute, err := filepath.Abs(path)
	return err == nil && s.Skip[absolute]
}

func (s Scanner) scanFile(path, displayPath string) ([]domain.SourceTag, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxSourceFileSize {
		return nil, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	probe := content[:min(len(content), binaryProbeSize)]
	if bytes.IndexByte(probe, 0) >= 0 {
		return nil, nil
	}
	file := SourceFile{DisplayPath: displayPath, Content: string(content)}
	var tags []domain.SourceTag
	for _, extractor := range s.Extractors {
		if extractor.Handles(displayPath) {
			tags = append(tags, extractor.Extract(file)...)
		}
	}
	return tags, nil
}
