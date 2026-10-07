package proof

import (
	"cmp"
	"encoding/json"
	"fmt"
	"strings"
)

// PendingReason says why a proof is not there yet.
type PendingReason string

// Pending reasons.
const (
	// PendingAwaitingHost: the host harness has requests to answer.
	PendingAwaitingHost PendingReason = "awaiting_host"
	// PendingAwaitingProof: another proof this one consumes is not there yet.
	PendingAwaitingProof PendingReason = "awaiting_proof"
	// PendingAwaitingHostJudgment: a judgment of the host is needed on a survivor CI produced.
	PendingAwaitingHostJudgment PendingReason = "awaiting_host_judgment"
	// PendingHostUnresponsive: the host did not answer after every emission.
	PendingHostUnresponsive PendingReason = "host_unresponsive"
)

var pendingReasons = []PendingReason{PendingAwaitingHost, PendingAwaitingProof, PendingAwaitingHostJudgment, PendingHostUnresponsive}

// PendingReasons lists the closed set of reasons.
func PendingReasons() []PendingReason { return append([]PendingReason(nil), pendingReasons...) }

// Pending is a proof a module is still producing for a criterion.
type Pending struct {
	Module    string        `json:"module"`
	Criterion string        `json:"criterion"`
	Reason    PendingReason `json:"reason"`
}

// ManifestEntry lists one statement file of the verified set with the SHA-256 of its bytes.
type ManifestEntry struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// Manifest is the `manifest.json` of the verified proof set, written by the verifier.
type Manifest struct {
	VerifierVersion string `json:"verifierVersion"`
	// Trust tells how the set was verified: signatures against the trust policy, or an acceptance of local statements.
	Trust             Trust           `json:"trust"`
	TrustedRootDigest string          `json:"trustedRootDigest,omitempty"`
	PolicyDigest      string          `json:"policyDigest,omitempty"`
	Statements        []ManifestEntry `json:"statements"`
	Pending           []Pending       `json:"pending"`
}

// ParseManifest reads a manifest and returns every violation of its invariants.
// Unknown members are ignored.
func ParseManifest(data []byte) (Manifest, []Violation, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, nil, fmt.Errorf("decode manifest: %w", err)
	}
	var sink collector
	checkNotEmpty(&sink, "verifierVersion", manifest.VerifierVersion)
	checkMember(&sink, "trust", manifest.Trust, []Trust{TrustSigned, TrustLocal})
	if manifest.Trust == TrustSigned {
		checkDigest(&sink, "trustedRootDigest", manifest.TrustedRootDigest)
		checkDigest(&sink, "policyDigest", manifest.PolicyDigest)
	}
	for index, entry := range manifest.Statements {
		path := fmt.Sprintf("statements[%d]", index)
		if !IsRelativePOSIXPath(entry.Path) {
			sink.add(CodeSubject, path+".path", "%q is not a relative POSIX path", entry.Path)
		}
		checkDigest(&sink, path+".digest", entry.Digest)
	}
	checkAscending(&sink, "statements", manifest.Statements, func(a, b ManifestEntry) int { return strings.Compare(a.Path, b.Path) })
	for index, pending := range manifest.Pending {
		path := fmt.Sprintf("pending[%d]", index)
		checkNotEmpty(&sink, path+".module", pending.Module)
		if _, _, ok := ParseCriterion(pending.Criterion); !ok {
			sink.add(CodeRequirement, path+".criterion", "%q is not ID~REV", pending.Criterion)
		}
		checkMember(&sink, path+".reason", pending.Reason, pendingReasons)
	}
	checkAscending(&sink, "pending", manifest.Pending, func(a, b Pending) int {
		return cmp.Or(strings.Compare(a.Module, b.Module), strings.Compare(a.Criterion, b.Criterion), strings.Compare(string(a.Reason), string(b.Reason)))
	})
	return manifest, sink.violations, nil
}

// ParseCriterion splits `ID~REV`.
func ParseCriterion(text string) (id string, revision int, ok bool) {
	id, revisionText, found := strings.Cut(text, RevisionSeparator)
	if !found || id == "" {
		return "", 0, false
	}
	if _, err := fmt.Sscanf(revisionText, "%d", &revision); err != nil || revision < MinimumRevision || fmt.Sprint(revision) != revisionText {
		return "", 0, false
	}
	return id, revision, true
}
