package analyzer

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

//go:embed popular_packages.json
var popularPackagesJSON []byte

// PopularPackages is the list of top popular npm package names, loaded at init.
var PopularPackages []string

func init() {
	json.Unmarshal(popularPackagesJSON, &PopularPackages)
}

// TyposquatAnalyzer detects package names that are suspiciously similar to popular packages.
type TyposquatAnalyzer struct {
	MaxDistance int // 0 means use default (2)
}

func (a *TyposquatAnalyzer) Name() string      { return "typosquat" }
func (a *TyposquatAnalyzer) NeedsTarball() bool { return false }

func (a *TyposquatAnalyzer) maxDist() int {
	if a.MaxDistance > 0 {
		return a.MaxDistance
	}
	return 2
}

func (a *TyposquatAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	name := pkg.Name

	// Strip scope for comparison — @types/express shouldn't flag against express
	bareName := name
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			bareName = parts[1]
		}
	}

	// If the package itself is in the popular list, it's not a typosquat
	for _, popular := range PopularPackages {
		if name == popular || bareName == popular {
			return nil, nil
		}
	}

	var findings []npmvet.Finding
	maxDist := a.maxDist()

	for i, popular := range PopularPackages {
		// Compare bare name against popular package (also strip scope from popular)
		popularBare := popular
		if strings.HasPrefix(popular, "@") {
			parts := strings.SplitN(popular, "/", 2)
			if len(parts) == 2 {
				popularBare = parts[1]
			}
		}

		dist := levenshtein(bareName, popularBare)
		if dist > 0 && dist <= maxDist {
			severity := npmvet.SeverityWarning
			// Distance=1 from a top-100 package is critical
			if dist == 1 && i < 100 {
				severity = npmvet.SeverityCritical
			}

			findings = append(findings, npmvet.Finding{
				Analyzer: a.Name(),
				Package:  pkg.Name,
				Severity: severity,
				Title:    "Possible typosquat of " + popular,
				Detail:   formatTyposquatDetail(bareName, popularBare, dist),
			})
			// Only report the closest match
			break
		}
	}

	return findings, nil
}

func formatTyposquatDetail(name, popular string, dist int) string {
	return "\"" + name + "\" is " + itoa(dist) + " edit(s) away from \"" + popular + "\""
}

func itoa(n int) string {
	if n == 1 {
		return "1"
	}
	if n == 2 {
		return "2"
	}
	return string(rune('0' + n))
}

// levenshtein computes the Levenshtein distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single-row optimization
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev = curr
	}

	return prev[lb]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
