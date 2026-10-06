package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ExampleMarker introduces an inline example in a statement.
const ExampleMarker = "Example:"

// Tokens of the example grammar.
const (
	callOpen        = '('
	callClose       = ')'
	argumentSplit   = ','
	argumentBinding = '='
	quote           = '"'
	escape          = '\\'
	outcomeArrow    = "=>"
	raisesKeyword   = "raises"
	trueKeyword     = "true"
	falseKeyword    = "false"
	nullKeyword     = "null"

	blanks         = " \t"
	wordTerminator = blanks + string(argumentSplit) + string(callClose) + string(argumentBinding)

	datetimeLayout        = "2006-01-02T15:04"
	datetimeSecondsLayout = ":05"
	datetimeZoneLayout    = "Z07:00"
)

var quotedEscapes = map[byte]byte{quote: quote, escape: escape, 'n': '\n', 't': '\t'}

// LiteralType is the type of a value written in an example.
type LiteralType uint8

const (
	LiteralDatetime LiteralType = iota
	LiteralDecimal
	LiteralInteger
	LiteralBoolean
	LiteralNull
	LiteralString
	// LiteralName types the exception name of an outcome that raises.
	LiteralName
)

var literalTypeNames = []string{"datetime", "decimal", "integer", "boolean", "null", "string", "name"}

const literalTypeKind = "literal type"

func (l LiteralType) String() string { return enumString(literalTypeKind, literalTypeNames, l) }
func (l LiteralType) MarshalText() ([]byte, error) {
	return enumText(literalTypeKind, literalTypeNames, l)
}
func (l *LiteralType) UnmarshalText(text []byte) (err error) {
	*l, err = parseEnum[LiteralType](literalTypeKind, literalTypeNames, string(text))
	return err
}

// LiteralTypeNames lists the wire names of every literal type.
func LiteralTypeNames() []string { return append([]string(nil), literalTypeNames...) }

// OutcomeKind says whether an example expects a value or an exception.
type OutcomeKind uint8

const (
	OutcomeReturns OutcomeKind = iota
	OutcomeRaises
)

var outcomeKindNames = []string{"returns", "raises"}

const outcomeKindKind = "outcome kind"

func (k OutcomeKind) String() string { return enumString(outcomeKindKind, outcomeKindNames, k) }
func (k OutcomeKind) MarshalText() ([]byte, error) {
	return enumText(outcomeKindKind, outcomeKindNames, k)
}
func (k *OutcomeKind) UnmarshalText(text []byte) (err error) {
	*k, err = parseEnum[OutcomeKind](outcomeKindKind, outcomeKindNames, string(text))
	return err
}

// OutcomeKindNames lists the wire names of every outcome kind.
func OutcomeKindNames() []string { return append([]string(nil), outcomeKindNames...) }

// Literal is a typed value. Value is its text: as written for a datetime, a
// decimal, an integer, a boolean and null; unquoted and unescaped for a string.
type Literal struct {
	Type  LiteralType
	Value string
}

// Argument is one named argument of an example's call.
type Argument struct {
	Name  string
	Value Literal
}

// ExpectedOutcome is what an example expects: a returned value, or the name of a raised exception.
type ExpectedOutcome struct {
	Kind  OutcomeKind
	Value Literal
}

// Example is one inline example of a statement: a call and what it must give.
type Example struct {
	// Index is the 1-based position of the example among the markers of the statement.
	Index     int
	Operation string
	Args      []Argument
	Expect    ExpectedOutcome
}

// MalformedExample is a segment after a marker that the grammar does not accept.
type MalformedExample struct {
	Index  int
	Text   string
	Reason string
}

// ExampleSet is what a statement holds: its prose and its examples.
type ExampleSet struct {
	// Prose is the statement before the first marker.
	Prose     string
	Examples  []Example
	Malformed []MalformedExample
}

// DetachedExample is a line that reads as an example but that no statement absorbed.
type DetachedExample struct {
	Location Location
	Text     string
}

