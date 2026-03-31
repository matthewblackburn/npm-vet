package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/matthewblackburn/npm-vet/internal/analyzer"
	"github.com/matthewblackburn/npm-vet/internal/config"
	"github.com/matthewblackburn/npm-vet/internal/delegator"
	"github.com/matthewblackburn/npm-vet/internal/npmvet"
	"github.com/matthewblackburn/npm-vet/internal/parser"
	"github.com/matthewblackburn/npm-vet/internal/pipeline"
	"github.com/matthewblackburn/npm-vet/internal/registry"
	"github.com/matthewblackburn/npm-vet/internal/reporter"
)

// RunWrapper is the main entry point for wrapper mode.
// It parses npm args, vets packages if it's an install command,
// and delegates to the real npm.
func RunWrapper(args []string) int {
	parsed := parser.ParseArgs(args)

	// Non-install commands: passthrough immediately
	if !parsed.IsInstall {
		if err := delegator.ExecNpm(args); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
			return 2
		}
		return 0
	}

	cfg := config.Load()

	// Resolve packages to vet
	var packages []npmvet.PackageSpec

	if len(parsed.Packages) > 0 {
		// Named install: convert CLI specifiers to PackageSpecs
		for _, spec := range parsed.Packages {
			name, versionRange := parser.SplitPackageSpec(spec)
			packages = append(packages, npmvet.PackageSpec{
				Name:         name,
				VersionRange: versionRange,
			})
		}
	} else {
		// Bare install or ci: read from package.json
		dir, _ := os.Getwd()
		pkgs, err := parser.ReadPackageJSON(dir)
		if err != nil {
			// Can't read package.json — just passthrough to npm (it'll handle the error)
			if err := delegator.ExecNpm(args); err != nil {
				fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
				return 2
			}
			return 0
		}
		packages = pkgs

		// Lockfile fast-path
		resolved, _ := parser.ReadLockfile(dir)
		if resolved != nil {
			for i, pkg := range packages {
				if v, ok := resolved[pkg.Name]; ok {
					packages[i].Version = v
				}
			}
		}
	}

	if len(packages) == 0 {
		if err := delegator.ExecNpm(args); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
			return 2
		}
		return 0
	}

	fmt.Fprintf(os.Stderr, "npm-vet: vetting %d package(s)...\n", len(packages))

	// Build analyzer list
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
		fmt.Fprintf(os.Stderr, "npm-vet: error during vetting: %v\n", err)
		// On error, still allow install to proceed
		if err := delegator.ExecNpm(args); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
			return 2
		}
		return 0
	}

	// No findings: proceed
	if len(result.Findings) == 0 {
		fmt.Fprintln(os.Stderr, "npm-vet: ✓ No security concerns found.")
		if err := delegator.ExecNpm(args); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
			return 2
		}
		return 0
	}

	// Report findings
	reporter.ConsoleReport(result.Findings, true)

	// Check if any finding meets the fail_on threshold
	threshold := npmvet.ParseSeverity(cfg.FailOn)
	exceedsThreshold := false
	for _, f := range result.Findings {
		if f.Severity.AtLeast(threshold) {
			exceedsThreshold = true
			break
		}
	}

	mode := cfg.EffectiveMode()

	switch mode {
	case config.ModeBlock:
		// Hard block: no prompt, no override. Agents cannot bypass this.
		if exceedsThreshold {
			fmt.Fprintf(os.Stderr, "npm-vet: BLOCKED — findings at or above %q severity. Installation refused.\n", cfg.FailOn)
			fmt.Fprintln(os.Stderr, "npm-vet: To allow this install, add the package to your .npm-vetrc allowlist.")
			return 1
		}
		// Below threshold: proceed silently
	case config.ModeReport:
		// Report only: always proceed after showing findings
		fmt.Fprintln(os.Stderr, "npm-vet: (report mode) proceeding with install.")
	default: // ModePrompt
		if exceedsThreshold {
			// Default to No when findings exceed threshold
			if !reporter.PromptContinue(false) {
				fmt.Fprintln(os.Stderr, "npm-vet: installation aborted.")
				return 1
			}
		} else {
			// Below threshold but still has findings — default to Yes
			if !reporter.PromptContinue(true) {
				fmt.Fprintln(os.Stderr, "npm-vet: installation aborted.")
				return 1
			}
		}
	}

	if err := delegator.ExecNpm(args); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet: %v\n", err)
		return 2
	}
	return 0
}
