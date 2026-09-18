package analysis

import (
	"strings"
	"unicode"
)

// normalizeIdentity folds whitespace and underscores, which some runners
// substitute for each other in test names, into single spaces.
func normalizeIdentity(text string) string {
	fields := strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) || r == '_' })
	return strings.Join(fields, " ")
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// containsLabel reports whether label occurs in identity without cutting a word in two.
func containsLabel(identity, label string) bool {
	if label == "" {
		return true
	}
	labelRunes := []rune(label)
	guardStart := isWordRune(labelRunes[0])
	guardEnd := isWordRune(labelRunes[len(labelRunes)-1])
	for offset := 0; ; {
		index := strings.Index(identity[offset:], label)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(label)
		if boundaryBefore(identity, start, guardStart) && boundaryAfter(identity, end, guardEnd) {
			return true
		}
		offset = start + 1
	}
}

func boundaryBefore(text string, index int, guarded bool) bool {
	if !guarded || index == 0 {
		return true
	}
	previous := []rune(text[:index])
	return !isWordRune(previous[len(previous)-1])
}

func boundaryAfter(text string, index int, guarded bool) bool {
	if !guarded || index >= len(text) {
		return true
	}
	next := []rune(text[index:])
	return !isWordRune(next[0])
}
