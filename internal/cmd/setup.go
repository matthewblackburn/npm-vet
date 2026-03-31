package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// RunSetup creates a shim directory with an `npm` symlink pointing to npm-vet,
// and prints instructions to add it to PATH. This makes npm-vet transparently
// intercept all `npm install` calls from any tool, script, or agent.
func RunSetup(args []string) int {
	// Determine shim directory
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot determine home directory: %v\n", err)
		return 2
	}
	shimDir := filepath.Join(home, ".npm-vet", "bin")

	// Find our own binary path
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot determine own path: %v\n", err)
		return 2
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot resolve own path: %v\n", err)
		return 2
	}

	// Create shim directory
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot create shim directory: %v\n", err)
		return 2
	}

	shimPath := filepath.Join(shimDir, "npm")

	// Remove existing shim if present
	os.Remove(shimPath)

	// Create symlink: ~/.npm-vet/bin/npm → /path/to/npm-vet
	if err := os.Symlink(self, shimPath); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot create symlink: %v\n", err)
		return 2
	}

	fmt.Fprintf(os.Stderr, "✓ Created shim: %s → %s\n\n", shimPath, self)

	// Check if already in PATH
	pathEnv := os.Getenv("PATH")
	alreadyInPath := false
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == shimDir {
			alreadyInPath = true
			break
		}
	}

	if alreadyInPath {
		fmt.Fprintln(os.Stderr, "✓ Shim directory is already in your PATH.")
		verifySetup(shimDir)
		return 0
	}

	// Detect shell and print appropriate instructions
	shell := detectShell()
	rcFile := shellRCFile(shell, home)
	exportLine := fmt.Sprintf(`export PATH="%s:$PATH"`, shimDir)

	// Check if --apply flag is set
	apply := false
	for _, arg := range args {
		if arg == "--apply" {
			apply = true
		}
	}

	if apply && rcFile != "" {
		if err := appendToFile(rcFile, exportLine); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet setup: cannot update %s: %v\n", rcFile, err)
			fmt.Fprintf(os.Stderr, "\nAdd this line manually to your shell profile:\n\n  %s\n\n", exportLine)
			return 0
		}
		fmt.Fprintf(os.Stderr, "✓ Added PATH entry to %s\n\n", rcFile)
		fmt.Fprintf(os.Stderr, "Run this to activate now (or open a new terminal):\n\n  source %s\n\n", rcFile)
	} else {
		fmt.Fprintln(os.Stderr, "Add the shim directory to your PATH. Either:")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "  Option 1: Run setup with --apply to auto-update your shell profile:\n\n")
		fmt.Fprintf(os.Stderr, "    npm-vet setup --apply\n\n")
		fmt.Fprintf(os.Stderr, "  Option 2: Add this line manually to %s:\n\n", rcFile)
		fmt.Fprintf(os.Stderr, "    %s\n\n", exportLine)
	}

	fmt.Fprintln(os.Stderr, "Once active, ALL npm install calls will be intercepted — including")
	fmt.Fprintln(os.Stderr, "from scripts, CI agents, IDE extensions, and other tools.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To disable: npm-vet teardown")

	return 0
}

// RunTeardown removes the shim and PATH entry.
func RunTeardown(args []string) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet teardown: cannot determine home directory: %v\n", err)
		return 2
	}

	shimDir := filepath.Join(home, ".npm-vet", "bin")
	shimPath := filepath.Join(shimDir, "npm")

	if err := os.Remove(shimPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "npm-vet teardown: cannot remove shim: %v\n", err)
		return 2
	}

	fmt.Fprintln(os.Stderr, "✓ Removed npm shim.")
	fmt.Fprintln(os.Stderr)

	shell := detectShell()
	rcFile := shellRCFile(shell, home)
	if rcFile != "" {
		fmt.Fprintf(os.Stderr, "You may also want to remove the PATH entry from %s:\n\n", rcFile)
		fmt.Fprintf(os.Stderr, "  export PATH=\"%s:$PATH\"\n\n", shimDir)
	}

	return 0
}

func detectShell() string {
	// Check SHELL env var
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell)
	}
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

func shellRCFile(shell, home string) string {
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		// macOS uses .bash_profile, Linux uses .bashrc
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, ".bash_profile")
		}
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return filepath.Join(home, ".profile")
	}
}

func appendToFile(path, line string) error {
	// Check if line already exists
	data, err := os.ReadFile(path)
	if err == nil && strings.Contains(string(data), line) {
		return nil // already present
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n# npm-vet: intercept npm install for security vetting\n%s\n", line)
	return err
}

func verifySetup(shimDir string) {
	// Verify that `which npm` points to our shim
	out, err := exec.Command("which", "npm").Output()
	if err != nil {
		return
	}
	npmPath := strings.TrimSpace(string(out))
	if strings.HasPrefix(npmPath, shimDir) {
		fmt.Fprintln(os.Stderr, "✓ `npm` now routes through npm-vet.")
	} else {
		fmt.Fprintf(os.Stderr, "⚠ `which npm` resolves to %s (not the shim).\n", npmPath)
		fmt.Fprintln(os.Stderr, "  Open a new terminal or source your shell profile.")
	}
}
