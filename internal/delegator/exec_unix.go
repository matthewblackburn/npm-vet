//go:build !windows

package delegator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ExecNpm replaces the current process with the real npm binary, passing
// through the given arguments. This function does not return on success.
func ExecNpm(args []string) error {
	npmPath, err := FindRealNpm()
	if err != nil {
		return err
	}

	// Build argv: npm + original args
	argv := append([]string{"npm"}, args...)

	return syscall.Exec(npmPath, argv, os.Environ())
}

// FindRealNpm locates the real npm binary, skipping our own binary.
func FindRealNpm() (string, error) {
	// Check env var override first
	if envPath := os.Getenv("NPM_VET_NPM_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("NPM_VET_NPM_PATH=%q does not exist", envPath)
	}

	selfPath, _ := os.Executable()
	if selfPath != "" {
		selfPath, _ = filepath.EvalSymlinks(selfPath)
	}

	// Scan PATH for npm, skipping our own binary
	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, ":") {
		candidate := filepath.Join(dir, "npm")
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}

		// Resolve symlinks to check if it's us
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			resolved = candidate
		}
		if selfPath != "" && resolved == selfPath {
			continue
		}

		// Verify it's executable
		if err := isExecutable(candidate); err != nil {
			continue
		}

		return candidate, nil
	}

	// Fallback: use exec.LookPath (which might find us, but it's a last resort)
	if path, err := exec.LookPath("npm"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("npm not found in PATH. Set NPM_VET_NPM_PATH to the real npm binary")
}

func isExecutable(path string) error {
	return syscall.Access(path, 0x1) // X_OK
}
