package scan

import (
	"regexp"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

// placeholder matches the parts of a test title that the runner substitutes:
// printf verbs, ${expr}, $name and {index} templates.
var placeholder = regexp.MustCompile(`%[a-zA-Z#%]|\$\{[^}]*\}|\$[A-Za-z_][\w.]*|\{[\w.]*\}`)

const stringDelimiters = "\"'`"

// BracketExtractor finds bracket tags in any text file.
type BracketExtractor struct {
	Grammar domain.TagGrammar
}

// Handles accepts every file: the bracket tag is language-agnostic.
func (BracketExtractor) Handles(string) bool { return true }

// Extract returns the bracket tags of a file, labelled with the test title they sit in.
func (e BracketExtractor) Extract(file SourceFile) []domain.SourceTag {
	var tags []domain.SourceTag
	lines := newLineIndex(file.Content)
	for _, match := range e.Grammar.FindBracketTags(file.Content) {
		lineStart := lines.lineStart(match.Start)
		lineEnd := lineStart + len(lineAt(file.Content, lineStart))
		line := file.Content[lineStart:lineEnd]
		tags = append(tags, domain.SourceTag{
			Location:  domain.Location{File: file.DisplayPath, Line: lines.lineOf(match.Start)},
			Label:     titleAround(line, match.Start-lineStart, match.End-lineStart),
			Refs:      match.Tag.Refs,
			Malformed: match.Tag.Malformed,
		})
	}
	return tags
}

func lineAt(content string, start int) string {
	if end := strings.IndexByte(content[start:], '\n'); end >= 0 {
		return content[start : start+end]
	}
	return content[start:]
}

// titleAround returns the static part of the string literal that encloses
// line[start:end], or "" when the tag is not inside a string literal.
func titleAround(line string, start, end int) string {
	literalStart, literalEnd, enclosed := enclosingLiteral(line, start, end)
	if !enclosed {
		return ""
	}
	fragmentStart, fragmentEnd := literalStart, literalEnd
	for _, span := range placeholder.FindAllStringIndex(line[literalStart:literalEnd], -1) {
		spanStart, spanEnd := literalStart+span[0], literalStart+span[1]
		if spanEnd <= start && spanEnd > fragmentStart {
			fragmentStart = spanEnd
		}
		if spanStart >= end && spanStart < fragmentEnd {
			fragmentEnd = spanStart
		}
	}
	return strings.TrimSpace(line[fragmentStart:fragmentEnd])
}

// enclosingLiteral finds the quoted span of a line that contains [start, end).
func enclosingLiteral(line string, start, end int) (int, int, bool) {
	var delimiter byte
	literalStart := 0
	for index := 0; index < len(line); index++ {
		char := line[index]
		switch {
		case delimiter == 0 && strings.IndexByte(stringDelimiters, char) >= 0:
			delimiter, literalStart = char, index+1
		case delimiter != 0 && char == '\\':
			index++
		case delimiter != 0 && char == delimiter:
			if literalStart <= start && end <= index {
				return literalStart, index, true
			}
			delimiter = 0
		}
	}
	return 0, 0, false
}
