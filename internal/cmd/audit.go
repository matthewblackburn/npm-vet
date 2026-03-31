package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/matthewblackburn/npm-vet/internal/analyzer"
	"github.com/matthewblackburn/npm-vet/internal/config"
	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/parser"
	"github.com/matthewblackburn/npm-vet/internal/pipeline"
	"github.com/matthewblackburn/npm-vet/internal/registry"
	"github.com/matthewblackburn/npm-vet/internal/reporter"
)

// RunAudit is the main entry point for audit mode (CI/CD).
// It scans package.json, runs analyzers, and exits with appropriate code.
//
// Exit codes: 0=clean, 1=findings above threshold, 2=error
func RunAudit(args []string, version string) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "Output findings as JSON")
	sarifOutput := fs.Bool("sarif", false, "Output findings as SARIF 2.1.0")
	failOn := fs.String("fail-on", "", "Minimum severity to fail on: critical, warning, info")
	dir := fs.String("dir", ".", "Directory containing package.json")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet audit: %v\n", err)
		return 2
	}

	cfg := config.Load()

	// CLI flags override config
	if *failOn != "" {
		cfg.FailOn = *failOn
	}
	if cfg.FailOn == "" {
		cfg.FailOn = "critical"
	}

	// Read package.json
	packages, err := parser.ReadPackageJSON(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet audit: %v\n", err)
		return 2
	}

	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "npm-vet audit: no dependencies found in package.json")
		return 0
	}

	// Lockfile fast-path: use resolved versions if available
	resolved, _ := parser.ReadLockfile(*dir)
	if resolved != nil {
		for i, pkg := range packages {
			if v, ok := resolved[pkg.Name]; ok {
				packages[i].Version = v
			}
		}
	}

	// Build analyzer list based on config
	var analyzers []analyzer.Analyzer
	for _, a := range analyzer.All() {
		if cfg.IsAnalyzerEnabled(a.Name()) {
			analyzers = append(analyzers, a)
		}
	}

	client := registry.NewClient()

	result, err := pipeline.Run(context.Background(), packages, pipeline.Config{
		Analyzers: analyzers,
		Client:    client,
		Allowlist: cfg.Allowlist,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet audit: %v\n", err)
		return 2
	}

	// Output report
	if *jsonOutput {
		if err := reporter.JSONReport(result.Findings); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet audit: %v\n", err)
			return 2
		}
	} else if *sarifOutput {
		if err := reporter.SARIFReport(result.Findings, version); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet audit: %v\n", err)
			return 2
		}
	} else {
		isCI := os.Getenv("CI") != ""
		reporter.ConsoleReport(result.Findings, !isCI)
	}

	// Determine exit code based on --fail-on threshold
	threshold := npmvet.ParseSeverity(cfg.FailOn)
	for _, f := range result.Findings {
		if f.Severity.AtLeast(threshold) {
			return 1
		}
	}

	return 0
}
