package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

const (
	// DefaultCommandTimeout is the timeout for CLI commands in fuzz tests
	// This prevents hanging when fuzzing creates valid URLs that don't respond
	DefaultCommandTimeout = 10 * time.Second
)

// RunCommand executes the CLI binary with given arguments and returns output
// It uses a timeout to prevent hanging during fuzz tests
func RunCommand(bin string, args ...string) (string, error) {
	return RunCommandWithTimeout(bin, DefaultCommandTimeout, args...)
}

// RunCommandWithTimeout executes the CLI binary with a specific timeout
func RunCommandWithTimeout(bin string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if timeout was exceeded
		if ctx.Err() == context.DeadlineExceeded {
			return stderr.String(), context.DeadlineExceeded
		}
		return stderr.String(), err
	}
	return out.String(), nil
}
