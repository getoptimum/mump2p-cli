package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func PrepareCLI() (cliPath string, cleanup func(), err error) {
	if err := LoadEnv(); err != nil {
		return "", nil, err
	}

	tokenPath, err := SetupTokenFile()
	if err != nil {
		return "", nil, err
	}
	if err := os.Setenv("MUMP2P_AUTH_PATH", tokenPath); err != nil {
		return "", nil, fmt.Errorf("failed to set MUMP2P_AUTH_PATH: %w", err)
	}

	cli := os.Getenv("MUMP2P_E2E_CLI_BINARY")
	if cli == "" {
		cli = filepath.Join("..", "dist", fmt.Sprintf("mump2p-%s", runtime.GOOS))
	} else if !filepath.IsAbs(cli) {
		// Treat relative paths as repo-root relative so the command works when
		// executed from the e2e package directory.
		cli = filepath.Join("..", cli)
	}

	abs, err := filepath.Abs(cli)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve CLI path %q: %w", cli, err)
	}
	cli = abs

	if _, err := os.Stat(cli); os.IsNotExist(err) {
		fmt.Println("[e2e] CLI not found, building via make build-local...")
		cmd := exec.Command("make", "-C", "..", "build-local")
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
