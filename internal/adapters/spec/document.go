// Package spec loads requirements from Markdown and YAML specifications.
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	initialLineBuffer = 64 * 1024
	maxLineLength     = 4 * 1024 * 1024
)

// Document is what one spec file declares.
type Document struct {
	Requirements []domain.Requirement
	Problems     []domain.SpecProblem
}

// Format is a spec file format.
type Format uint8

const (
	FormatMarkdown Format = iota
	FormatYAML
)

var formatsByExtension = map[string]Format{
	".md":       FormatMarkdown,
	".markdown": FormatMarkdown,
	".yaml":     FormatYAML,
	".yml":      FormatYAML,
}

// FormatOf recognises a spec file by its extension.
func FormatOf(path string) (Format, bool) {
	format, known := formatsByExtension[strings.ToLower(filepath.Ext(path))]
	return format, known
}

// Extensions lists the file extensions recognised as specs.
func Extensions() []string {
	return []string{".md", ".markdown", ".yaml", ".yml"}
}

// Loader reads spec files of either format into the same model.
type Loader struct {
	markdown MarkdownParser
	yaml     YAMLParser
}

// NewLoader builds a loader for the project's ID convention.
func NewLoader(ids domain.IDPattern) Loader {
	return Loader{markdown: NewMarkdownParser(ids), yaml: YAMLParser{IDs: ids}}
}

// LoadFile reads one spec file. displayPath is the path recorded in locations.
func (l Loader) LoadFile(path, displayPath string) (Document, error) {
	format, known := FormatOf(path)
	if !known {
		return Document{}, fmt.Errorf("%s: unsupported spec format (expected one of %v)", displayPath, Extensions())
	}
	file, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer file.Close()
	if format == FormatYAML {
		return l.yaml.Parse(file, displayPath)
	}
	return l.markdown.Parse(file, displayPath)
}
