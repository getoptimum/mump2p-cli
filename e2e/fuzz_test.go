package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzPublishTopicName tests the publish command with a topic name
func FuzzPublishTopicName(f *testing.F) {
	require.NotEmpty(f, cliBinaryPath, "CLI binary path must be set by TestMain")

	f.Add("")
	f.Add("\x00")
	f.Add("../../../etc/passwd")
	f.Add(strings.Repeat("a", 1000))
	f.Add("test\n\r\t")
	f.Add("test'; DROP TABLE")
	f.Add("${HOME}")

	f.Fuzz(func(t *testing.T, topic string) {
		if len(topic) > 5000 {
			t.Skip()
		}

		// For obviously invalid input, verify graceful error handling
		if strings.Contains(topic, "\x00") {
			args := []string{"publish", "--topic=" + topic, "--message=test"}
			out, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in topic name")
			// Verify error is handled gracefully (not a panic)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on topic with null byte: %v\nOutput: %s", err, out)
			}
			return
		}

		args := []string{"publish", "--topic=" + topic, "--message=test"}
		out, err := RunCommand(cliBinaryPath, args...)
		// For fuzzing, we need to detect failures - invalid topics should fail gracefully
		// Valid topics might succeed (if subscribed) or fail (if not subscribed)
		// The key is that the CLI should handle the input without panicking
		if err != nil {
			// Any error should be handled gracefully (not a panic or crash)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on topic %q: %v\nOutput: %s", topic, err, out)
			}
			// For invalid topics, errors are expected and acceptable
			t.Logf("Command failed (expected for invalid input): %v\nOutput: %s", err, out)
		}
	})
}

// FuzzPublishMessage tests the publish command with a message.
// This is distinct from FuzzPublishTopicName which tests topic name validation;
// this test focuses on message content validation and handling.
func FuzzPublishMessage(f *testing.F) {
	require.NotEmpty(f, cliBinaryPath, "CLI binary path must be set by TestMain")

	f.Add("")
	f.Add("\x00")
	f.Add(strings.Repeat("a", 10000))
	f.Add("{\"test\": \"value\"}")
	f.Add("test<script>alert(1)</script>")

	f.Fuzz(func(t *testing.T, message string) {
		if len(message) > 50000 {
			t.Skip()
		}

		// For obviously invalid input, verify graceful error handling
		if strings.Contains(message, "\x00") {
			args := []string{"publish", "--topic=fuzz-test", "--message=" + message}
			out, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in message")
			// Verify error is handled gracefully (not a panic)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on message with null byte: %v\nOutput: %s", err, out)
			}
			return
		}

		args := []string{"publish", "--topic=fuzz-test", "--message=" + message}
		out, err := RunCommand(cliBinaryPath, args...)
		// For fuzzing, we need to detect failures - invalid messages should fail gracefully
		// Valid messages might succeed (if topic is subscribed) or fail (if not subscribed)
		// The key is that the CLI should handle the input without panicking
		if err != nil {
			// Any error should be handled gracefully (not a panic or crash)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on message %q: %v\nOutput: %s", message, err, out)
			}
			// For invalid messages, errors are expected and acceptable
			t.Logf("Command failed (expected for invalid input): %v\nOutput: %s", err, out)
		}
	})
}

// FuzzServiceURL tests the health command with a service URL
func FuzzServiceURL(f *testing.F) {
	require.NotEmpty(f, cliBinaryPath, "CLI binary path must be set by TestMain")

	f.Add("not-a-url")
	f.Add("://broken")
	f.Add("http://")
	f.Add("http://localhost:-8080")
	f.Add("http://localhost:99999")
	f.Add("javascript:alert(1)")
	f.Add("\x00")

	f.Fuzz(func(t *testing.T, url string) {
		if len(url) > 1000 {
			t.Skip()
		}

		// For obviously invalid input, verify graceful error handling
		if strings.Contains(url, "\x00") {
			args := []string{"health", "--service-url=" + url}
			out, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in service URL")
			// Verify error is handled gracefully (not a panic)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on service URL with null byte: %v\nOutput: %s", err, out)
			}
			return
		}

		args := []string{"health", "--service-url=" + url}
		out, err := RunCommand(cliBinaryPath, args...)
		// For fuzzing, we need to detect failures - invalid URLs should fail gracefully
		// Valid URLs might succeed (if proxy is reachable) or fail (if not)
		// The key is that the CLI should handle the input without panicking
		if err != nil {
			// Any error should be handled gracefully (not a panic or crash)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on URL %q: %v\nOutput: %s", url, err, out)
			}
			// For invalid URLs, errors are expected and acceptable
			t.Logf("Command failed (expected for invalid URL): %v\nOutput: %s", err, out)
		}
	})
}

// FuzzFilePath tests the publish command with a file path
func FuzzFilePath(f *testing.F) {
	require.NotEmpty(f, cliBinaryPath, "CLI binary path must be set by TestMain")

	f.Add("")
	f.Add("nonexistent.txt")
	f.Add("../../../etc/passwd")
	f.Add("/dev/null")
	f.Add("test\x00file.txt")
	f.Add(strings.Repeat("a", 500) + ".txt")

	f.Fuzz(func(t *testing.T, filepath string) {
		if len(filepath) > 2000 {
			t.Skip()
		}

		// For obviously invalid input, verify graceful error handling
		if strings.Contains(filepath, "\x00") {
			args := []string{"publish", "--topic=fuzz-test", "--file=" + filepath}
			out, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in file path")
			// Verify error is handled gracefully (not a panic)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on file path with null byte: %v\nOutput: %s", err, out)
			}
			return
		}

		args := []string{"publish", "--topic=fuzz-test", "--file=" + filepath}
		out, err := RunCommand(cliBinaryPath, args...)
		// For fuzzing, we need to detect failures - invalid file paths should fail gracefully
		// Valid file paths might succeed (if file exists and topic is subscribed) or fail (if not)
		// The key is that the CLI should handle the input without panicking
		if err != nil {
			// Any error should be handled gracefully (not a panic or crash)
			if strings.Contains(out, "panic") || strings.Contains(out, "fatal") {
				t.Fatalf("CLI panicked or crashed on file path %q: %v\nOutput: %s", filepath, err, out)
			}
			// For invalid file paths, errors are expected and acceptable
			t.Logf("Command failed (expected for invalid file path): %v\nOutput: %s", err, out)
		}
	})
}
