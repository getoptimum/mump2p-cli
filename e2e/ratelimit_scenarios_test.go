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

// TestRateLimiterScenarios validates that usage stats change correctly after publishing messages
func TestRateLimiterScenarios(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-%d", time.Now().Unix())

	// Get initial usage stats
	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get initial usage stats")

	validatorBefore := NewValidator(usageBefore)
	usageInfoBefore, err := validatorBefore.ValidateUsage()
	require.NoError(t, err, "Failed to parse initial usage stats")

	beforeCount := parsePublishCount(usageInfoBefore.PublishCount)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err = subCmd.Start()
	require.NoError(t, err, "Failed to start subscriber")
	time.Sleep(2 * time.Second)

	// Publish multiple messages
	numMessages := 3
	for i := 0; i < numMessages; i++ {
		msg := fmt.Sprintf("RateLimitTest-%d", i+1)
		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--message="+msg,
			"--service-url="+serviceURL)
		require.NoError(t, err, "Publish %d failed: %s", i+1, out)
		time.Sleep(500 * time.Millisecond) // Small delay between publishes
	}

	cancel()
	subCmd.Wait()
	time.Sleep(1 * time.Second)

	// Get usage stats after publishing
	usageAfter, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats after publishing")

	validatorAfter := NewValidator(usageAfter)
	usageInfoAfter, err := validatorAfter.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats after publishing")

	afterCount := parsePublishCount(usageInfoAfter.PublishCount)

	// Verify publish count increased
	require.GreaterOrEqual(t, afterCount, beforeCount+numMessages,
		"Publish count should increase by at least %d (before: %d, after: %d)",
		numMessages, beforeCount, afterCount)

	// Verify data usage is present
	require.Contains(t, usageAfter, "Data Used:", "Usage stats should show data usage")
}

// TestRateLimiterWithGRPC validates usage tracking with gRPC protocol
func TestRateLimiterWithGRPC(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-grpc-%d", time.Now().Unix())

	// Get initial usage stats
	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get initial usage stats")

	validatorBefore := NewValidator(usageBefore)
	usageInfoBefore, err := validatorBefore.ValidateUsage()
	require.NoError(t, err, "Failed to parse initial usage stats")

	beforeCount := parsePublishCount(usageInfoBefore.PublishCount)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--grpc", "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err = subCmd.Start()
	require.NoError(t, err, "Failed to start subscriber")
	time.Sleep(2 * time.Second)

	// Publish via gRPC
	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--message=RateLimitGRPCTest",
		"--grpc",
		"--service-url="+serviceURL)
	require.NoError(t, err, "gRPC publish failed: %s", out)

	cancel()
	subCmd.Wait()
	time.Sleep(1 * time.Second)

	// Get usage stats after publishing
	usageAfter, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats after publishing")

	validatorAfter := NewValidator(usageAfter)
	usageInfoAfter, err := validatorAfter.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats after publishing")

	afterCount := parsePublishCount(usageInfoAfter.PublishCount)

	// Verify publish count increased
	require.GreaterOrEqual(t, afterCount, beforeCount+1,
		"Publish count should increase by at least 1 (before: %d, after: %d)",
		beforeCount, afterCount)
}

// TestRateLimiterWithFile validates usage tracking when publishing from file
func TestRateLimiterWithFile(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-file-%d", time.Now().Unix())

	// Create a temporary test file
	testFile := fmt.Sprintf("/tmp/test-publish-%d.txt", time.Now().Unix())
	testContent := "Test file content for rate limit tracking"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "Failed to create test file")
	defer os.Remove(testFile)

	// Get initial usage stats
	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get initial usage stats")

	validatorBefore := NewValidator(usageBefore)
	usageInfoBefore, err := validatorBefore.ValidateUsage()
	require.NoError(t, err, "Failed to parse initial usage stats")

	beforeCount := parsePublishCount(usageInfoBefore.PublishCount)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err = subCmd.Start()
	require.NoError(t, err, "Failed to start subscriber")
	time.Sleep(2 * time.Second)

	// Publish from file
	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--file="+testFile,
		"--service-url="+serviceURL)
	require.NoError(t, err, "File publish failed: %s", out)

	cancel()
	subCmd.Wait()
	time.Sleep(1 * time.Second)

	// Get usage stats after publishing
	usageAfter, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats after publishing")

	validatorAfter := NewValidator(usageAfter)
	usageInfoAfter, err := validatorAfter.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats after publishing")

	afterCount := parsePublishCount(usageInfoAfter.PublishCount)

	// Verify publish count increased
	require.GreaterOrEqual(t, afterCount, beforeCount+1,
		"Publish count should increase by at least 1 (before: %d, after: %d)",
		beforeCount, afterCount)
}
