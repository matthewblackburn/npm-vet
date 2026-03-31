//go:build windows

package delegator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecNpm runs the real npm binary with the given arguments.
// On Windows, we can't replace the process, so we spawn npm as a child
// and forward the exit code.
func ExecNpm(args []string) error {
	npmPath, err := FindRealNpm()
	if err != nil {
		return err
	}

	cmd := exec.Command(npmPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil // unreachable
}

// FindRealNpm locates the real npm binary, skipping our own binary.
func FindRealNpm() (string, error) {
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

	pathEnv := os.Getenv("PATH")
	for _, dir := range strings.Split(pathEnv, ";") {
		for _, name := range []string{"npm.cmd", "npm.exe", "npm"} {
			candidate := filepath.Join(dir, name)
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				continue
			}

			resolved, err := filepath.EvalSymlinks(candidate)
			if err != nil {
				resolved = candidate
			}
			if selfPath != "" && resolved == selfPath {
				continue
			}

			return candidate, nil
		}
	}

	if path, err := exec.LookPath("npm"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("npm not found in PATH. Set NPM_VET_NPM_PATH to the real npm binary")
}
