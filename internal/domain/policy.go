package domain

// Policy decides how severe each finding is and which severities block.
type Policy struct {
	Severities map[Category]Severity
	FailOn     Severity
	Advisory   bool
}

// DefaultFailOn is the lowest severity that blocks when none is configured.
const DefaultFailOn = SeverityError

// SeverityOf returns the configured severity of a category.
func (p Policy) SeverityOf(category Category) Severity {
	if severity, overridden := p.Severities[category]; overridden {
		return severity
	}
	return category.DefaultSeverity()
}

// Blocks reports whether a finding of that severity fails the run.
func (p Policy) Blocks(severity Severity) bool {
	return severity != SeverityOff && severity >= p.FailOn
}
