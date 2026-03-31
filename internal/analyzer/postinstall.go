package analyzer

import (
	"regexp"
	"strings"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// PostinstallAnalyzer checks for suspicious lifecycle scripts.
type PostinstallAnalyzer struct{}

func (a *PostinstallAnalyzer) Name() string      { return "postinstall" }
func (a *PostinstallAnalyzer) NeedsTarball() bool { return false }

var lifecycleKeys = []string{"preinstall", "install", "postinstall", "prepare"}

// Patterns that indicate dangerous script behavior
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bcurl\b`),
	regexp.MustCompile(`\bwget\b`),
	regexp.MustCompile(`\bsh\s+-c\b`),
	regexp.MustCompile(`\bbash\s+-c\b`),
	regexp.MustCompile(`\bpowershell\b`),
	regexp.MustCompile(`\bnode\s+-e\b`),
	regexp.MustCompile(`\bexec\b`),
	regexp.MustCompile(`https?://`),
	regexp.MustCompile(`\|\s*(sh|bash|node)\b`),
}

func (a *PostinstallAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	if ctx.Metadata == nil || ctx.Metadata.Scripts == nil {
		return nil, nil
	}

	var findings []npmvet.Finding

	for _, key := range lifecycleKeys {
		script, ok := ctx.Metadata.Scripts[key]
		if !ok {
			continue
		}

		script = strings.TrimSpace(script)
		if script == "" {
			continue
		}

		// Check for dangerous patterns
		isDangerous := false
		var matchedPatterns []string
		for _, pat := range dangerousPatterns {
			if pat.MatchString(script) {
				isDangerous = true
				matchedPatterns = append(matchedPatterns, pat.String())
			}
		}

		if isDangerous {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityCritical,
				Title:    "Dangerous lifecycle script: " + key,
				Detail:   "Script: " + truncate(script, 200) + "\nMatched patterns: " + strings.Join(matchedPatterns, ", "),
			})
		} else {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "Lifecycle script present: " + key,
				Detail:   "Script: " + truncate(script, 200),
			})
		}
	}

	return findings, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
