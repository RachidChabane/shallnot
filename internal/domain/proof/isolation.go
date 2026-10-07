package proof

import "slices"

// Isolation is the level of isolation an executor ran with.
type Isolation string

// Isolation levels, listed from the strongest to the weakest.
const (
	IsolationRestrictedAttested   Isolation = "restricted-attested"
	IsolationRestricted           Isolation = "restricted"
	IsolationRestrictedUnverified Isolation = "restricted-unverified"
	IsolationUnrestricted         Isolation = "unrestricted"
	IsolationInline               Isolation = "inline"
)

// DefaultMinIsolation is the floor below which a response earns no judged pass credit.
const DefaultMinIsolation = IsolationRestricted

var isolations = []Isolation{
	IsolationRestrictedAttested, IsolationRestricted, IsolationRestrictedUnverified,
	IsolationUnrestricted, IsolationInline,
}

// Isolations lists the levels, strongest first.
func Isolations() []Isolation { return slices.Clone(isolations) }

// IsolationNames lists the wire names of the levels, strongest first.
func IsolationNames() []string {
	names := make([]string, len(isolations))
	for index, level := range isolations {
		names[index] = string(level)
	}
	return names
}

// Strength ranks the level: the larger, the stronger. An unknown level ranks 0.
func (i Isolation) Strength() int {
	index := slices.Index(isolations, i)
	if index < 0 {
		return 0
	}
	return len(isolations) - index
}

// AtLeast reports whether the level is as strong as the floor. An unknown level is never at least any floor.
func (i Isolation) AtLeast(floor Isolation) bool {
	return i.Strength() > 0 && i.Strength() >= floor.Strength()
}
