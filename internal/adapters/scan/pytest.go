package scan

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	pythonExtension    = ".py"
	argumentSeparators = ", \t\r\n"
)

var (
	pytestMarker       = regexp.MustCompile(`pytest\.mark\.` + domain.TagKeyword + `\(([^)]*)\)`)
	pytestProperty     = regexp.MustCompile(`record_property\(\s*["']` + domain.TagKeyword + `["']\s*,([^)]*)\)`)
	pythonLiteral      = regexp.MustCompile(`"([^"]*)"|'([^']*)'`)
	pythonDefinition   = regexp.MustCompile(`(?m)^[ \t]*(?:async[ \t]+)?(?:def|class)[ \t]+(\w+)`)
	pythonModuleMarker = regexp.MustCompile(`^\s*pytestmark\s*=`)
)

// PytestExtractor finds `pytest.mark.verifies("ID~REV")` markers and
// `record_property("verifies", "ID~REV")` calls in Python files.
type PytestExtractor struct {
	Grammar domain.TagGrammar
}

// Handles accepts Python files.
func (PytestExtractor) Handles(displayPath string) bool {
	return strings.EqualFold(filepath.Ext(displayPath), pythonExtension)
}

// Extract returns the pytest tags of a file, labelled with the test they decorate.
func (e PytestExtractor) Extract(file SourceFile) []domain.SourceTag {
	lines := newLineIndex(file.Content)
	var tags []domain.SourceTag
	for _, match := range pytestMarker.FindAllStringSubmatchIndex(file.Content, -1) {
		if commentedOut(file.Content, lines, match[0]) {
			continue
		}
		tags = append(tags, e.tag(file, lines, match, e.markerLabel(file, lines, match)))
	}
	for _, match := range pytestProperty.FindAllStringSubmatchIndex(file.Content, -1) {
		if commentedOut(file.Content, lines, match[0]) {
			continue
		}
		tags = append(tags, e.tag(file, lines, match, definitionBefore(file.Content, match[0])))
	}
	return tags
}

func (e PytestExtractor) tag(file SourceFile, lines lineIndex, match []int, label string) domain.SourceTag {
	raw := file.Content[match[0]:match[1]]
	arguments := file.Content[match[2]:match[3]]
	tag := domain.SourceTag{
		Location: domain.Location{File: file.DisplayPath, Line: lines.lineOf(match[0])},
		Label:    label,
	}
	literals := pythonLiteral.FindAllStringSubmatch(arguments, -1)
	nonLiteral := strings.Trim(pythonLiteral.ReplaceAllString(arguments, ""), argumentSeparators)
	if len(literals) == 0 || nonLiteral != "" {
		tag.Malformed = append(tag.Malformed, domain.MalformedTag{Raw: raw, Reason: "the requirements must be given as string literals"})
		return tag
	}
	for _, literal := range literals {
		parsed := e.Grammar.ParseRefList(raw, literal[1]+literal[2])
		tag.Refs = append(tag.Refs, parsed.Refs...)
		tag.Malformed = append(tag.Malformed, parsed.Malformed...)
	}
	return tag
}

// markerLabel names what a marker decorates: the module for `pytestmark`,
// otherwise the next function or class.
func (e PytestExtractor) markerLabel(file SourceFile, lines lineIndex, match []int) string {
	lineStart := lines.lineStart(match[0])
	if pythonModuleMarker.MatchString(file.Content[lineStart:match[0]]) {
		return strings.TrimSuffix(filepath.Base(file.DisplayPath), filepath.Ext(file.DisplayPath))
	}
	if next := pythonDefinition.FindStringSubmatch(file.Content[match[1]:]); next != nil {
		return next[1]
	}
	return ""
}

func definitionBefore(content string, offset int) string {
	definitions := pythonDefinition.FindAllStringSubmatch(content[:offset], -1)
	if len(definitions) == 0 {
		return ""
	}
	return definitions[len(definitions)-1][1]
}

func commentedOut(content string, lines lineIndex, offset int) bool {
	return strings.Contains(content[lines.lineStart(offset):offset], "#")
}
