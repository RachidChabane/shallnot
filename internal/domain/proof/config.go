package proof

import (
	"fmt"
	"slices"
	"strings"
)

// Defaults of the `proofs` configuration.
const (
	DefaultVerifiedDir        = ".spec-proof/verified"
	DefaultValidityThreshold  = 0.70
	DefaultAlignmentThreshold = 0.70
	DefaultMaxHostExclusions  = 2
)

// Trust is the level of trust the gate demands of the proofs it consumes.
type Trust string

// Trust levels.
const (
	// TrustSigned admits only statements produced by CI in a verified set whose manifest was written by the signature verifier.
	TrustSigned Trust = "signed"
	// TrustLocal also admits the unsigned statements produced on the developer's machine; it is feedback, never proof.
	TrustLocal Trust = "local"
)

// Escalation tells whether a finding that escalates a judged signal fails the verdict.
type Escalation string

// Escalation modes.
const (
	EscalationError   Escalation = "error"
	EscalationWarning Escalation = "warning"
)

// SignedPassBasis is the evidence a signed pass may rest on.
type SignedPassBasis string

// Pass bases.
const (
	// BasisExecutedOnly grants pass credit and exclusions on executed evidence alone.
	BasisExecutedOnly SignedPassBasis = "executed-only"
	// BasisCorroborated grants them to judged claims that name their executed corroboration.
	BasisCorroborated SignedPassBasis = "corroborated"
	// BasisJudged also grants them to judged claims without further corroboration.
	BasisJudged SignedPassBasis = "judged"
)

// Thresholds are the confidences at which a judged claim has power.
type Thresholds struct {
	// Validity is tau_judge: the confidence of a mutant validity that blocks or earns pass credit.
	Validity float64
	// Alignment is the confidence of an ALIGNED verdict that clears alignment.
	Alignment float64
}

// Config is the `proofs` block of shallnot.yaml with its defaults applied.
type Config struct {
	// VerifiedDir holds the verified proof set, relative to the configuration file.
	VerifiedDir string
	// Require lists the kinds whose proof every criterion in focus must hold.
	Require []Kind
	// Code lists the globs of the source files a proof's code_tree input is derived from.
	Code            []string
	Trust           Trust
	Thresholds      Thresholds
	Escalation      Escalation
	SignedPassBasis SignedPassBasis
	// MinIsolation is the isolation floor below which a judged response earns no pass credit.
	MinIsolation Isolation
	// MaxHostExclusions bounds the survivors a criterion may exclude on a judgment alone.
	MaxHostExclusions int
}

// DefaultConfig returns the configuration of a project that sets nothing but the block itself.
func DefaultConfig() Config {
	return Config{
		VerifiedDir:       DefaultVerifiedDir,
		Trust:             TrustSigned,
		Thresholds:        Thresholds{Validity: DefaultValidityThreshold, Alignment: DefaultAlignmentThreshold},
		Escalation:        EscalationError,
		SignedPassBasis:   BasisCorroborated,
		MinIsolation:      DefaultMinIsolation,
		MaxHostExclusions: DefaultMaxHostExclusions,
	}
}

// Requires reports whether every criterion must hold a proof of the kind.
func (c Config) Requires(kind Kind) bool { return slices.Contains(c.Require, kind) }

func parseClosed[T ~string](what, text string, allowed []T) (T, error) {
	for _, candidate := range allowed {
		if string(candidate) == text {
			return candidate, nil
		}
	}
	names := make([]string, len(allowed))
	for index, candidate := range allowed {
		names[index] = string(candidate)
	}
	var zero T
	return zero, fmt.Errorf("%s %q is not one of %s", what, text, strings.Join(names, ", "))
}

// ParseTrust reads a trust level.
func ParseTrust(text string) (Trust, error) {
	return parseClosed("trust", text, []Trust{TrustSigned, TrustLocal})
}

// ParseEscalation reads an escalation mode.
func ParseEscalation(text string) (Escalation, error) {
	return parseClosed("escalation", text, []Escalation{EscalationError, EscalationWarning})
}

// ParseSignedPassBasis reads a pass basis.
func ParseSignedPassBasis(text string) (SignedPassBasis, error) {
	return parseClosed("signed_pass_basis", text, []SignedPassBasis{BasisExecutedOnly, BasisCorroborated, BasisJudged})
}

// ParseIsolation reads an isolation level.
func ParseIsolation(text string) (Isolation, error) {
	return parseClosed("min_isolation", text, isolations)
}

// ParseRequired reads a kind a criterion must hold a proof of, named by its kind or by its predicate type URI.
// Completeness is advisory and cannot be required.
func ParseRequired(text string) (Kind, error) {
	kind, known := KindOfPredicateType(text)
	if !known {
		var err error
		if kind, err = parseClosed("require", text, Kinds()); err != nil {
			return "", err
		}
	}
	if kind == KindCompleteness {
		return "", fmt.Errorf("require: %s proofs are advisory and cannot be required", KindCompleteness)
	}
	return kind, nil
}

// KindNames lists the wire names of the kinds.
func KindNames() []string {
	names := make([]string, 0, len(Kinds()))
	for _, kind := range Kinds() {
		names = append(names, string(kind))
	}
	return names
}

// TrustNames lists the wire names of the trust levels.
func TrustNames() []string { return []string{string(TrustSigned), string(TrustLocal)} }

// EscalationNames lists the wire names of the escalation modes.
func EscalationNames() []string { return []string{string(EscalationError), string(EscalationWarning)} }

// SignedPassBasisNames lists the wire names of the pass bases.
func SignedPassBasisNames() []string {
	return []string{string(BasisExecutedOnly), string(BasisCorroborated), string(BasisJudged)}
}
