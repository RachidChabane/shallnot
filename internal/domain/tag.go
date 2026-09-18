package domain

import (
	"regexp"
	"strings"
)

// TagKeyword opens a binding tag and names the results property that carries one.
const TagKeyword = "verifies"

// refSeparator separates the references of one tag.
const refSeparator = ","

// refPadding is trimmed around each reference. Runners that rewrite spaces in
// test names (go test) turn them into underscores.
const refPadding = " \t_"

// bracketTag matches the keyword and a reference list between square brackets, on one line.
var bracketTag = regexp.MustCompile(`\[` + TagKeyword + `(?:[\s_:]+([^\]\n]*))?\]`)

// TagGrammar reads binding tags whose IDs follow the project's pattern.
type TagGrammar struct {
	IDs IDPattern
}

// BracketTagMatch is a bracket tag found in a text, with its byte offsets.
type BracketTagMatch struct {
	Tag   Tag
	Start int
	End   int
}

// FindBracketTags returns every bracket tag of a text, in order.
func (g TagGrammar) FindBracketTags(text string) []BracketTagMatch {
	var matches []BracketTagMatch
	for _, indexes := range bracketTag.FindAllStringSubmatchIndex(text, -1) {
		raw := text[indexes[0]:indexes[1]]
		list := ""
		if indexes[2] >= 0 {
			list = text[indexes[2]:indexes[3]]
		}
		matches = append(matches, BracketTagMatch{Tag: g.ParseRefList(raw, list), Start: indexes[0], End: indexes[1]})
	}
	return matches
}

// ParseRefList reads a comma-separated list of references. raw is the text reported when it is malformed.
func (g TagGrammar) ParseRefList(raw, list string) Tag {
	tag := Tag{Raw: raw}
	if strings.Trim(list, refPadding) == "" {
		tag.Malformed = append(tag.Malformed, MalformedTag{Raw: raw, Reason: "the tag cites no requirement"})
		return tag
	}
	for _, piece := range strings.Split(list, refSeparator) {
		ref, err := g.IDs.ParseRef(strings.Trim(piece, refPadding))
		if err != nil {
			tag.Malformed = append(tag.Malformed, MalformedTag{Raw: raw, Reason: err.Error()})
			continue
		}
		tag.Refs = append(tag.Refs, ref)
	}
	return tag
}
