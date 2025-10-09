package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribeCommand(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://34.146.222.111:8080"
	}

	testTopic := fmt.Sprintf("test-sub-%d", time.Now().Unix())

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   []string
		timeout     time.Duration
	}{
		{
			name:        "subscribe WebSocket",
			args:        []string{"subscribe", "--topic=" + testTopic, "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"subscription", testTopic},
			timeout:     5 * time.Second,
		},
		{
			name:        "subscribe gRPC",
			args:        []string{"subscribe", "--topic=" + testTopic, "--grpc", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{"subscription", testTopic},
			timeout:     5 * time.Second,
		},
		{
			name:        "subscribe with debug WebSocket",
			args:        []string{"--debug", "subscribe", "--topic=" + testTopic, "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{testTopic},
			timeout:     5 * time.Second,
		},
		{
			name:        "subscribe with debug gRPC",
			args:        []string{"--debug", "subscribe", "--topic=" + testTopic, "--grpc", "--service-url=" + serviceURL},
			expectError: false,
			expectOut:   []string{testTopic},
			timeout:     5 * time.Second,
		},
		{
			name:        "subscribe missing topic flag",
			args:        []string{"subscribe"},
			expectError: true,
			expectOut:   []string{},
			timeout:     1 * time.Second,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, cliBinaryPath, tt.args...)
			cmd.Env = os.Environ()

			output, err := cmd.CombinedOutput()
			out := string(output)

			if tt.expectError {
				require.Error(t, err, "Expected command to fail but it succeeded. Output: %s", out)
			} else {
				// For subscribe, context deadline exceeded is expected (we kill it after timeout)
				if ctx.Err() == context.DeadlineExceeded {
					// This is OK - we just wanted to test connection
					for _, expected := range tt.expectOut {
						require.Contains(t, strings.ToLower(out), strings.ToLower(expected),
							"Expected output to contain %q, got %q", expected, out)
					}
				} else if err != nil {
					t.Logf("Subscribe command ended early: %v\nOutput: %s", err, out)
					// Still check if we got expected output before exit
					for _, expected := range tt.expectOut {
						require.Contains(t, strings.ToLower(out), strings.ToLower(expected),
							"Expected output to contain %q, got %q", expected, out)
					}
				}
			}
		})
	}
}
