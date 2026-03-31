package analyzer

import (
	"testing"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"express", "expresss", 1},
		{"lodash", "lodahs", 2},
		{"react", "reacr", 1},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestTyposquatAnalyzer_PopularPackageItself(t *testing.T) {
	a := &TyposquatAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "express"},
		AnalyzerContext{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("popular package itself should not trigger, got %d findings", len(findings))
	}
}

func TestTyposquatAnalyzer_CloseTypo(t *testing.T) {
	a := &TyposquatAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "expresss"},
		AnalyzerContext{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected finding for 'expresss' (typo of express)")
	}
	// express is in top 100 and distance=1, should be critical
	if findings[0].Severity != npmvet.SeverityCritical {
		t.Errorf("expected critical, got %s", findings[0].Severity)
	}
}

func TestTyposquatAnalyzer_ScopedPackageNotFlagged(t *testing.T) {
	a := &TyposquatAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "@types/node"},
		AnalyzerContext{},
	)
	if err != nil {
		t.Fatal(err)
	}
	// @types/node is in the popular list, should not flag
	if len(findings) != 0 {
		t.Errorf("@types/node should not be flagged, got %d findings", len(findings))
	}
}

func TestTyposquatAnalyzer_UnrelatedName(t *testing.T) {
	a := &TyposquatAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "my-unique-project-name-xyz"},
		AnalyzerContext{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("unrelated name should not trigger, got %d findings", len(findings))
	}
}
