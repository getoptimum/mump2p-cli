package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// getInitialPublishCount is a helper function to get the initial publish count from usage stats
func getInitialPublishCount(t *testing.T) int {
	t.Helper()
	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get initial usage stats")

	validatorBefore := NewValidator(usageBefore)
	usageInfoBefore, err := validatorBefore.ValidateUsage()
	require.NoError(t, err, "Failed to parse initial usage stats")

	return parsePublishCount(usageInfoBefore.PublishCount)
}

// TestRateLimiterScenarios validates that usage stats change correctly after publishing messages
func TestRateLimiterScenarios(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-%d", time.Now().Unix())

	beforeCount := getInitialPublishCount(t)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err := subCmd.Start()
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

	// Verify publish count increased exactly by numMessages (tests not run in parallel)
	require.Equal(t, beforeCount+numMessages, afterCount,
		"Publish count should increase by exactly %d (before: %d, after: %d)",
		numMessages, beforeCount, afterCount)

	// Verify data usage is present
	require.Contains(t, usageAfter, "Data Used:", "Usage stats should show data usage")
}

// TestRateLimitExceededPerHour tests that per-hour rate limit is enforced
func TestRateLimitExceededPerHour(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-hour-%d", time.Now().Unix())

	// Get initial usage stats to determine per-hour limit
	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get initial usage stats")

	validatorBefore := NewValidator(usageBefore)
	usageInfoBefore, err := validatorBefore.ValidateUsage()
	require.NoError(t, err, "Failed to parse initial usage stats")

	limitPerHour, err := strconv.Atoi(usageInfoBefore.PublishLimitPerHour)
	require.NoError(t, err, "Failed to parse per-hour limit")
	require.Greater(t, limitPerHour, 0, "Per-hour limit should be greater than 0")

	// Get current publish count
	currentCount := parsePublishCount(usageInfoBefore.PublishCount)

	// Calculate how many more publishes we can do before hitting the limit
	remaining := limitPerHour - currentCount
	if remaining <= 0 {
		t.Skipf("Already at or over per-hour limit (%d/%d). Cannot test limit enforcement.", currentCount, limitPerHour)
	}
	// Skip if remaining is too high to avoid long test times (e.g., > 100)
	if remaining > 100 {
		t.Skipf("Per-hour limit is too high (%d remaining). Skipping to avoid long test times.", remaining)
	}

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err = subCmd.Start()
	require.NoError(t, err, "Failed to start subscriber")
	time.Sleep(2 * time.Second)

	// Publish up to the limit (should succeed)
	for i := 0; i < remaining; i++ {
		msg := fmt.Sprintf("RateLimitHourTest-%d", i+1)
		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--message="+msg,
			"--service-url="+serviceURL)
		require.NoError(t, err, "Publish %d should succeed: %s", i+1, out)
	}

	// Try to publish one more (should exceed per-hour limit)
	msg := fmt.Sprintf("RateLimitHourTest-%d", remaining+1)
	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--message="+msg,
		"--service-url="+serviceURL)
	require.Error(t, err, "Publish should fail when exceeding per-hour limit. Output: %s", out)
	require.Contains(t, strings.ToLower(out), "per-hour", "Error should mention per-hour limit. Got: %s", out)

	cancel()
	subCmd.Wait()
}

// TestRateLimitExceededMessageSize tests that message size limit is enforced
func TestRateLimitExceededMessageSize(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-size-%d", time.Now().Unix())

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err := subCmd.Start()
	require.NoError(t, err, "Failed to start subscriber")
	time.Sleep(2 * time.Second)

	// Create a file with content that exceeds any reasonable limit
	// Default MaxMessageSize is 2MB, but token might have higher limit
	// Use 10MB to ensure it exceeds any reasonable limit
	dir := t.TempDir()
	largeFile := filepath.Join(dir, "large-message.txt")
	largeContent := strings.Repeat("A", 10*1024*1024) // 10MB - should exceed any reasonable limit
	err = os.WriteFile(largeFile, []byte(largeContent), 0644)
	require.NoError(t, err, "Failed to create large test file")

	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--file="+largeFile,
		"--service-url="+serviceURL)
	require.Error(t, err, "Publish should fail when message size exceeds limit. Output: %s", out)
	// Error could be from CLI rate limiter ("message size exceeds limit") or server ("request entity too large")
	lowerOut := strings.ToLower(out)
	hasSizeError := strings.Contains(lowerOut, "message size") ||
		strings.Contains(lowerOut, "entity too large") ||
		strings.Contains(lowerOut, "request entity too large")
	require.True(t, hasSizeError, "Error should mention message size or entity too large. Got: %s", out)

	cancel()
	subCmd.Wait()
}

// TestRateLimiterWithGRPC validates usage tracking with gRPC protocol
func TestRateLimiterWithGRPC(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-grpc-%d", time.Now().Unix())

	beforeCount := getInitialPublishCount(t)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--grpc", "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err := subCmd.Start()
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

	// Verify publish count increased exactly by 1 (tests not run in parallel)
	require.Equal(t, beforeCount+1, afterCount,
		"Publish count should increase by exactly 1 (before: %d, after: %d)",
		beforeCount, afterCount)
}

// TestRateLimiterWithFile validates usage tracking when publishing from file
func TestRateLimiterWithFile(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()
	testTopic := fmt.Sprintf("ratelimit-file-%d", time.Now().Unix())

	// Create a temporary test file using t.TempDir() for consistency
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test-publish.txt")
	testContent := "Test file content for rate limit tracking"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "Failed to create test file")

	beforeCount := getInitialPublishCount(t)

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

	// Verify publish count increased exactly by 1 (tests not run in parallel)
	require.Equal(t, beforeCount+1, afterCount,
		"Publish count should increase by exactly 1 (before: %d, after: %d)",
		beforeCount, afterCount)
}
