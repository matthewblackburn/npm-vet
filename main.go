package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matthewblackburn/npm-vet/internal/cmd"
)

var version = "dev"

func main() {
	args := os.Args[1:]

	// Detect if invoked as "npm-vet" (directly) or "npm" (via shim/symlink).
	// When invoked as "npm", the subcommands (audit, setup, etc.) are accessed
	// via "npm vet audit", "npm vet setup", etc. to avoid colliding with real
	// npm subcommands. When invoked as "npm-vet", they're top-level.
	invokedAs := filepath.Base(os.Args[0])
	isShim := invokedAs == "npm"

	if isShim {
		// Invoked as "npm" via shim.
		// "npm vet ..." routes to npm-vet subcommands
		// everything else goes through wrapper mode (intercept installs, passthrough rest)
		if len(args) > 0 && args[0] == "vet" {
			handleSubcommand(args[1:])
			return
		}
		os.Exit(cmd.RunWrapper(args))
	}

	// Invoked as "npm-vet" directly
	if len(args) > 0 {
		handleSubcommand(args)
		return
	}

	// No args: show usage
	printUsage()
}

func handleSubcommand(args []string) {
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "audit":
		os.Exit(cmd.RunAudit(args[1:], version))
	case "setup":
		os.Exit(cmd.RunSetup(args[1:]))
	case "teardown":
		os.Exit(cmd.RunTeardown(args[1:]))
	case "version", "--version", "-v":
		fmt.Printf("npm-vet %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		// If invoked as npm-vet with npm-style args, treat as wrapper
		os.Exit(cmd.RunWrapper(args))
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `npm-vet %s — Security vetting for npm packages

Usage:
  npm-vet setup [--apply]        Install npm shim to intercept all npm install calls
  npm-vet teardown               Remove the npm shim
  npm-vet audit [flags]          Scan package.json (CI/CD mode)
  npm-vet install <pkg> ...      Vet packages then install via npm (wrapper mode)
  npm-vet <any npm command>      Pass through to real npm (with install interception)

When installed via 'npm-vet setup', all npm commands are transparently intercepted:
  npm install express            Vetted automatically before install
  npm test                       Passed through to real npm unchanged

Audit flags:
  --json                         Output findings as JSON
  --sarif                        Output findings as SARIF 2.1.0
  --fail-on=<severity>           Exit 1 if findings at this level: critical, warning, info
  --dir=<path>                   Directory containing package.json (default: .)

`, version)
}
