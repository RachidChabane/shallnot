package proof

import (
	"strconv"
	"strings"
)

// ConfidenceSource tells where the confidence of a decision came from, so that thresholds are keyed per source. It is
// a kind or a `kind(detail)` or `kind:detail` string.
type ConfidenceSource string

// Kinds of confidence source, as the format writes them.
const (
	// SourceExecution is a decision that rests on execution or construction: the module fixed the confidence.
	SourceExecution ConfidenceSource = "execution"
	// SourceHostVotes is the confidence of k blind votes of the host, written `host-votes(k)`.
	SourceHostVotes ConfidenceSource = "host-votes"
	// SourceSystemOne is the confidence a System One endpoint reported, written `system-one:<model>@<endpointDigest>`.
	SourceSystemOne ConfidenceSource = "system-one"

	hostVotesOpen  = "("
	hostVotesClose = ")"
	systemOneColon = ":"
)

// SourceKinds lists the kinds of confidence source in a stable order.
func SourceKinds() []ConfidenceSource {
	return []ConfidenceSource{SourceExecution, SourceHostVotes, SourceSystemOne}
}

// SourceKindNames lists the wire names of the kinds of confidence source.
func SourceKindNames() []string {
	names := make([]string, 0, len(SourceKinds()))
	for _, kind := range SourceKinds() {
		names = append(names, string(kind))
	}
	return names
}

// IsKnown reports whether the source has one of the forms of the format.
func (s ConfidenceSource) IsKnown() bool {
	text := string(s)
	switch {
	case s == SourceExecution:
		return true
	case strings.HasPrefix(text, string(SourceHostVotes)+hostVotesOpen) && strings.HasSuffix(text, hostVotesClose):
		inner := strings.TrimSuffix(strings.TrimPrefix(text, string(SourceHostVotes)+hostVotesOpen), hostVotesClose)
		votes, err := strconv.Atoi(inner)
		return err == nil && votes >= 1
	case strings.HasPrefix(text, string(SourceSystemOne)+systemOneColon):
		return len(text) > len(SourceSystemOne)+len(systemOneColon)
	}
	return false
}

// Kind returns the kind of the source: the source without its detail. A source that is not known has no kind.
func (s ConfidenceSource) Kind() (ConfidenceSource, bool) {
	if !s.IsKnown() {
		return "", false
	}
	text := string(s)
	if cut := strings.IndexAny(text, hostVotesOpen+systemOneColon); cut >= 0 {
		return ConfidenceSource(text[:cut]), true
	}
	return s, true
}
