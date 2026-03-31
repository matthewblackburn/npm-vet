package analyzer

import (
	"testing"
	"time"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

func TestDownloadsAnalyzer_HighDownloads(t *testing.T) {
	a := &DownloadsAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "popular-pkg"},
		AnalyzerContext{Downloads: &registry.DownloadStats{Downloads: 500000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("high download package should have no findings, got %d", len(findings))
	}
}

func TestDownloadsAnalyzer_LowDownloads(t *testing.T) {
	a := &DownloadsAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "obscure-pkg", Version: "1.0.0"},
		AnalyzerContext{
			Downloads: &registry.DownloadStats{Downloads: 50},
			Metadata: &registry.PackageMetadata{
				Time: map[string]string{
					"1.0.0": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != npmvet.SeverityWarning {
		t.Errorf("expected warning for old low-download pkg, got %s", findings[0].Severity)
	}
}

func TestDownloadsAnalyzer_LowDownloadsNew(t *testing.T) {
	a := &DownloadsAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "brand-new-pkg", Version: "0.0.1"},
		AnalyzerContext{
			Downloads: &registry.DownloadStats{Downloads: 5},
			Metadata: &registry.PackageMetadata{
				Time: map[string]string{
					"0.0.1": time.Now().Add(-2 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != npmvet.SeverityCritical {
		t.Errorf("expected critical for new low-download pkg, got %s", findings[0].Severity)
	}
}

func TestDownloadsAnalyzer_NilStats(t *testing.T) {
	a := &DownloadsAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "unknown-pkg"},
		AnalyzerContext{Downloads: nil},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("nil stats should skip gracefully, got %d findings", len(findings))
	}
}

func TestDownloadsAnalyzer_ModerateDownloads(t *testing.T) {
	a := &DownloadsAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "mid-pkg"},
		AnalyzerContext{Downloads: &registry.DownloadStats{Downloads: 500}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 info finding, got %d", len(findings))
	}
	if findings[0].Severity != npmvet.SeverityInfo {
		t.Errorf("expected info, got %s", findings[0].Severity)
	}
}
