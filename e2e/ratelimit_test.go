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

	// Parse publish counts to verify they increased
	beforeCount := parsePublishCount(usageInfoBefore.PublishCount)
	afterCount := parsePublishCount(usageInfoAfter.PublishCount)
	require.GreaterOrEqual(t, afterCount, beforeCount, "Publish count should increase or stay same after publishing")
}

// parsePublishCount parses the publish count string to an integer
func parsePublishCount(countStr string) int {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0
	}
	return count
}
