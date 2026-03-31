package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the tool configuration from .npm-vetrc and CLI flags.
type Config struct {
	Allowlist  []string            `json:"allowlist"`
	Strict     bool                `json:"strict"`
	Mode       string              `json:"mode"`    // "prompt" (default), "block", or "report"
	FailOn     string              `json:"fail_on"` // "critical", "warning", "info"
	Analyzers  map[string]bool     `json:"analyzers"`
	Thresholds Thresholds          `json:"thresholds"`
}

// Mode constants
const (
	ModePrompt = "prompt" // Show findings, ask user (default). Agents can bypass.
	ModeBlock  = "block"  // Hard exit 1 on findings >= fail_on. No prompt. No override.
	ModeReport = "report" // Show findings but always proceed (log-only).
)

// Thresholds configures analyzer-specific thresholds.
type Thresholds struct {
	MinWeeklyDownloads int `json:"min_weekly_downloads"`
	MaxTyposquatDist   int `json:"max_typosquat_distance"`
	MaxStringLength    int `json:"max_string_length"`
	MaxTarballSizeMB   int `json:"max_tarball_size_mb"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Mode:   ModePrompt,
		FailOn: "critical",
		Analyzers: map[string]bool{
			"postinstall": true,
			"obfuscation": true,
			"network":     true,
			"maintainer":  true,
			"typosquat":   true,
			"downloads":   true,
		},
		Thresholds: Thresholds{
			MinWeeklyDownloads: 100,
			MaxTyposquatDist:   2,
			MaxStringLength:    1000,
			MaxTarballSizeMB:   5,
		},
	}
}

// Load searches for a .npm-vetrc config file and merges it with defaults.
// Search order: cwd → walk up to root → ~/.npm-vetrc → defaults.
func Load() Config {
	cfg := Default()

	// Try to find .npm-vetrc starting from cwd
	if path := findConfig(); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &cfg)
		}
	}

	return cfg
}

// findConfig walks up from cwd looking for .npm-vetrc, then falls back to home dir.
func findConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return homeConfig()
	}

	for {
		path := filepath.Join(dir, ".npm-vetrc")
		if _, err := os.Stat(path); err == nil {
			return path
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return homeConfig()
}

func homeConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".npm-vetrc")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

// EffectiveMode returns the mode to use, checking the NPM_VET_MODE env var first.
func (c Config) EffectiveMode() string {
	if env := os.Getenv("NPM_VET_MODE"); env != "" {
		switch env {
		case ModeBlock, ModePrompt, ModeReport:
			return env
		}
	}
	if c.Mode != "" {
		return c.Mode
	}
	return ModePrompt
}

// IsAnalyzerEnabled returns whether an analyzer is enabled in the config.
func (c Config) IsAnalyzerEnabled(name string) bool {
	if c.Analyzers == nil {
		return true // all enabled by default
	}
	enabled, ok := c.Analyzers[name]
	if !ok {
		return true // not mentioned = enabled
	}
	return enabled
}
