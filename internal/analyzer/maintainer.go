package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

// MaintainerAnalyzer detects changes in package maintainers.
type MaintainerAnalyzer struct {
	// CacheDir overrides the default cache directory for testing.
	CacheDir string
}

func (a *MaintainerAnalyzer) Name() string      { return "maintainer" }
func (a *MaintainerAnalyzer) NeedsTarball() bool { return false }

type maintainerCache struct {
	Packages map[string]maintainerEntry `json:"packages"`
}

type maintainerEntry struct {
	Maintainers []string `json:"maintainers"`
}

func (a *MaintainerAnalyzer) Analyze(pkg npmvet.PackageSpec, ctx AnalyzerContext) ([]npmvet.Finding, error) {
	if ctx.Metadata == nil {
		return nil, nil
	}

	current := extractMaintainerNames(ctx.Metadata.Maintainers)
	if len(current) == 0 {
		return nil, nil
	}

	cache, cachePath := a.loadCache()

	previous, hasPrevious := cache.Packages[pkg.Name]

	// Update cache with current maintainers
	cache.Packages[pkg.Name] = maintainerEntry{Maintainers: current}
	a.saveCache(cache, cachePath)

	// Silent on first encounter (no prior data to compare)
	if !hasPrevious {
		return nil, nil
	}

	prevSet := toSet(previous.Maintainers)
	currSet := toSet(current)

	var added, removed []string
	for _, m := range current {
		if !prevSet[m] {
			added = append(added, m)
		}
	}
	for _, m := range previous.Maintainers {
		if !currSet[m] {
			removed = append(removed, m)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return nil, nil
	}

	var findings []npmvet.Finding

	// All maintainers replaced → critical
	if len(removed) == len(previous.Maintainers) && len(added) > 0 {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityCritical,
			Title:    "All maintainers replaced",
			Detail: fmt.Sprintf("Previous: %s\nCurrent: %s",
				strings.Join(previous.Maintainers, ", "),
				strings.Join(current, ", ")),
		})
		return findings, nil
	}

	if len(added) > 0 {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityWarning,
			Title:    "New maintainer(s) added",
			Detail:   "Added: " + strings.Join(added, ", "),
		})
	}

	if len(removed) > 0 {
		findings = append(findings, npmvet.Finding{
			Analyzer: a.Name(),
			Package:  pkg.Name,
			Severity: npmvet.SeverityWarning,
			Title:    "Maintainer(s) removed",
			Detail:   "Removed: " + strings.Join(removed, ", "),
		})
	}

	return findings, nil
}

func (a *MaintainerAnalyzer) cacheDir() string {
	if a.CacheDir != "" {
		return a.CacheDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".npm-vet")
}

func (a *MaintainerAnalyzer) loadCache() (*maintainerCache, string) {
	dir := a.cacheDir()
	if dir == "" {
		return &maintainerCache{Packages: make(map[string]maintainerEntry)}, ""
	}

	path := filepath.Join(dir, "maintainers.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &maintainerCache{Packages: make(map[string]maintainerEntry)}, path
	}

	var cache maintainerCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return &maintainerCache{Packages: make(map[string]maintainerEntry)}, path
	}
	if cache.Packages == nil {
		cache.Packages = make(map[string]maintainerEntry)
	}

	return &cache, path
}

func (a *MaintainerAnalyzer) saveCache(cache *maintainerCache, path string) {
	if path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	// Non-fatal: log warning on write failure (e.g., read-only filesystem)
	os.WriteFile(path, data, 0o644)
}

func extractMaintainerNames(maintainers []registry.Maintainer) []string {
	names := make([]string, 0, len(maintainers))
	for _, m := range maintainers {
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)
	return names
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
