package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishCommand(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()

	testTopic := fmt.Sprintf("test-publish-%d", time.Now().Unix())

	// Start a subscriber in the background to enable publishing
	t.Log("Starting background subscriber to enable publishing...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	require.NoError(t, subCmd.Start(), "Failed to start background subscriber")

	// Wait for subscription to be active
	time.Sleep(2 * time.Second)
	t.Log("Subscriber active, proceeding with publish tests...")

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

	for _, tt := range tests {
		tt := tt
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

	// Cleanup: stop subscriber
	cancel()
	subCmd.Wait()
	t.Log("Background subscriber stopped")
}
