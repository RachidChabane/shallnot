// Package testkit holds helpers shared by the tests.
package testkit

import (
	"strings"
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain"
)

// Tag builds a bracket tag citing refs. Tests build sample tags with it so
// that the samples are not themselves bindings of the test file they sit in.
func Tag(refs ...string) string {
	return "[" + domain.TagKeyword + " " + strings.Join(refs, ", ") + "]"
}

// Grammar returns a tag grammar for the default ID pattern.
func Grammar(t testing.TB) domain.TagGrammar {
	t.Helper()
	return GrammarFor(t, domain.DefaultIDPattern)
}

// GrammarFor returns a tag grammar for an ID pattern.
func GrammarFor(t testing.TB, pattern string) domain.TagGrammar {
	t.Helper()
	ids, err := domain.NewIDPattern(pattern)
	if err != nil {
		t.Fatalf("id pattern: %v", err)
	}
	return domain.TagGrammar{IDs: ids}
}

// Ref builds a reference.
func Ref(id string, revision int) domain.Ref {
	return domain.Ref{ID: domain.RequirementID(id), Revision: domain.Revision(revision)}
}
