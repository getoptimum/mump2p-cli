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

func TestFullWorkflow(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	serviceURL := GetDefaultProxy()

	testTopic := fmt.Sprintf("workflow-%d", time.Now().Unix())

	t.Run("1_check_version", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "version")
		require.NoError(t, err, "version command failed: %v\nOutput: %s", err, out)
		require.Contains(t, out, "Version:", "Expected version output")
		require.Contains(t, out, "Commit:", "Expected commit hash")
	})

	t.Run("2_check_authentication", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "whoami")
		require.NoError(t, err, "whoami command failed: %v\nOutput: %s", err, out)
		require.Contains(t, out, "Authentication Status:", "Expected auth status")
		require.Contains(t, out, "Client ID:", "Expected client ID")
	})

	t.Run("3_check_proxy_health", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "health", "--service-url="+serviceURL)
		require.NoError(t, err, "health command failed: %v\nOutput: %s", err, out)
		require.Contains(t, out, "Proxy Health Status:", "Expected health status")
	})

	// Start subscriber in background before publishing
	t.Log("Starting background subscriber for workflow tests...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+serviceURL)
	subCmd.Env = os.Environ()
	err := subCmd.Start()
	require.NoError(t, err, "Failed to start background subscriber")

	// Wait for subscription to be active
	time.Sleep(2 * time.Second)
	t.Log("Subscriber active, proceeding with publish tests...")

	t.Run("4_publish_http_message", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--message=Integration test message",
			"--service-url="+serviceURL)
		require.NoError(t, err, "HTTP publish failed: %v\nOutput: %s", err, out)
		require.Contains(t, strings.ToLower(out), "published", "Expected published confirmation")
	})

	t.Run("5_publish_grpc_message", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "publish",
			"--topic="+testTopic,
			"--message=Integration gRPC test",
			"--grpc",
			"--service-url="+serviceURL)
		require.NoError(t, err, "gRPC publish failed: %v\nOutput: %s", err, out)
		require.Contains(t, strings.ToLower(out), "published", "Expected published confirmation")
	})

	t.Run("6_check_usage_stats", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "usage")
		require.NoError(t, err, "usage command failed: %v\nOutput: %s", err, out)
		require.Contains(t, out, "Publish (hour):", "Expected usage stats")
		require.Contains(t, out, "Data Used:", "Expected data usage")
	})

	t.Run("7_list_topics", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "list-topics", "--service-url="+serviceURL)
		require.NoError(t, err, "list-topics failed: %v\nOutput: %s", err, out)
		require.Contains(t, out, "Subscribed Topics", "Expected topics list")
		require.Contains(t, out, "Client:", "Expected client info")
	})

	t.Run("8_debug_mode_publish", func(t *testing.T) {
		out, err := RunCommand(cliBinaryPath, "--debug", "publish",
			"--topic="+testTopic,
			"--message=Debug workflow test",
			"--service-url="+serviceURL)
		require.NoError(t, err, "Debug publish failed: %v\nOutput: %s", err, out)
		require.Contains(t, strings.ToLower(out), "publish:", "Expected debug output")
		require.Contains(t, strings.ToLower(out), "sender_info:", "Expected sender info")
	})

	// Cleanup: stop subscriber
	cancel()
	subCmd.Wait()
	t.Log("Background subscriber stopped")
}

func TestCrossProxyWorkflow(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	// Test publishing and subscribing across different proxies
	proxies := TestProxies

	testTopic := fmt.Sprintf("cross-proxy-%d", time.Now().Unix())

	// Start subscriber on first proxy
	t.Log("Starting background subscriber for cross-proxy tests...")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath, "subscribe", "--topic="+testTopic, "--service-url="+proxies[0])
	subCmd.Env = os.Environ()
	err := subCmd.Start()
	require.NoError(t, err, "Failed to start background subscriber")

	// Wait for subscription to be active
	time.Sleep(2 * time.Second)
	t.Log("Subscriber active on first proxy...")

	for i, proxy := range proxies {
		proxyName := fmt.Sprintf("proxy_%d", i+1)
		t.Run(proxyName+"_publish", func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, "publish",
				"--topic="+testTopic,
				"--message=Cross-proxy test from "+proxyName,
				"--service-url="+proxy)
			require.NoError(t, err, "Publish to %s failed: %v\nOutput: %s", proxy, err, out)
			require.Contains(t, strings.ToLower(out), "published", "Expected published confirmation")
		})

		t.Run(proxyName+"_health", func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, "health", "--service-url="+proxy)
			require.NoError(t, err, "Health check for %s failed: %v\nOutput: %s", proxy, err, out)
			require.Contains(t, out, "Proxy Health Status:", "Expected health status")
		})
	}

	// Cleanup: stop subscriber
	cancel()
	subCmd.Wait()
	t.Log("Background subscriber stopped")
}
