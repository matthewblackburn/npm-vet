package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// ObfuscationAnalyzer scans package source code for signs of obfuscation.
type ObfuscationAnalyzer struct {
	MaxStringLength int // 0 means use default (1000)
}

func (a *ObfuscationAnalyzer) Name() string      { return "obfuscation" }
func (a *ObfuscationAnalyzer) NeedsTarball() bool { return true }

func (a *ObfuscationAnalyzer) maxStringLen() int {
	if a.MaxStringLength > 0 {
		return a.MaxStringLength
	}
	return 1000
}

var (
	evalPattern       = regexp.MustCompile(`\beval\s*\(`)
	newFunctionPat    = regexp.MustCompile(`new\s+Function\s*\(`)
	bufferFromPat     = regexp.MustCompile(`Buffer\.from\s*\([^)]*(?:base64|hex)["']`)
	hexEscapePattern  = regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	unicodeEscapePat  = regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
)

func (a *ObfuscationAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	if ctx.Tarball == nil {
		return nil, nil
	}

	var findings []npmvet.Finding
	hasEval := false
	hasLongString := false
	maxStringLen := a.maxStringLen()

	for _, file := range ctx.Tarball {
		// Skip non-JS files and minified bundles (common in legitimate packages)
		if !isJSFile(file.Path) {
			continue
		}

		content := file.Content

		// Check for eval()
		if evalPattern.MatchString(content) {
			hasEval = true
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "eval() usage detected",
				Detail:   fmt.Sprintf("File: %s", file.Path),
			})
		}

		// Check for new Function()
		if newFunctionPat.MatchString(content) {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "new Function() usage detected",
				Detail:   fmt.Sprintf("File: %s", file.Path),
			})
		}

		// Check for Buffer.from with base64/hex encoding
		if bufferFromPat.MatchString(content) {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "Buffer.from() with encoding detected",
				Detail:   fmt.Sprintf("File: %s — may indicate encoded payload", file.Path),
			})
		}

		// Check for suspiciously long strings
		if hasLongStringLiteral(content, maxStringLen) {
			hasLongString = true
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    fmt.Sprintf("String longer than %d characters", maxStringLen),
				Detail:   fmt.Sprintf("File: %s", file.Path),
			})
		}

		// Check for high density of hex escapes
		hexCount := len(hexEscapePattern.FindAllString(content, -1))
		if hexCount > 20 {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "High density of hex escape sequences",
				Detail:   fmt.Sprintf("File: %s — %d occurrences", file.Path, hexCount),
			})
		}

		// Check for high density of unicode escapes
		unicodeCount := len(unicodeEscapePat.FindAllString(content, -1))
		if unicodeCount > 20 {
			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: npmvet.SeverityWarning,
				Title:    "High density of unicode escape sequences",
				Detail:   fmt.Sprintf("File: %s — %d occurrences", file.Path, unicodeCount),
			})
		}
	}

	// Escalate: eval + long encoded string = critical
	if hasEval && hasLongString {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityCritical,
			Title:    "eval() combined with long encoded strings",
			Detail:   "This combination is a strong indicator of obfuscated malicious code",
		})
	}

	return findings, nil
}

// hasLongStringLiteral checks if the content contains a string literal longer than maxLen.
func hasLongStringLiteral(content string, maxLen int) bool {
	inString := false
	stringChar := byte(0)
	currentLen := 0

	for i := 0; i < len(content); i++ {
		c := content[i]

		if inString {
			if c == '\\' && i+1 < len(content) {
				i++ // skip escaped char
				currentLen += 2
				continue
			}
			if c == stringChar {
				if currentLen > maxLen {
					return true
				}
				inString = false
				continue
			}
			currentLen++
		} else {
			if c == '\'' || c == '"' || c == '`' {
				inString = true
				stringChar = c
				currentLen = 0
			}
		}
	}

	return false
}

func isJSFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".js") ||
		strings.HasSuffix(lower, ".cjs") ||
		strings.HasSuffix(lower, ".mjs")
}
