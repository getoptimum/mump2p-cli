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

// TestPublishWithoutSubscriptionOnDifferentNode tests that publishing fails
// when there's no subscriber on a different proxy node
func TestPublishWithoutSubscriptionOnDifferentNode(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	proxies := TestProxies

	testTopic := fmt.Sprintf("no-sub-cross-%d", time.Now().Unix())

	tests := []struct {
		name        string
		subscribeOn int // proxy index to subscribe on
		publishOn   int // proxy index to publish on
		shouldFail  bool
		description string
	}{
		{
			name:        "publish_without_any_subscription",
			subscribeOn: -1, // No subscription
			publishOn:   0,
			shouldFail:  true,
			description: "Publishing without any subscriber should fail",
		},
		{
			name:        "cross_proxy_same_region",
			subscribeOn: 0, // Tokyo proxy 1
			publishOn:   1, // Tokyo proxy 2
			shouldFail:  false,
			description: "Publishing to different proxy in same region should work",
		},
		{
			name:        "cross_proxy_different_region",
			subscribeOn: 0, // Tokyo
			publishOn:   2, // Singapore
			shouldFail:  false,
			description: "Publishing across regions should work when subscriber exists",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			var cancel context.CancelFunc
			var subCmd *exec.Cmd

			// Start subscriber if needed
			if tt.subscribeOn >= 0 {
				ctx, c := context.WithCancel(context.Background())
				cancel = c
				defer cancel()

				subProxy := proxies[tt.subscribeOn]
				t.Logf("Starting subscriber on proxy %d: %s", tt.subscribeOn, subProxy)

				subCmd = exec.CommandContext(ctx, cliBinaryPath,
					"subscribe",
					"--topic="+testTopic,
					"--service-url="+subProxy)
				subCmd.Env = os.Environ()
				err := subCmd.Start()
				require.NoError(t, err, "Failed to start subscriber")

				// Wait for subscriber to be active
				time.Sleep(3 * time.Second)
				t.Log("Subscriber active")
			} else {
				t.Log("No subscriber started (testing publish without subscription)")
			}

			// Attempt to publish
			pubProxy := proxies[tt.publishOn]
			t.Logf("Publishing on proxy %d: %s", tt.publishOn, pubProxy)

			out, err := RunCommand(cliBinaryPath, "publish",
				"--topic="+testTopic,
				fmt.Sprintf("--message=CrossNodeTest-%s", tt.name),
				"--service-url="+pubProxy)

			// Validate expectations
			if tt.shouldFail {
				// Should fail with "topic not assigned" or similar
				if err == nil {
					t.Errorf("CRITICAL: Publishing without subscriber should have failed but succeeded! Output: %s", out)
				} else {
					lowerOut := strings.ToLower(out)
					if strings.Contains(lowerOut, "topic not assigned") ||
						strings.Contains(lowerOut, "not found") ||
						strings.Contains(lowerOut, "failed") {
						t.Logf("✅ Correctly rejected: %s", out)
					} else {
						t.Logf("⚠️ Failed but with unexpected error: %s", out)
					}
				}
			} else {
				// Should succeed
				require.NoError(t, err, "Publish failed: %v\nOutput: %s", err, out)

				validator := NewValidator(out)
				err := validator.ValidatePublishSuccess()
				require.NoError(t, err, "Publish validation failed")

				t.Logf("✅ Cross-node publish succeeded")
			}

			// Cleanup
			if cancel != nil {
				cancel()
				if subCmd != nil {
					subCmd.Wait()
				}
			}

			// Wait between tests to avoid rate limiting
			time.Sleep(2 * time.Second)
		})
	}
}

