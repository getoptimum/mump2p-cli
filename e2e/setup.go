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
	os.Setenv("MUMP2P_AUTH_PATH", tokenPath)

	cli := os.Getenv("MUMP2P_E2E_CLI_BINARY")
	if cli == "" {
		switch runtime.GOOS {
		case "linux":
			cli = "dist/mump2p-linux"
		case "darwin":
			cli = "dist/mump2p-mac"
		default:
			return "", nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	}

	cli = filepath.Join("..", "dist", fmt.Sprintf("mump2p-%s", runtime.GOOS))
	abs, err := filepath.Abs(cli)

	if err != nil {
		return "", nil, err
	}
	cli = abs

	if _, err := os.Stat(cli); os.IsNotExist(err) {
		fmt.Println("[e2e] CLI not found, building via make build-local...")
		//cmd := exec.Command("make", "build-local")
		cmd := exec.Command("make", "-C", "..", "build-local")
		cmd.Env = injectBuildEnv()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("build failed: %w", err)
		}
	}

	stat, err := os.Stat(cli)
	if err != nil || stat.Mode()&0111 == 0 {
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
