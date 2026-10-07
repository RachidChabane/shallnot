package proof

// State is how the gate classified a proof.
type State string

// States of a proof.
const (
	StateValid     State = "valid"
	StateStale     State = "stale"
	StateMalformed State = "malformed"
	StateIgnored   State = "ignored"
	StateMissing   State = "missing"
	StatePending   State = "pending"
)

// ModuleOutcome is what one module's proofs concluded for a criterion.
type ModuleOutcome string

// Outcomes of a module for a criterion.
const (
	OutcomePass     ModuleOutcome = "pass"
	OutcomeEscalate ModuleOutcome = "escalate"
	OutcomeBlock    ModuleOutcome = "block"
	// OutcomeAbsent: no valid proof, so the module cannot pass the criterion.
	OutcomeAbsent ModuleOutcome = "absent"
	// OutcomeAdvisory: the module only reports.
	OutcomeAdvisory ModuleOutcome = "advisory"
)

// ManifestReport describes the manifest the gate consumed.
type ManifestReport struct {
	Path              string `json:"path"`
	Digest            string `json:"digest"`
	VerifierVersion   string `json:"verifierVersion,omitempty"`
	TrustedRootDigest string `json:"trustedRootDigest,omitempty"`
	PolicyDigest      string `json:"policyDigest,omitempty"`
}

// ConsumedProof describes one statement file of the verified set and how it was classified.
type ConsumedProof struct {
	Path          string `json:"path"`
	Digest        string `json:"digest"`
	PredicateType string `json:"predicateType,omitempty"`
	Subject       string `json:"subject,omitempty"`
	State         State  `json:"state"`
	Reason        string `json:"reason,omitempty"`
}

// ModuleReport is the standing of one module's proofs for a criterion.
type ModuleReport struct {
	State   State         `json:"state"`
	Outcome ModuleOutcome `json:"outcome"`
}

// CriterionReport is what the proofs concluded for one criterion.
type CriterionReport struct {
	Ref string `json:"ref"`
	// Modules is keyed by the proof kind of each module that is required or present.
	Modules map[Kind]ModuleReport `json:"modules"`
	// Absent names the required modules without a valid proof.
	Absent []Kind `json:"absent"`
	// Channels counts the judged claims by the channel that answered them.
	Channels map[Channel]int `json:"channels"`
	// Isolation counts the judged claims by the isolation level of their executor.
	Isolation map[Isolation]int `json:"isolation"`
	// HostExclusions counts the survivors excluded on a judgment alone.
	HostExclusions int `json:"hostExclusions"`
	// ScreenExclusions counts the operator mutants screened out before they were listed.
	ScreenExclusions int `json:"screenExclusions"`
}

// PolicyReport restates the knobs of the policy that shaped the outcome.
type PolicyReport struct {
	SignedPassBasis    SignedPassBasis `json:"signedPassBasis"`
	MinIsolation       Isolation       `json:"minIsolation"`
	MaxHostExclusions  int             `json:"maxHostExclusions"`
	ValidityThreshold  float64         `json:"validityThreshold"`
	AlignmentThreshold float64         `json:"alignmentThreshold"`
	// ValidityThresholdBySource and AlignmentThresholdBySource list the thresholds that differ by confidence source.
	ValidityThresholdBySource  map[ConfidenceSource]float64 `json:"validityThresholdBySource,omitempty"`
	AlignmentThresholdBySource map[ConfidenceSource]float64 `json:"alignmentThresholdBySource,omitempty"`
	Escalation                 Escalation                   `json:"escalation"`
	// IsolationFloorLowered flags a floor weaker than the default.
	IsolationFloorLowered bool `json:"isolationFloorLowered"`
}

// Report is what `extensions.proofs` of the JSON report holds.
type Report struct {
	Trust    Trust             `json:"trust"`
	Policy   PolicyReport      `json:"policy"`
	Manifest *ManifestReport   `json:"manifest"`
	Consumed []ConsumedProof   `json:"consumed"`
	Pending  []Pending         `json:"pending"`
	Criteria []CriterionReport `json:"criteria"`
}
