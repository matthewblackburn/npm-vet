package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/matthewblackburn/npm-vet/internal/npmvet"
)

func TestJSONReport_Empty(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := JSONReport(nil)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var findings []npmvet.Finding
	if err := json.Unmarshal(buf.Bytes(), &findings); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected empty array, got %d items", len(findings))
	}
}

func TestJSONReport_WithFindings(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	findings := []npmvet.Finding{
		{
			Analyzer: "typosquat",
			Package:  "expresss",
			Severity: npmvet.SeverityCritical,
			Title:    "Possible typosquat of express",
			Detail:   "distance 1",
		},
	}

	err := JSONReport(findings)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result []npmvet.Finding
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result))
	}
	if result[0].Package != "expresss" {
		t.Errorf("package = %q, want %q", result[0].Package, "expresss")
	}
}

func TestSARIFReport(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	findings := []npmvet.Finding{
		{
			Analyzer: "postinstall",
			Package:  "evil-pkg",
			Severity: npmvet.SeverityCritical,
			Title:    "Dangerous script",
			Detail:   "curl",
		},
	}

	err := SARIFReport(findings, "1.0.0")
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// Validate it's valid JSON and has expected structure
	var sarif map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &sarif); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}
	if sarif["version"] != "2.1.0" {
		t.Errorf("SARIF version = %v, want 2.1.0", sarif["version"])
	}
	runs := sarif["runs"].([]interface{})
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
}
