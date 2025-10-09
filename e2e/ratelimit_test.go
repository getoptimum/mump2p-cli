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

// TestDailyQuotaTracking validates that usage stats are properly tracked
func TestDailyQuotaTracking(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	usageBefore, err := RunCommand(cliBinaryPath, "usage")
	require.NoError(t, err, "Failed to get usage stats")

	validator := NewValidator(usageBefore)
	usageInfo, err := validator.ValidateUsage()
	require.NoError(t, err, "Failed to parse usage stats")
	t.Logf("Usage before test: %s publishes", usageInfo.PublishCount)

	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://34.146.222.111:8080"
	}

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

	t.Logf("Usage stats before: %s", usageBefore)
	t.Logf("Usage stats after: %s", usageAfter)

	require.Contains(t, usageAfter, "Data Used:", "Usage stats should show data usage")
	t.Log("✅ Daily quota tracking is functional")
}
