package analyzer

import (
	"testing"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

func TestMaintainerAnalyzer_FirstEncounter(t *testing.T) {
	dir := t.TempDir()
	a := &MaintainerAnalyzer{CacheDir: dir}

	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "new-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{
			Maintainers: []registry.Maintainer{{Name: "alice", Email: "alice@example.com"}},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	// Silent on first encounter
	if len(findings) != 0 {
		t.Errorf("expected no findings on first encounter, got %d", len(findings))
	}
}

func TestMaintainerAnalyzer_NoChange(t *testing.T) {
	dir := t.TempDir()
	a := &MaintainerAnalyzer{CacheDir: dir}

	meta := &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}, {Name: "bob"}},
	}
	pkg := npmvet.PackageSpec{Name: "stable-pkg"}

	// First call: populate cache
	a.Analyze(pkg, AnalyzerContext{Metadata: meta})

	// Second call: same maintainers
	findings, err := a.Analyze(pkg, AnalyzerContext{Metadata: meta})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for unchanged maintainers, got %d", len(findings))
	}
}

func TestMaintainerAnalyzer_NewMaintainerAdded(t *testing.T) {
	dir := t.TempDir()
	a := &MaintainerAnalyzer{CacheDir: dir}
	pkg := npmvet.PackageSpec{Name: "growing-pkg"}

	// First call
	a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}},
	}})

	// Second call: new maintainer
	findings, err := a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}, {Name: "bob"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != npmvet.SeverityWarning {
		t.Errorf("expected warning, got %s", findings[0].Severity)
	}
}

func TestMaintainerAnalyzer_MaintainerRemoved(t *testing.T) {
	dir := t.TempDir()
	a := &MaintainerAnalyzer{CacheDir: dir}
	pkg := npmvet.PackageSpec{Name: "shrinking-pkg"}

	a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}, {Name: "bob"}},
	}})

	findings, err := a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != npmvet.SeverityWarning {
		t.Errorf("expected warning, got %s", findings[0].Severity)
	}
}

func TestMaintainerAnalyzer_AllReplaced(t *testing.T) {
	dir := t.TempDir()
	a := &MaintainerAnalyzer{CacheDir: dir}
	pkg := npmvet.PackageSpec{Name: "takeover-pkg"}

	a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "alice"}, {Name: "bob"}},
	}})

	findings, err := a.Analyze(pkg, AnalyzerContext{Metadata: &registry.PackageMetadata{
		Maintainers: []registry.Maintainer{{Name: "eve"}, {Name: "mallory"}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	hasCritical := false
	for _, f := range findings {
		if f.Severity == npmvet.SeverityCritical {
			hasCritical = true
		}
	}
	if !hasCritical {
		t.Error("expected critical finding when all maintainers replaced")
	}
}
