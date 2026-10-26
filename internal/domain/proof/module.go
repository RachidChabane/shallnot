package proof

import "strings"

// ModuleName names the binary of the module that produces the proofs of a kind.
func (k Kind) ModuleName() string {
	switch k {
	case KindAlignment:
		return "musthold-alignment"
	case KindDiscriminatingPower:
		return "musthold-dp"
	default:
		return "musthold-completeness"
	}
}

const (
	fulfilSkillSuffix = "-fulfil"
	// UmbrellaSkill is the skill that runs the fulfil loop of every module present.
	UmbrellaSkill = "musthold:prove-all"
)

// KindOfModule resolves the name of a module, by its binary or by its kind, to its kind.
func KindOfModule(name string) (Kind, bool) {
	for _, kind := range Kinds() {
		if name == kind.ModuleName() || name == string(kind) {
			return kind, true
		}
	}
	return "", false
}

// FulfilSkill names the skill that answers the requests a module has pending, or the umbrella skill for an unknown module.
func FulfilSkill(module string) string {
	kind, known := KindOfModule(module)
	if !known {
		return UmbrellaSkill
	}
	return kind.ModuleName() + fulfilSkillSuffix
}

// ProveCommand is the command that regenerates the proofs of a module for the criteria that changed.
func (k Kind) ProveCommand() string {
	return strings.Join([]string{k.ModuleName(), "prove", "--changed"}, " ")
}

// PublishCommand names how to publish proofs into the configured verified set.
// Signed trust requires bundles from musthold-ledger ledger export, never local acceptance.
func (c Config) PublishCommand() string {
	input := "--bundles <ledger export dir>"
	if c.Trust == TrustLocal {
		input = "--accept-local " + DefaultCacheDir
	}
	return "musthold-verify " + input + " --out " + c.VerifiedDir
}
