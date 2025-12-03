package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDailyQuotaTracking validates that usage stats are properly tracked
func TestDailyQuotaTracking(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats")

	validator := NewValidator(usageBefore)
	usageInfoBefore, err := validator.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats")

	serviceURL := GetDefaultProxy()

	testTopic := fmt.Sprintf("quota-%d", time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err = subCmd.Start()
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	out, err := RunCommand(cliBinaryPath, "publish", "--topic="+testTopic, "--message=QuotaTrackingTest", "--service-url="+serviceURL)
	require.NoError(t, err, "Publish failed: %s", out)

	cancel()
	subCmd.Wait()
	time.Sleep(1 * time.Second)

	usageAfter, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats after publish")

	validatorAfter := NewValidator(usageAfter)
	usageInfoAfter, err := validatorAfter.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats after publish")

	// Verify usage increased
	require.Contains(t, usageAfter, "Data Used:", "Usage stats should show data usage")

	// Parse publish counts to verify they increased exactly by 1 (tests not run in parallel)
	beforeCount := parsePublishCount(t, usageInfoBefore.PublishCount)
	afterCount := parsePublishCount(t, usageInfoAfter.PublishCount)
	require.Equal(t, beforeCount+1, afterCount,
		"Publish count should increase by exactly 1 (before: %d, after: %d)",
		beforeCount, afterCount)
}

// parsePublishCount parses the publish count string to an integer
func parsePublishCount(t *testing.T, countStr string) int {
	t.Helper()
	count, err := strconv.Atoi(countStr)
	if err != nil {
		t.Logf("Failed to parse publish count '%s': %v", countStr, err)
		return 0
	}
	return count
}
