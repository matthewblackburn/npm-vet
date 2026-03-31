package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

// packageJSON represents the relevant fields of a package.json file.
type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// lockfileJSON represents the relevant fields of a package-lock.json (v2/v3).
type lockfileJSON struct {
	LockfileVersion int                       `json:"lockfileVersion"`
	Packages        map[string]lockfileEntry  `json:"packages"`
}

type lockfileEntry struct {
	Version  string `json:"version"`
	Resolved string `json:"resolved"`
}

// ReadPackageJSON reads package.json from the given directory and returns
// all direct dependencies as PackageSpecs.
func ReadPackageJSON(dir string) ([]npmvet.PackageSpec, error) {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %w", err)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}

	var specs []npmvet.PackageSpec

	for name, versionRange := range pkg.Dependencies {
		specs = append(specs, npmvet.PackageSpec{
			Name:         name,
			VersionRange: versionRange,
		})
	}

	for name, versionRange := range pkg.DevDependencies {
		specs = append(specs, npmvet.PackageSpec{
			Name:         name,
			VersionRange: versionRange,
		})
	}

	return specs, nil
}

// ReadLockfile attempts to read package-lock.json from the given directory
// and returns resolved versions. Returns nil (not error) if no lockfile exists.
func ReadLockfile(dir string) (map[string]string, error) {
	path := filepath.Join(dir, "package-lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading package-lock.json: %w", err)
	}

	var lock lockfileJSON
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing package-lock.json: %w", err)
	}

	resolved := make(map[string]string)

	// In lockfile v2/v3, direct deps are under "packages" with keys like
	// "node_modules/express" or "node_modules/@types/node"
	for key, entry := range lock.Packages {
		if key == "" {
			continue // root package entry
		}
		// Extract package name from the key
		// "node_modules/@scope/name" → "@scope/name"
		// "node_modules/name" → "name"
		name := key
		const prefix = "node_modules/"
		if idx := lastIndex(name, prefix); idx >= 0 {
			name = name[idx+len(prefix):]
		}
		if entry.Version != "" {
			resolved[name] = entry.Version
		}
	}

	return resolved, nil
}

func lastIndex(s, substr string) int {
	idx := -1
	searchFrom := 0
	for {
		i := indexOf(s[searchFrom:], substr)
		if i == -1 {
			break
		}
		idx = searchFrom + i
		searchFrom = idx + 1
	}
	return idx
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
