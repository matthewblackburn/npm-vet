package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.FailOn != "critical" {
		t.Errorf("default FailOn = %q, want %q", cfg.FailOn, "critical")
	}
	if !cfg.IsAnalyzerEnabled("postinstall") {
		t.Error("postinstall should be enabled by default")
	}
	if cfg.Thresholds.MinWeeklyDownloads != 100 {
		t.Errorf("default MinWeeklyDownloads = %d, want 100", cfg.Thresholds.MinWeeklyDownloads)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	dir := t.TempDir()
	rc := `{
		"allowlist": ["safe-pkg", "@types/*"],
		"strict": true,
		"fail_on": "warning",
		"thresholds": {
			"min_weekly_downloads": 500
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, ".npm-vetrc"), []byte(rc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir so findConfig finds it
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	cfg := Load()
	if !cfg.Strict {
		t.Error("expected strict=true from config file")
	}
	if cfg.FailOn != "warning" {
		t.Errorf("FailOn = %q, want %q", cfg.FailOn, "warning")
	}
	if len(cfg.Allowlist) != 2 {
		t.Errorf("Allowlist len = %d, want 2", len(cfg.Allowlist))
	}
	if cfg.Thresholds.MinWeeklyDownloads != 500 {
		t.Errorf("MinWeeklyDownloads = %d, want 500", cfg.Thresholds.MinWeeklyDownloads)
	}
}

func TestIsAnalyzerEnabled(t *testing.T) {
	cfg := Default()

	// All enabled by default
	if !cfg.IsAnalyzerEnabled("postinstall") {
		t.Error("postinstall should be enabled")
	}

	// Disable one
	cfg.Analyzers["postinstall"] = false
	if cfg.IsAnalyzerEnabled("postinstall") {
		t.Error("postinstall should be disabled")
	}

	// Unknown analyzer = enabled
	if !cfg.IsAnalyzerEnabled("unknown-analyzer") {
		t.Error("unknown analyzer should default to enabled")
	}
}
