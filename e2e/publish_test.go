package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishCommand(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()

	testTopic := fmt.Sprintf("test-publish-%d", time.Now().Unix())

	// Start a subscriber in the background to enable publishing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	require.NoError(t, subCmd.Start(), "Failed to start background subscriber")

	// Wait for subscription to be active
	time.Sleep(2 * time.Second)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   []string
	}{
		{
			name:        "publish HTTP inline message",
			args:        []string{"publish", "--topic=" + testTopic, "--message=Hello E2E Test", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"published", "topic"},
		},
		{
			name:        "publish gRPC inline message",
			args:        []string{"publish", "--topic=" + testTopic, "--message=Hello gRPC Test", "--grpc", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"published"},
		},
		{
			name:        "publish with debug mode HTTP",
			args:        []string{"--debug", "publish", "--topic=" + testTopic, "--message=Debug test", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"publish:", "sender_info:", "topic:"},
		},
		{
			name:        "publish with debug mode gRPC",
			args:        []string{"--debug", "publish", "--topic=" + testTopic, "--message=Debug gRPC", "--grpc", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"publish:", "sender_info:", "topic:"},
		},
		{
			name:        "publish missing topic flag",
			args:        []string{"publish", "--message=test"},
			expectError: true,
			expectOut:   []string{},
		},
		{
			name:        "publish missing message flag",
			args:        []string{"publish", "--topic=" + testTopic},
			expectError: true,
			expectOut:   []string{},
		},
		{
			name:        "publish with invalid service-url",
			args:        []string{"publish", "--topic=" + testTopic, "--message=test", "--service-url=invalid-url"},
			expectError: true,
			expectOut:   []string{},
		},
	}

	// Run the basic tests first
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, tt.args...)

			if tt.expectError {
				require.Error(t, err, "Expected command to fail but it succeeded. Output: %s", out)
			} else {
				require.NoError(t, err, "Command failed: %v\nOutput: %s", err, out)

				// Strict validation for publish success
				validator := NewValidator(out)
				err := validator.ValidatePublishSuccess()
				require.NoError(t, err, "Publish validation failed: %v", err)
			}
		})
	}

	// Test --file flag scenarios
	t.Run("publish from file HTTP", func(t *testing.T) {
		dir := t.TempDir()
		testFile := filepath.Join(dir, "test-publish.txt")
		testContent := "Test file content for HTTP publish"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err, "Failed to create test file")

		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--file="+testFile,
			"--service-url="+serviceURL)
		require.NoError(t, err, "File publish failed: %v\nOutput: %s", err, out)

		validator := NewValidator(out)
		err = validator.ValidatePublishSuccess()
		require.NoError(t, err, "File publish validation failed")
	})

	t.Run("publish from file gRPC", func(t *testing.T) {
		dir := t.TempDir()
		testFile := filepath.Join(dir, "test-publish-grpc.txt")
		testContent := "Test file content for gRPC publish"
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		require.NoError(t, err, "Failed to create test file")

		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--file="+testFile,
			"--grpc",
			"--service-url="+serviceURL)
		require.NoError(t, err, "File gRPC publish failed: %v\nOutput: %s", err, out)

		validator := NewValidator(out)
		err = validator.ValidatePublishSuccess()
		require.NoError(t, err, "File gRPC publish validation failed")
	})

	t.Run("publish file not found", func(t *testing.T) {
		dir := t.TempDir()
		nonExistentFile := filepath.Join(dir, "nonexistent-file.txt")
		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--file="+nonExistentFile,
			"--service-url="+serviceURL)
		require.Error(t, err, "Expected file not found error. Output: %s", out)
		require.Contains(t, strings.ToLower(out), "failed to read file", "Expected file read error")
	})

	t.Run("publish file and message both (should fail)", func(t *testing.T) {
		dir := t.TempDir()
		testFile := filepath.Join(dir, "test-publish-both.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err, "Failed to create test file")

		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--file="+testFile,
			"--message=test",
			"--service-url="+serviceURL)
		require.Error(t, err, "Expected error when both --file and --message are provided. Output: %s", out)
		require.Contains(t, strings.ToLower(out), "only one", "Expected error about using only one option")
	})

	// Cleanup: stop subscriber
	cancel()
	subCmd.Wait()
}
