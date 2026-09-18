// Package report renders a completed run for its different readers.
package report

import (
	"fmt"
	"io"

	"github.com/RachidChabane/shallnot/internal/app"
)

// Reporter renders a run in one format.
type Reporter interface {
	Write(writer io.Writer, outcome app.Outcome) error
}

// Format names an output format.
type Format uint8

const (
	FormatTerminal Format = iota
	FormatJSON
	FormatMarkdown
	FormatGitHub
)

var formatNames = []string{"terminal", "json", "markdown", "github"}

func (f Format) String() string { return formatNames[f] }

// FormatNames lists the wire names of every format.
func FormatNames() []string { return append([]string(nil), formatNames...) }

// ParseFormat resolves a format name.
func ParseFormat(text string) (Format, error) {
	for index, name := range formatNames {
		if name == text {
			return Format(index), nil
		}
	}
	return 0, fmt.Errorf("unknown format %q (expected one of %v)", text, formatNames)
}

// For returns the reporter of a format.
func For(format Format) Reporter {
	switch format {
	case FormatJSON:
		return JSONReporter{}
	case FormatMarkdown:
		return MarkdownReporter{}
	case FormatGitHub:
		return GitHubReporter{}
	default:
		return TerminalReporter{}
	}
}