// TestCrossProxyFailover tests behavior when one proxy is down
func TestCrossProxyFailover(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	testTopic := fmt.Sprintf("failover-%d", time.Now().Unix())

	// Test with a definitely invalid proxy
	invalidProxy := "http://192.0.2.1:8080" // TEST-NET-1 (non-routable)

	t.Log("Testing publish to unreachable proxy (should fail quickly)")

	// Start subscriber on valid proxy first
	validProxy := GetDefaultProxy()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subCmd := exec.CommandContext(ctx, cliBinaryPath,
		"subscribe",
		"--topic="+testTopic,
		"--service-url="+validProxy)
	subCmd.Env = os.Environ()
	err := subCmd.Start()
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// Try publishing to invalid proxy (should fail)
	start := time.Now()
	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--message=FailoverTest",
		"--service-url="+invalidProxy)
	duration := time.Since(start)

	// Should fail
	require.Error(t, err, "Publishing to unreachable proxy should fail. Output: %s", out)

	// Should fail relatively quickly (not hang forever)
	require.Less(t, duration.Seconds(), 35.0,
		"Publish to unreachable proxy should timeout/fail within 35 seconds, took %v", duration)

	t.Logf("✅ Correctly failed to publish to unreachable proxy in %v", duration)

	cancel()
	subCmd.Wait()
}

// TestMultipleSubscribersOnDifferentProxies tests message delivery to multiple subscribers
func TestMultipleSubscribersOnDifferentProxies(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	proxies := GetProxySubset(2)

	testTopic := fmt.Sprintf("multi-sub-%d", time.Now().Unix())
	testMessage := fmt.Sprintf("MultiSubTest-%d", time.Now().Unix())

	t.Log("Starting subscribers on multiple proxies...")

	// Start subscribers on both proxies
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var subCmds []*exec.Cmd
	for i, proxy := range proxies {
		subCmd := exec.CommandContext(ctx, cliBinaryPath,
			"subscribe",
			"--topic="+testTopic,
			"--service-url="+proxy)
		subCmd.Env = os.Environ()
		err := subCmd.Start()
		require.NoError(t, err, "Failed to start subscriber %d on %s", i, proxy)
		subCmds = append(subCmds, subCmd)

		t.Logf("Subscriber %d started on %s", i, proxy)
	}

	// Wait for all subscribers to be ready
	time.Sleep(4 * time.Second)
	t.Log("All subscribers ready")

	// Publish message to first proxy
	out, err := RunCommand(cliBinaryPath, "publish",
		"--topic="+testTopic,
		"--message="+testMessage,
		"--service-url="+proxies[0])

	require.NoError(t, err, "Publish failed: %v\nOutput: %s", err, out)

	validator := NewValidator(out)
	err = validator.ValidatePublishSuccess()
	require.NoError(t, err)

	t.Log("✅ Successfully published to topic with multiple cross-proxy subscribers")

	// Cleanup
	cancel()
	for _, cmd := range subCmds {
		cmd.Wait()
	}
}

// TestProxyHealthBeforePublish tests that checking health before publishing is reliable
func TestProxyHealthBeforePublish(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	proxies := TestProxies

	for i, proxy := range proxies {
		t.Run(fmt.Sprintf("proxy_%d", i+1), func(t *testing.T) {
			// Check health first
			healthOut, healthErr := RunCommand(cliBinaryPath, "health", "--service-url="+proxy)

			if healthErr != nil {
				t.Skipf("Proxy %s is unhealthy, skipping: %v", proxy, healthErr)
				return
			}

			validator := NewValidator(healthOut)
			healthInfo, err := validator.ValidateHealthCheck()
			require.NoError(t, err, "Health check validation failed")

			t.Logf("Proxy %s health: %s", proxy, healthInfo.Status)

			// If healthy, test should be able to publish (with subscriber)
			testTopic := fmt.Sprintf("health-pub-%d-%d", i, time.Now().Unix())

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			subCmd := exec.CommandContext(ctx, cliBinaryPath,
				"subscribe",
				"--topic="+testTopic,
				"--service-url="+proxy)
			subCmd.Env = os.Environ()
			err = subCmd.Start()
			require.NoError(t, err)

			time.Sleep(2 * time.Second)

			pubOut, pubErr := RunCommand(cliBinaryPath, "publish",
				"--topic="+testTopic,
				"--message=HealthTest",
				"--service-url="+proxy)

			cancel()
			subCmd.Wait()

			require.NoError(t, pubErr, "Proxy reported healthy but publish failed: %s", pubOut)
			t.Logf("✅ Healthy proxy successfully processed publish")
		})
	}
}