// markerPattern finds a marker, with its optional list bullet, at the start of
// the statement or after whitespace.
var markerPattern = regexp.MustCompile(`(?:^|\s)(?:[-*+]\s+)?` + regexp.QuoteMeta(ExampleMarker))

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	datetimePattern   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})?$`)
	decimalPattern    = regexp.MustCompile(`^-?\d+\.\d+$`)
	integerPattern    = regexp.MustCompile(`^-?\d+$`)
)

// ParseExamples splits a joined statement at each example marker. The text
// before the first marker is the prose; every segment after a marker is an
// example, or a malformed example when the grammar rejects it.
func ParseExamples(statement string) ExampleSet {
	matches := markerPattern.FindAllStringIndex(statement, -1)
	if len(matches) == 0 {
		return ExampleSet{Prose: strings.TrimSpace(statement)}
	}
	set := ExampleSet{Prose: strings.TrimSpace(statement[:matches[0][0]])}
	for position, match := range matches {
		end := len(statement)
		if position+1 < len(matches) {
			end = matches[position+1][0]
		}
		segment := strings.TrimSpace(statement[match[1]:end])
		index := position + 1
		example, err := parseExample(segment)
		if err != nil {
			set.Malformed = append(set.Malformed, MalformedExample{Index: index, Text: segment, Reason: err.Error()})
			continue
		}
		example.Index = index
		set.Examples = append(set.Examples, example)
	}
	return set
}

func parseExample(segment string) (Example, error) {
	if segment == "" {
		return Example{}, errors.New("the example is empty")
	}
	scanner := &exampleScanner{text: segment}
	example, err := scanner.call()
	if err != nil {
		return Example{}, err
	}
	scanner.skipSpace()
	if !scanner.consume(outcomeArrow) {
		return Example{}, fmt.Errorf("expected %q after the call", outcomeArrow)
	}
	scanner.skipSpace()
	if example.Expect, err = scanner.outcome(); err != nil {
		return Example{}, err
	}
	scanner.skipSpace()
	if !scanner.atEnd() {
		return Example{}, fmt.Errorf("unexpected text after the outcome: %q", scanner.rest())
	}
	return example, nil
}

type exampleScanner struct {
	text     string
	position int
}

func (s *exampleScanner) atEnd() bool  { return s.position >= len(s.text) }
func (s *exampleScanner) rest() string { return s.text[s.position:] }

func (s *exampleScanner) skipSpace() {
	for !s.atEnd() && strings.IndexByte(blanks, s.text[s.position]) >= 0 {
		s.position++
	}
}

func (s *exampleScanner) consume(token string) bool {
	if strings.HasPrefix(s.rest(), token) {
		s.position += len(token)
		return true
	}
	return false
}

func (s *exampleScanner) consumeByte(expected byte) bool {
	if !s.atEnd() && s.text[s.position] == expected {
		s.position++
		return true
	}
	return false
}

// word reads up to the next space, comma, closing parenthesis or equal sign.
func (s *exampleScanner) word() string {
	start := s.position
	for !s.atEnd() && strings.IndexByte(wordTerminator, s.text[s.position]) < 0 {
		s.position++
	}
	return s.text[start:s.position]
}

func (s *exampleScanner) identifier(role string) (string, error) {
	name := s.word()
	if !identifierPattern.MatchString(name) {
		return "", fmt.Errorf("expected %s, got %q", role, name)
	}
	return name, nil
}

func (s *exampleScanner) call() (Example, error) {
	var example Example
	var err error
	start := s.position
	for !s.atEnd() && s.text[s.position] != callOpen && strings.IndexByte(blanks, s.text[s.position]) < 0 {
		s.position++
	}
	example.Operation = s.text[start:s.position]
	if !identifierPattern.MatchString(example.Operation) {
		return Example{}, fmt.Errorf("expected an operation name, got %q", example.Operation)
	}
	s.skipSpace()
	if !s.consumeByte(callOpen) {
		return Example{}, fmt.Errorf("expected %q after the operation %s", string(callOpen), example.Operation)
	}
	s.skipSpace()
	if s.consumeByte(callClose) {
		return example, nil
	}
	seen := map[string]bool{}
	for {
		var argument Argument
		if argument, err = s.argument(); err != nil {
			return Example{}, err
		}
		if seen[argument.Name] {
			return Example{}, fmt.Errorf("the argument %s is given twice", argument.Name)
		}
		seen[argument.Name] = true
		example.Args = append(example.Args, argument)
		s.skipSpace()
		if s.consumeByte(callClose) {
			return example, nil
		}
		if !s.consumeByte(argumentSplit) {
			return Example{}, fmt.Errorf("expected %q or %q after the argument %s", string(argumentSplit), string(callClose), argument.Name)
		}
		s.skipSpace()
	}
}

func (s *exampleScanner) argument() (Argument, error) {
	name, err := s.identifier("an argument name")
	if err != nil {
		return Argument{}, err
	}
	s.skipSpace()
	if !s.consumeByte(argumentBinding) {
		return Argument{}, fmt.Errorf("expected %q after the argument name %s", string(argumentBinding), name)
	}
	s.skipSpace()
	value, err := s.literal()
	return Argument{Name: name, Value: value}, err
}

func (s *exampleScanner) outcome() (ExpectedOutcome, error) {
	if strings.HasPrefix(s.rest(), raisesKeyword) {
		after := s.rest()[len(raisesKeyword):]
		if after == "" || strings.IndexByte(blanks, after[0]) >= 0 {
			s.position += len(raisesKeyword)
			s.skipSpace()
			name, err := s.identifier("an exception name after " + raisesKeyword)
			return ExpectedOutcome{Kind: OutcomeRaises, Value: Literal{Type: LiteralName, Value: name}}, err
		}
	}
	value, err := s.literal()
	return ExpectedOutcome{Kind: OutcomeReturns, Value: value}, err
}

func (s *exampleScanner) literal() (Literal, error) {
	if s.consumeByte(quote) {
		return s.quoted()
	}
	token := s.word()
	switch {
	case token == "":
		return Literal{}, errors.New("expected a value")
	case token == trueKeyword || token == falseKeyword:
		return Literal{Type: LiteralBoolean, Value: token}, nil
	case token == nullKeyword:
		return Literal{Type: LiteralNull, Value: token}, nil
	case integerPattern.MatchString(token):
		return Literal{Type: LiteralInteger, Value: token}, nil
	case decimalPattern.MatchString(token):
		return Literal{Type: LiteralDecimal, Value: token}, nil
	case datetimePattern.MatchString(token):
		if err := validateDatetime(token); err != nil {
			return Literal{}, err
		}
		return Literal{Type: LiteralDatetime, Value: token}, nil
	}
	return Literal{}, fmt.Errorf("%q is not a datetime, decimal, integer, true, false, null or quoted string", token)
}

func (s *exampleScanner) quoted() (Literal, error) {
	var value strings.Builder
	for !s.atEnd() {
		character := s.text[s.position]
		s.position++
		switch character {
		case quote:
			return Literal{Type: LiteralString, Value: value.String()}, nil
		case escape:
			if s.atEnd() {
				return Literal{}, errors.New("the string ends with a lone backslash")
			}
			decoded, known := quotedEscapes[s.text[s.position]]
			if !known {
				return Literal{}, fmt.Errorf("unknown escape \\%c in a string", s.text[s.position])
			}
			s.position++
			value.WriteByte(decoded)
		default:
			value.WriteByte(character)
		}
	}
	return Literal{}, errors.New("the string is not closed")
}

// validateDatetime checks the calendar and clock values of a token that has the shape of an ISO-8601 datetime.
func validateDatetime(token string) error {
	groups := datetimePattern.FindStringSubmatch(token)
	layout := datetimeLayout
	if groups[1] != "" {
		layout += datetimeSecondsLayout
	}
	if groups[3] != "" {
		layout += datetimeZoneLayout
	}
	if _, err := time.Parse(layout, token); err != nil {
		return fmt.Errorf("%q is not a valid ISO-8601 datetime", token)
	}
	return nil
}
