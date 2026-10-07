package proofgate

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/RachidChabane/shallnot/internal/domain"
	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

// classified is a statement file with the state the gate gave it.
type classified struct {
	proof.LoadedStatement
	State  proof.State
	Reason string
}

func (c classified) valid() bool { return c.State == proof.StateValid }

// classify gives each statement of the verified set a state. A proof is valid only when its subject, every input
// the gate can derive from the tree and every proof it consumed are current, and its trust is the one demanded.
func classify(input Input) []classified {
	if !input.Evidence.Usable() {
		return nil
	}
	statements := make([]classified, len(input.Evidence.Statements))
	for index, loaded := range input.Evidence.Statements {
		statements[index] = classified{LoadedStatement: loaded, State: proof.StateValid}
		switch {
		case loaded.Ignored:
			statements[index].State = proof.StateIgnored
		case loaded.Problem != "":
			statements[index].State, statements[index].Reason = proof.StateMalformed, loaded.Problem
		default:
			if reason := staleReason(input, *loaded.Statement); reason != "" {
				statements[index].State, statements[index].Reason = proof.StateStale, reason
			}
		}
	}
	invalidateConsumersOfStale(statements)
	return statements
}

// invalidateConsumersOfStale marks stale every statement that consumes a proof which is not valid, until none changes.
func invalidateConsumersOfStale(statements []classified) {
	for changed := true; changed; {
		changed = false
		valid := map[proof.Kind]map[string]bool{}
		for _, candidate := range statements {
			if candidate.valid() {
				if valid[candidate.Statement.Kind] == nil {
					valid[candidate.Statement.Kind] = map[string]bool{}
				}
				valid[candidate.Statement.Kind][candidate.Digest] = true
			}
		}
		for index, candidate := range statements {
			if !candidate.valid() {
				continue
			}
			for _, consumed := range candidate.Statement.Core().Inputs {
				if consumed.Role == proof.RoleProof && !valid[proof.Kind(consumed.Name)][consumed.Digest.SHA256] {
					statements[index].State = proof.StateStale
					statements[index].Reason = fmt.Sprintf("the %s proof %s it consumes is not in the verified set or is stale", consumed.Name, abbreviate(consumed.Digest.SHA256))
					changed = true
					break
				}
			}
		}
	}
}

// staleReason says why a well-formed statement is not current, or returns "".
func staleReason(input Input, statement proof.Statement) string {
	core := statement.Core()
	if input.Config.Trust == proof.TrustSigned {
		if input.Evidence.Manifest.Trust != proof.TrustSigned {
			return "the verified set was accepted locally; trust: signed requires verified signatures"
		}
		if core.ProducedBy != proof.ProducedByCI {
			return "it was produced locally; trust: signed requires a proof produced by CI"
		}
	}
	if reason := subjectStaleReason(input, statement); reason != "" {
		return reason
	}
	for _, consumed := range core.Inputs {
		if reason := inputStaleReason(input.Evidence.Derived, consumed); reason != "" {
			return reason
		}
	}
	return ""
}

func subjectStaleReason(input Input, statement proof.Statement) string {
	subject := statement.Subject[0]
	if statement.Kind == proof.KindCompleteness {
		current, known := input.Evidence.Derived.SpecFiles[subject.Name]
		switch {
		case !known:
			return fmt.Sprintf("%s is not a known spec file", subject.Name)
		case current != subject.Digest.SHA256:
			return fmt.Sprintf("%s changed since the proof was produced", subject.Name)
		}
		return ""
	}
	requirement, _ := statement.Requirement()
	known, declared := input.Known[domain.RequirementID(requirement.ID)]
	switch {
	case !declared:
		return fmt.Sprintf("no spec declares %s", requirement.ID)
	case int(known.Revision) != requirement.Revision:
		return fmt.Sprintf("the proof is for %s but the spec declares revision %d", subject.Name, known.Revision)
	case known.Digest != subject.Digest.SHA256:
		return fmt.Sprintf("the statement of %s changed since the proof was produced", subject.Name)
	}
	return ""
}

func inputStaleReason(derived proof.DerivedInputs, consumed proof.Input) string {
	set, derivable := derived.Set(consumed.Role)
	if !derivable {
		return ""
	}
	switch {
	case set == nil:
		return fmt.Sprintf("the %s input cannot be checked: shallnot.yaml does not name its files", consumed.Role)
	case consumed.Digest.SHA256 != set.Digest:
		return fmt.Sprintf("the %s files changed since the proof was produced", consumed.Role)
	case consumed.Annotations != nil && len(consumed.Annotations.Globs) > 0 && !slices.Equal(consumed.Annotations.Globs, set.Globs):
		return fmt.Sprintf("the proof expanded its %s input from [%s], not from the configured [%s]", consumed.Role,
			strings.Join(consumed.Annotations.Globs, listSeparator), strings.Join(set.Globs, listSeparator))
	}
	return ""
}

// consumedReport lists every statement file with its state, ordered by path.
func consumedReport(statements []classified) []proof.ConsumedProof {
	consumed := make([]proof.ConsumedProof, 0, len(statements))
	for _, statement := range statements {
		entry := proof.ConsumedProof{Path: statement.Path, Digest: statement.Digest, State: statement.State, Reason: statement.Reason}
		if statement.Statement != nil {
			entry.PredicateType = statement.Statement.PredicateType
			if len(statement.Statement.Subject) > 0 {
				entry.Subject = statement.Statement.Subject[0].Name
			}
		}
		consumed = append(consumed, entry)
	}
	sort.SliceStable(consumed, func(i, j int) bool { return consumed[i].Path < consumed[j].Path })
	return consumed
}
