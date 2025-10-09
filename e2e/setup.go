package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// PrepareCLI sets up the test environment and returns the CLI binary path
func PrepareCLI() (cliPath string, cleanup func(), err error) {
	fmt.Println("[e2e] Loading the environment")
	if err := LoadEnv(); err != nil {
		return "", nil, err
	}
	fmt.Println("[e2e] Loading environment completed")

	fmt.Println("[e2e] Trying to setup token file")
	tokenPath, err := SetupTokenFile()
	if err != nil {
		return "", nil, err
	}
	fmt.Println("[e2e] Setting up the token file completed successfully")

	if err := os.Setenv("MUMP2P_AUTH_PATH", tokenPath); err != nil {
		return "", nil, fmt.Errorf("failed to set MUMP2P_AUTH_PATH: %w", err)
	}

	fmt.Println("[e2e] Trying to find repo root")
	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", nil, err
	}

	cli := os.Getenv("MUMP2P_E2E_CLI_BINARY")
	if cli == "" {
		osName := runtime.GOOS
		if osName == "darwin" {
			osName = "mac"
		}
		cli = filepath.Join(repoRoot, "dist", fmt.Sprintf("mump2p-%s", osName))
	} else if !filepath.IsAbs(cli) {
		cli = filepath.Join(repoRoot, cli)
	}

	if _, err := os.Stat(cli); errors.Is(err, os.ErrNotExist) {
		return "", nil, fmt.Errorf("binary not found at %s\nRun 'make build' first with release credentials", cli)
	}

	stat, err := os.Stat(cli)
	if err != nil {
		return "", nil, fmt.Errorf("failed to stat CLI binary: %w", err)
	}
	if stat.Mode()&0111 == 0 {
		return "", nil, fmt.Errorf("binary %s is not executable", cli)
	}

	cleanup = func() {
		_ = os.RemoveAll(filepath.Dir(tokenPath))
	}

	return cli, cleanup, nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not locate repo root from %s", wd)
}
