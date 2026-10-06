package domain

// Dimensions of the completeness checklist c1: the aspects of a feature that a
// spec may declare out of its scope.
const (
	DimensionErrorHandling       = "error_handling"
	DimensionBoundaries          = "boundaries"
	DimensionPermissions         = "permissions"
	DimensionConcurrency         = "concurrency"
	DimensionCompatibility       = "compatibility"
	DimensionInputValidation     = "input_validation"
	DimensionTimeAndLocale       = "time_and_locale"
	DimensionStateAndPersistence = "state_and_persistence"
)

var dimensions = []string{
	DimensionErrorHandling,
	DimensionBoundaries,
	DimensionPermissions,
	DimensionConcurrency,
	DimensionCompatibility,
	DimensionInputValidation,
	DimensionTimeAndLocale,
	DimensionStateAndPersistence,
}

// Dimensions lists every dimension a spec may declare out of scope.
func Dimensions() []string { return append([]string(nil), dimensions...) }

// IsKnownDimension reports whether a dimension is in the closed list.
func IsKnownDimension(dimension string) bool {
	for _, known := range dimensions {
		if known == dimension {
			return true
		}
	}
	return false
}

// OutOfScope is a spec's declaration that one dimension is not covered, and why.
type OutOfScope struct {
	Location  Location
	Dimension string
	Reason    string
}
