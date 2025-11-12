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
			_, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in topic name")
			return
		}

		args := []string{"publish", "--topic=" + topic, "--message=test"}
		out, err := RunCommand(cliBinaryPath, args...)
		if err != nil {
			t.Logf("Command failed (expected for invalid input): %v\nOutput: %s", err, out)
		}
	})
}

// FuzzPublishMessage tests the publish command with a message
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
			_, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in message")
			return
		}

		args := []string{"publish", "--topic=fuzz-test", "--message=" + message}
		out, err := RunCommand(cliBinaryPath, args...)
		if err != nil {
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

	f.Fuzz(func(t *testing.T, url string) {
		if len(url) > 1000 {
			t.Skip()
		}

		args := []string{"health", "--service-url=" + url}
		out, err := RunCommand(cliBinaryPath, args...)
		if err != nil {
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
			_, err := RunCommand(cliBinaryPath, args...)
			require.Error(t, err, "CLI should reject null bytes in file path")
			return
		}

		args := []string{"publish", "--topic=fuzz-test", "--file=" + filepath}
		out, err := RunCommand(cliBinaryPath, args...)
		if err != nil {
			t.Logf("Command failed (expected for invalid file path): %v\nOutput: %s", err, out)
		}
	})
}
