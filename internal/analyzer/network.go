package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// NetworkAnalyzer detects network-related imports in install scripts.
type NetworkAnalyzer struct{}

func (a *NetworkAnalyzer) Name() string      { return "network" }
func (a *NetworkAnalyzer) NeedsTarball() bool { return true }

var networkModulePatterns = []*regexp.Regexp{
	regexp.MustCompile(`require\s*\(\s*['"](?:http|https|net|dgram|dns|http2)['"]\s*\)`),
	regexp.MustCompile(`from\s+['"](?:http|https|net|dgram|dns|http2)['"]`),
	regexp.MustCompile(`\bfetch\s*\(`),
	regexp.MustCompile(`\bXMLHttpRequest\b`),
	regexp.MustCompile(`require\s*\(\s*['"]child_process['"]\s*\)`),
	regexp.MustCompile(`from\s+['"]child_process['"]`),
}

var networkCommandPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bcurl\b`),
	regexp.MustCompile(`\bwget\b`),
	regexp.MustCompile(`\bnc\b`),
}

// lifecycleScriptKeys are the scripts that run during install.
var lifecycleScriptKeys = []string{"preinstall", "install", "postinstall", "prepare"}

func (a *NetworkAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	if ctx.Metadata == nil {
		return nil, nil
	}

	// Only analyze if the package has lifecycle scripts
	hasLifecycle := false
	var scriptFiles []string
	for _, key := range lifecycleScriptKeys {
		script, ok := ctx.Metadata.Scripts[key]
		if !ok || strings.TrimSpace(script) == "" {
			continue
		}
		hasLifecycle = true

		// Extract referenced script files (e.g., "node scripts/postinstall.js")
		if strings.HasPrefix(script, "node ") {
			parts := strings.Fields(script)
			if len(parts) >= 2 {
				scriptFiles = append(scriptFiles, parts[1])
			}
		}

		// Check the script string itself for network commands
		for _, pat := range networkCommandPatterns {
			if pat.MatchString(script) {
				return []npmvet.Finding{{
					Analyzer: a.Name(),
					Package:  pkg.Name,
					Severity: npmvet.SeverityCritical,
					Title:    "Network command in lifecycle script",
					Detail:   fmt.Sprintf("Script %s: %s", key, truncate(script, 200)),
				}}, nil
			}
		}
	}

	if !hasLifecycle || ctx.Tarball == nil {
		return nil, nil
	}

	var findings []npmvet.Finding

	// Scan referenced script files in the tarball
	for _, file := range ctx.Tarball {
		if !isReferencedFile(file.Path, scriptFiles) {
			continue
		}

		for _, pat := range networkModulePatterns {
			if pat.MatchString(file.Content) {
				findings = append(findings, npmvet.Finding{
					Analyzer: a.Name(),
					Package:  pkg.Name,
					Severity: npmvet.SeverityCritical,
					Title:    "Network module usage in install script",
					Detail:   fmt.Sprintf("File: %s — pattern: %s", file.Path, pat.String()),
				})
				break // one finding per file is enough
			}
		}
	}

	return findings, nil
}

// isReferencedFile checks if a tarball file path matches any of the referenced script files.
func isReferencedFile(tarballPath string, scriptFiles []string) bool {
	// Tarball paths are typically "package/scripts/foo.js"
	for _, sf := range scriptFiles {
		if strings.HasSuffix(tarballPath, sf) || strings.Contains(tarballPath, "/"+sf) {
			return true
		}
	}
	return false
}
