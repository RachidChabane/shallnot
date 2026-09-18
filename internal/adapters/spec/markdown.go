package spec

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
)

const (
	linePrefix     = `^\s*(?:(#{1,6})\s+|(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?)?`
	emphasis       = "(?:\\*\\*|__|`)?"
	revisionSuffix = `(?:(` + domain.RevisionSeparator + `)([^\s:*_` + "`" + `]*))?`
)

var (
	fence            = regexp.MustCompile("^\\s*(```|~~~)")
	heading          = regexp.MustCompile(`^\s*#{1,6}\s`)
	nonTestableLine  = regexp.MustCompile(`(?i)^\s*(?:[-*+]\s+)?(?:\*\*|__)?non[-_ ]testable(?:\*\*|__)?\s*:?\s*(?:\*\*|__)?\s*(.*)$`)
	trailingEmphasis = regexp.MustCompile(`^(?:\*\*|__)\s*`)
)

// MarkdownParser finds requirement declarations embedded in Markdown prose.
type MarkdownParser struct {
	declaration *regexp.Regexp
}

// NewMarkdownParser builds a parser for the project's ID convention.
func NewMarkdownParser(ids domain.IDPattern) MarkdownParser {
	return MarkdownParser{declaration: regexp.MustCompile(
		linePrefix + emphasis + `(` + ids.Source() + `)` + revisionSuffix + emphasis + `\s*:` + emphasis + `\s*(.*)$`)}
}

// Declaration capture groups.
const (
	groupHeading = iota + 1
	groupID
	groupSeparator
	groupRevision
	groupRest
)

type markdownBlock struct {
	requirement     domain.Requirement
	isHeading       bool
	statement       []string
	statementEnded  bool
	justification   []string
	inJustification bool
}

// Parse reads the requirements of one Markdown document.
func (p MarkdownParser) Parse(reader io.Reader, displayPath string) (Document, error) {
	var document Document
	var current *markdownBlock
	flush := func() {
		if current == nil {
			return
		}
		requirement := current.finish()
		current = nil
		if requirement.Statement == "" {
			document.Problems = append(document.Problems, domain.SpecProblem{
				Location: requirement.Location,
				Message:  fmt.Sprintf("requirement %s has no statement", requirement.ID),
			})
			return
		}
		document.Requirements = append(document.Requirements, requirement)
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, initialLineBuffer), maxLineLength)
	inFence := false
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		if fence.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if groups := p.declaration.FindStringSubmatch(line); groups != nil {
			flush()
			location := domain.Location{File: displayPath, Line: lineNumber}
			block, problem := newMarkdownBlock(groups, location)
			if problem != nil {
				document.Problems = append(document.Problems, *problem)
				continue
			}
			current = block
			continue
		}
		if heading.MatchString(line) {
			flush()
			continue
		}
		if current != nil {
			current.absorb(line)
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return Document{}, fmt.Errorf("%s: %w", displayPath, err)
	}
	return document, nil
}

func newMarkdownBlock(groups []string, location domain.Location) (*markdownBlock, *domain.SpecProblem) {
	revision := domain.FirstRevision
	if groups[groupSeparator] != "" {
		parsed, err := domain.ParseRevision(groups[groupRevision])
		if err != nil {
			return nil, &domain.SpecProblem{
				Location: location,
				Message:  fmt.Sprintf("requirement %s: %v", groups[groupID], err),
			}
		}
		revision = parsed
	}
	block := &markdownBlock{
		isHeading: groups[groupHeading] != "",
		requirement: domain.Requirement{
			ID:       domain.RequirementID(groups[groupID]),
			Revision: revision,
			Location: location,
		},
	}
	rest := strings.TrimSpace(trailingEmphasis.ReplaceAllString(groups[groupRest], ""))
	if block.isHeading {
		block.requirement.Title = strings.TrimSpace(strings.TrimRight(rest, "# "))
	} else if rest != "" {
		block.statement = append(block.statement, rest)
	}
	return block, nil
}

func (b *markdownBlock) absorb(line string) {
	if groups := nonTestableLine.FindStringSubmatch(line); groups != nil {
		b.requirement.Status = domain.StatusNonTestable
		b.justification = strings.Fields(groups[1])
		b.statementEnded, b.inJustification = true, true
		return
	}
	text := strings.TrimSpace(line)
	switch {
	case text == "":
		b.statementEnded = b.statementEnded || len(b.statement) > 0
		b.inJustification = false
	case b.inJustification:
		b.justification = append(b.justification, text)
	case !b.statementEnded:
		b.statement = append(b.statement, text)
	}
}

func (b *markdownBlock) finish() domain.Requirement {
	b.requirement.Statement = strings.Join(b.statement, " ")
	b.requirement.Justification = strings.Join(b.justification, " ")
	if b.requirement.Statement == "" {
		b.requirement.Statement = b.requirement.Title
	}
	return b.requirement
}
