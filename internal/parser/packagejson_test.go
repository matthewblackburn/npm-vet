package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
		"name": "test-project",
		"dependencies": {
			"express": "^5.0.0",
			"@types/node": "^20.0.0"
		},
		"devDependencies": {
			"typescript": "~5.7.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := ReadPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(specs) != 3 {
		t.Fatalf("got %d specs, want 3", len(specs))
	}

	found := map[string]string{}
	for _, s := range specs {
		found[s.Name] = s.VersionRange
	}

	tests := map[string]string{
		"express":      "^5.0.0",
		"@types/node":  "^20.0.0",
		"typescript":   "~5.7.0",
	}
	for name, want := range tests {
		got, ok := found[name]
		if !ok {
			t.Errorf("missing package %q", name)
			continue
		}
		if got != want {
			t.Errorf("package %q version = %q, want %q", name, got, want)
		}
	}
}

func TestReadPackageJSON_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadPackageJSON(dir)
	if err == nil {
		t.Fatal("expected error for missing package.json")
	}
}

func TestReadPackageJSON_Malformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadPackageJSON(dir)
	if err == nil {
		t.Fatal("expected error for malformed package.json")
	}
}

func TestReadLockfile(t *testing.T) {
	dir := t.TempDir()
	lockJSON := `{
		"lockfileVersion": 3,
		"packages": {
			"": { "name": "test-project" },
			"node_modules/express": { "version": "5.1.0", "resolved": "https://registry.npmjs.org/express/-/express-5.1.0.tgz" },
			"node_modules/@types/node": { "version": "20.11.5", "resolved": "https://registry.npmjs.org/@types/node/-/node-20.11.5.tgz" },
			"node_modules/express/node_modules/qs": { "version": "6.11.0" }
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(lockJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ReadLockfile(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should find express, @types/node, and qs (nested dep)
	if v := resolved["express"]; v != "5.1.0" {
		t.Errorf("express version = %q, want %q", v, "5.1.0")
	}
	if v := resolved["@types/node"]; v != "20.11.5" {
		t.Errorf("@types/node version = %q, want %q", v, "20.11.5")
	}
	if v := resolved["qs"]; v != "6.11.0" {
		t.Errorf("qs version = %q, want %q", v, "6.11.0")
	}
}

func TestReadLockfile_NotFound(t *testing.T) {
	dir := t.TempDir()
	resolved, err := ReadLockfile(dir)
	if err != nil {
		t.Fatal("expected nil error for missing lockfile")
	}
	if resolved != nil {
		t.Fatal("expected nil resolved for missing lockfile")
	}
}
