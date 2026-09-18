// Package scan finds binding tags in test source files.
package scan

import "github.com/RachidChabane/shallnot/internal/domain"

// SourceFile is a text file under a test root.
type SourceFile struct {
	DisplayPath string
	Content     string
}

// TagExtractor is a strategy that recognises one way of writing binding tags.
type TagExtractor interface {
	// Handles reports whether the extractor applies to a file.
	Handles(displayPath string) bool
	// Extract returns the tags of a file it handles.
	Extract(file SourceFile) []domain.SourceTag
}

// DefaultExtractors returns the extractor of every supported tag syntax.
func DefaultExtractors(grammar domain.TagGrammar) []TagExtractor {
	return []TagExtractor{
		BracketExtractor{Grammar: grammar},
		PytestExtractor{Grammar: grammar},
	}
}

// lineIndex converts byte offsets of a text into 1-based line numbers.
type lineIndex struct {
	starts []int
}

func newLineIndex(content string) lineIndex {
	index := lineIndex{starts: []int{0}}
	for offset, char := range content {
		if char == '\n' {
			index.starts = append(index.starts, offset+1)
		}
	}
	return index
}

func (l lineIndex) lineOf(offset int) int {
	low, high := 0, len(l.starts)-1
	for low < high {
		middle := (low + high + 1) / 2
		if l.starts[middle] <= offset {
			low = middle
		} else {
			high = middle - 1
		}
	}
	return low + 1
}

func (l lineIndex) lineStart(offset int) int { return l.starts[l.lineOf(offset)-1] }
