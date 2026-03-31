package analyzer

import (
	"strings"
	"testing"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/registry"
)

func TestObfuscationAnalyzer_CleanCode(t *testing.T) {
	a := &ObfuscationAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "clean-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/index.js", Content: "module.exports = function() { return 42; };"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("clean code should have no findings, got %d", len(findings))
	}
}

func TestObfuscationAnalyzer_EvalUsage(t *testing.T) {
	a := &ObfuscationAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "eval-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/index.js", Content: "var x = eval('1+1');"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	hasWarning := false
	for _, f := range findings {
		if f.Severity == npmvet.SeverityWarning && strings.Contains(f.Title, "eval") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected warning for eval() usage")
	}
}

func TestObfuscationAnalyzer_LongString(t *testing.T) {
	a := &ObfuscationAnalyzer{MaxStringLength: 100}
	longStr := "'" + strings.Repeat("a", 150) + "'"
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "long-str-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/index.js", Content: "var x = " + longStr + ";"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	hasLongStr := false
	for _, f := range findings {
		if strings.Contains(f.Title, "String longer than") {
			hasLongStr = true
		}
	}
	if !hasLongStr {
		t.Error("expected finding for long string")
	}
}

func TestObfuscationAnalyzer_EvalPlusLongString_Critical(t *testing.T) {
	a := &ObfuscationAnalyzer{MaxStringLength: 50}
	longStr := "'" + strings.Repeat("x", 100) + "'"
	code := "eval(" + longStr + ");"
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "malicious-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/index.js", Content: code},
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
		t.Error("expected critical finding for eval + long string combo")
	}
}

func TestObfuscationAnalyzer_HexEscapes(t *testing.T) {
	a := &ObfuscationAnalyzer{}
	// Generate content with many hex escapes
	content := "var x = "
	for i := 0; i < 25; i++ {
		content += "\"\\x41\\x42\""
		if i < 24 {
			content += "+"
		}
	}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "hex-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/index.js", Content: content},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	hasHex := false
	for _, f := range findings {
		if strings.Contains(f.Title, "hex escape") {
			hasHex = true
		}
	}
	if !hasHex {
		t.Error("expected finding for high hex escape density")
	}
}

func TestObfuscationAnalyzer_SkipsNonJS(t *testing.T) {
	a := &ObfuscationAnalyzer{}
	findings, err := a.Analyze(
		npmvet.PackageSpec{Name: "readme-pkg"},
		AnalyzerContext{Tarball: []registry.TarballFile{
			{Path: "package/README.md", Content: "eval('this is markdown not code')"},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("non-JS files should be skipped, got %d findings", len(findings))
	}
}
