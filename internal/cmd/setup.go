package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// RunSetup creates a shim directory with an `npm` shim pointing to npm-vet,
// and prints instructions to add it to PATH. This makes npm-vet transparently
// intercept all `npm install` calls from any tool, script, or agent.
func RunSetup(args []string) int {
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

	// Create platform-specific shim
	if runtime.GOOS == "windows" {
		return setupWindows(args, shimDir, self, home)
	}
	return setupUnix(args, shimDir, self, home)
}

// setupUnix creates a symlink shim and updates the shell profile.
func setupUnix(args []string, shimDir, self, home string) int {
	shimPath := filepath.Join(shimDir, "npm")

	os.Remove(shimPath)

	if err := os.Symlink(self, shimPath); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot create symlink: %v\n", err)
		return 2
	}

	fmt.Fprintf(os.Stderr, "✓ Created shim: %s → %s\n\n", shimPath, self)

	if isInPath(shimDir) {
		fmt.Fprintln(os.Stderr, "✓ Shim directory is already in your PATH.")
		verifySetupUnix(shimDir)
		return 0
	}

	shell := detectShell()
	rcFile := shellRCFile(shell, home)
	exportLine := fmt.Sprintf(`export PATH="%s:$PATH"`, shimDir)

	if hasFlag(args, "--apply") && rcFile != "" {
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

	printFooter()
	return 0
}

// setupWindows creates a batch script shim and updates the PATH via PowerShell profile or registry.
func setupWindows(args []string, shimDir, self, home string) int {
	// Create npm.cmd that forwards to npm-vet.exe
	npmCmd := filepath.Join(shimDir, "npm.cmd")
	batchContent := fmt.Sprintf("@echo off\r\n\"%s\" %%*\r\n", self)

	if err := os.WriteFile(npmCmd, []byte(batchContent), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: cannot create npm.cmd: %v\n", err)
		return 2
	}

	// Also create npm.ps1 for PowerShell direct invocation
	npmPS1 := filepath.Join(shimDir, "npm.ps1")
	ps1Content := fmt.Sprintf("& \"%s\" @args\r\n", self)

	if err := os.WriteFile(npmPS1, []byte(ps1Content), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet setup: warning: could not create npm.ps1: %v\n", err)
		// Non-fatal — npm.cmd is sufficient
	}

	fmt.Fprintf(os.Stderr, "✓ Created shim: %s → %s\n", npmCmd, self)
	fmt.Fprintln(os.Stderr)

	if isInPath(shimDir) {
		fmt.Fprintln(os.Stderr, "✓ Shim directory is already in your PATH.")
		verifySetupWindows(shimDir)
		return 0
	}

	if hasFlag(args, "--apply") {
		// Update user PATH via PowerShell (persistent, user-level)
		psCmd := fmt.Sprintf(
			`[Environment]::SetEnvironmentVariable('Path', '%s;' + [Environment]::GetEnvironmentVariable('Path', 'User'), 'User')`,
			shimDir,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "npm-vet setup: cannot update PATH via PowerShell: %v\n", err)
			printWindowsManualInstructions(shimDir)
			return 0
		}

		fmt.Fprintf(os.Stderr, "✓ Added %s to your user PATH.\n\n", shimDir)
		fmt.Fprintln(os.Stderr, "Open a new terminal window to activate.")
		fmt.Fprintln(os.Stderr)

		// Also update the PowerShell profile for the current session pattern
		psProfile := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		profileLine := fmt.Sprintf(`$env:Path = "%s;" + $env:Path`, shimDir)
		appendToFile(psProfile, profileLine)
	} else {
		printWindowsManualInstructions(shimDir)
	}

	printFooter()
	return 0
}

func printWindowsManualInstructions(shimDir string) {
	fmt.Fprintln(os.Stderr, "Add the shim directory to your PATH. Either:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  Option 1: Run setup with --apply to auto-update your PATH:\n\n")
	fmt.Fprintf(os.Stderr, "    npm-vet setup --apply\n\n")
	fmt.Fprintf(os.Stderr, "  Option 2: Add manually via PowerShell (run as admin):\n\n")
	fmt.Fprintf(os.Stderr, "    [Environment]::SetEnvironmentVariable('Path', '%s;' + $env:Path, 'User')\n\n", shimDir)
	fmt.Fprintf(os.Stderr, "  Option 3: System Settings → Environment Variables → add %s to Path\n\n", shimDir)
}

func printFooter() {
	fmt.Fprintln(os.Stderr, "Once active, ALL npm install calls will be intercepted — including")
	fmt.Fprintln(os.Stderr, "from scripts, CI agents, IDE extensions, and other tools.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To disable: npm-vet teardown")
}

// RunTeardown removes the shim and PATH entry.
func RunTeardown(args []string) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "npm-vet teardown: cannot determine home directory: %v\n", err)
		return 2
	}

	shimDir := filepath.Join(home, ".npm-vet", "bin")

	if runtime.GOOS == "windows" {
		return teardownWindows(shimDir, home)
	}
	return teardownUnix(shimDir, home)
}

func teardownUnix(shimDir, home string) int {
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

func teardownWindows(shimDir, home string) int {
	// Remove npm.cmd and npm.ps1
	for _, name := range []string{"npm.cmd", "npm.ps1"} {
		path := filepath.Join(shimDir, name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "npm-vet teardown: cannot remove %s: %v\n", name, err)
		}
	}

	fmt.Fprintln(os.Stderr, "✓ Removed npm shim.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "You may also want to remove %s from your user PATH:\n\n", shimDir)
	fmt.Fprintf(os.Stderr, "  PowerShell:\n")
	fmt.Fprintf(os.Stderr, "    $path = [Environment]::GetEnvironmentVariable('Path', 'User') -replace [regex]::Escape('%s;'), ''\n", shimDir)
	fmt.Fprintf(os.Stderr, "    [Environment]::SetEnvironmentVariable('Path', $path, 'User')\n\n")
	fmt.Fprintf(os.Stderr, "  Or: System Settings → Environment Variables → remove %s from Path\n\n", shimDir)

	return 0
}

func detectShell() string {
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

func isInPath(dir string) bool {
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d == dir {
			return true
		}
	}
	return false
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func appendToFile(path, line string) error {
	data, err := os.ReadFile(path)
	if err == nil && strings.Contains(string(data), line) {
		return nil
	}

	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0o755)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n# npm-vet: intercept npm install for security vetting\n%s\n", line)
	return err
}

func verifySetupUnix(shimDir string) {
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

func verifySetupWindows(shimDir string) {
	out, err := exec.Command("where", "npm").Output()
	if err != nil {
		return
	}
	// `where npm` returns all matches, one per line. Check if ours is first.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), shimDir) {
		fmt.Fprintln(os.Stderr, "✓ `npm` now routes through npm-vet.")
	} else {
		fmt.Fprintf(os.Stderr, "⚠ `where npm` resolves to %s (not the shim).\n", strings.TrimSpace(lines[0]))
		fmt.Fprintln(os.Stderr, "  Open a new terminal to activate, or check PATH order.")
	}
}
