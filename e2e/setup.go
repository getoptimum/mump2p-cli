package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func PrepareCLI() (cliPath string, cleanup func(), err error) {
	fmt.Println("[e2e] Loading the environment")
	if err := LoadEnv(); err != nil {
		return "", nil, err
	}
	fmt.Println("[e2e] Loading environment completed")

	tokenPath, err := SetupTokenFile()
	fmt.Println("[e2e] Setting up the token file completed")

	if err != nil {
		return "", nil, err
	}
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
		cli = filepath.Join(repoRoot, "dist", fmt.Sprintf("mump2p-%s", runtime.GOOS))
	} else if !filepath.IsAbs(cli) {
		// Treat relative paths as repo-root relative so the command works when
		// executed from the e2e package directory or the repo root.
		cli = filepath.Join(repoRoot, cli)
	}

	if _, err := os.Stat(cli); errors.Is(err, os.ErrNotExist) {
		fmt.Println("[e2e] CLI not found, building via make build-local...")
		cmd := exec.Command("make", "build-local")
		cmd.Dir = repoRoot
		cmd.Env = injectBuildEnv()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("build failed: %w", err)
		}
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

func injectBuildEnv() []string {
	env := os.Environ()
	for key, value := range map[string]string{
		"DOMAIN":      os.Getenv("AUTH_DOMAIN"),
		"CLIENT_ID":   os.Getenv("AUTH_CLIENT_ID"),
		"AUDIENCE":    os.Getenv("AUTH_AUDIENCE"),
		"SERVICE_URL": os.Getenv("SERVICE_URL"),
	} {
		if value != "" {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return env
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
