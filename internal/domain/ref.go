package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// RevisionSeparator separates a requirement ID from its revision in a reference.
const RevisionSeparator = "~"

// FirstRevision is the revision of a requirement that declares none.
const FirstRevision Revision = 1

// DefaultIDPattern matches IDs such as REQ-042 and ABC-101.AC3.
const DefaultIDPattern = `[A-Z][A-Z0-9]*-[0-9]+(?:\.[A-Za-z0-9]+)*`

// RequirementID is the stable identifier of a requirement.
type RequirementID string

// Revision counts the meaning-changing edits of a requirement, starting at 1.
type Revision int

// Ref cites one requirement at one revision.
type Ref struct {
	ID       RequirementID
	Revision Revision
}

func (r Ref) String() string {
	return string(r.ID) + RevisionSeparator + strconv.Itoa(int(r.Revision))
}

// IDPattern is the project's convention for requirement IDs.
type IDPattern struct {
	source string
	whole  *regexp.Regexp
}

// NewIDPattern compiles a regular expression that a whole ID must match.
func NewIDPattern(source string) (IDPattern, error) {
	whole, err := regexp.Compile(`^(?:` + source + `)$`)
	if err != nil {
		return IDPattern{}, fmt.Errorf("invalid id pattern %q: %w", source, err)
	}
	for _, reserved := range []string{RevisionSeparator, ",", "]", " "} {
		if whole.MatchString(reserved) {
			return IDPattern{}, fmt.Errorf("id pattern %q must not match the reserved text %q", source, reserved)
		}
	}
	return IDPattern{source: source, whole: whole}, nil
}

// Source returns the regular expression the pattern was built from.
func (p IDPattern) Source() string { return p.source }

// Matches reports whether text is a well-formed requirement ID.
func (p IDPattern) Matches(text string) bool {
	return p.whole != nil && p.whole.MatchString(text) && !strings.ContainsAny(text, RevisionSeparator+",] \t")
}

// ParseRevision reads a revision number, which is a positive integer.
func ParseRevision(text string) (Revision, error) {
	number, err := strconv.Atoi(text)
	if err != nil || number < int(FirstRevision) || strconv.Itoa(number) != text {
		return 0, fmt.Errorf("revision %q is not a positive integer", text)
	}
	return Revision(number), nil
}

// ParseRef reads a reference of the form ID~REVISION.
func (p IDPattern) ParseRef(text string) (Ref, error) {
	id, revisionText, found := strings.Cut(text, RevisionSeparator)
	if !found {
		return Ref{}, fmt.Errorf("%q cites no revision (expected ID%sREVISION)", text, RevisionSeparator)
	}
	if !p.Matches(id) {
		return Ref{}, fmt.Errorf("%q does not match the id pattern %s", id, p.source)
	}
	revision, err := ParseRevision(revisionText)
	if err != nil {
		return Ref{}, err
	}
	return Ref{ID: RequirementID(id), Revision: revision}, nil
}
