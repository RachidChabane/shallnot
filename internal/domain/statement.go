package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// StatementCanonicalization identifies the canonical form of a requirement's
// meaning-bearing text, and so the meaning of every statement digest. Changing
// the form changes this identifier.
const StatementCanonicalization = "c1"

const (
	canonicalFieldSeparator = "\n"
	replacementCharacter    = "�"
	spaceCharacter          = " "
	crlf                    = "\r\n"
	carriageReturn          = "\r"
	lineFeed                = "\n"
)

// CanonicalText is the canonical form c1 of one text: invalid UTF-8 replaced,
// NFC, LF line endings, continuation lines joined, whitespace collapsed to one
// space, trimmed.
func CanonicalText(text string) string {
	text = strings.ToValidUTF8(text, replacementCharacter)
	text = norm.NFC.String(text)
	text = strings.ReplaceAll(text, crlf, lineFeed)
	text = strings.ReplaceAll(text, carriageReturn, lineFeed)
	text = strings.ReplaceAll(text, lineFeed, spaceCharacter)
	return strings.Join(strings.FieldsFunc(text, unicode.IsSpace), spaceCharacter)
}

// CanonicalForm is the text a statement digest is computed over: the canonical
// title, statement, status and justification, one per line in that order.
// Canonical texts hold no line feed, so the join is unambiguous.
func (r Requirement) CanonicalForm() string {
	return strings.Join([]string{
		CanonicalText(r.Title),
		CanonicalText(r.Statement),
		r.Status.String(),
		CanonicalText(r.Justification),
	}, canonicalFieldSeparator)
}

// StatementDigest is the lowercase hexadecimal SHA-256 of the canonical form.
func (r Requirement) StatementDigest() string {
	sum := sha256.Sum256([]byte(r.CanonicalForm()))
	return hex.EncodeToString(sum[:])
}
