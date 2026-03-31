package npmvet

// Severity represents the risk level of a finding.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Finding represents a single security concern discovered by an analyzer.
type Finding struct {
	Analyzer string   `json:"analyzer"`
	Package  string   `json:"package"`
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
}

// PackageSpec identifies a package to be vetted.
type PackageSpec struct {
	Name         string // e.g. "express", "@types/node"
	Version      string // resolved concrete version, e.g. "5.1.0"
	VersionRange string // original specifier, e.g. "^5.0.0", "latest"
}

// SeverityAtLeast returns true if s is at least as severe as threshold.
func (s Severity) AtLeast(threshold Severity) bool {
	return severityRank(s) >= severityRank(threshold)
}

func severityRank(s Severity) int {
	switch s {
	case SeverityInfo:
		return 0
	case SeverityWarning:
		return 1
	case SeverityCritical:
		return 2
	default:
		return -1
	}
}

// ParseSeverity parses a string into a Severity, returning SeverityInfo for unknown values.
func ParseSeverity(s string) Severity {
	switch s {
	case "critical":
		return SeverityCritical
	case "warning":
		return SeverityWarning
	case "info":
		return SeverityInfo
	default:
		return SeverityInfo
	}
}
