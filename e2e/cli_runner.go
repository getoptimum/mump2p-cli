package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func RunE2ETests() error {
	if err := LoadEnv(); err != nil {
		return err
	}

	tokenPath, err := SetupTokenFile()
	if err != nil {
		return err
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
			return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	}

	// Normalize to absolute path
	abs, err := filepath.Abs(cli)
	if err != nil {
		return fmt.Errorf("failed to resolve CLI path %q: %w", cli, err)
	}
	cli = abs

	// If binary missing try to build it
	if _, err := os.Stat(cli); err != nil {
		fmt.Println("[e2e] CLI binary not found, attempting to build")
		cmd := exec.Command("make", "build-local")
		cmd.Env = append(os.Environ(),
			"DOMAIN="+os.Getenv("AUTH_DOMAIN"),
			"CLIENT_ID="+os.Getenv("AUTH_CLIENT_ID"),
			"AUDIENCE="+os.Getenv("AUTH_AUDIENCE"),
			"SERVICE_URL="+os.Getenv("SERVICE_URL"),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("make build-local failed: %w", err)
		}
	}

	// Final existence + exec check
	stat, err := os.Stat(cli)
	if err != nil {
		return fmt.Errorf("binary %s still missing after build: %w", cli, err)
	}
	if stat.Mode()&0111 == 0 {
		return fmt.Errorf("binary %s is not executable", cli)
	}

	tests := []struct {
		name string
		args []string
	}{
		{"health", []string{"health", "--service-url=" + os.Getenv("SERVICE_URL")}},
		{"whoami", []string{"whoami"}},
		//{"subscribe", []string{
		//	"subscribe",
		//	"--topic=" + getTopic(),
		//	"--service-url=" + os.Getenv("SERVICE_URL"),
		//}},
		//
		//{"publish", []string{
		//	"publish",
		//	"--topic=" + getTopic(),
		//	"--message=" + getMessage(),
		//	"--service-url=" + os.Getenv("SERVICE_URL"),
		//}},
		{"list-topics", []string{"list-topics", "--service-url=" + os.Getenv("SERVICE_URL")}},
	}

	for _, test := range tests {
		fmt.Printf("[e2e] running %s\n", test.name)
		output, err := RunCommand(cli, test.args...)
		if err != nil {
			return fmt.Errorf("test %s failed: %w\nOutput: %s", test.name, err, output)
		}
	}

	return nil
}

func RunCommand(bin string, args ...string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	return out.String(), nil
}

func getTopic() string {
	topic := os.Getenv("MUMP2P_E2E_TOPIC")
	if topic == "" {
		topic = fmt.Sprintf("optimum-e2e-%d", time.Now().Unix())
	}
	return topic
}

func getMessage() string {
	msg := os.Getenv("MUMP2P_E2E_MESSAGE")
	if msg == "" {
		msg = fmt.Sprintf("hello from go e2e at %s", time.Now().UTC().Format(time.RFC3339))
	}
	return msg
}
