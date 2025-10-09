package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailureScenarios(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	tests := []struct {
		name string
		args []string
	}{
		// Invalid commands
		{"invalid command", []string{"invalid-command"}},
		{"unknown subcommand", []string{"foobar"}},

		// Unknown flags
		{"health unknown flag", []string{"health", "--invalid-flag"}},
		{"publish unknown flag", []string{"publish", "--invalid-flag"}},
		{"subscribe unknown flag", []string{"subscribe", "--unknown"}},

		// Missing required flags
		{"publish no topic", []string{"publish", "--message=test"}},
		{"publish no message", []string{"publish", "--topic=test"}},
		{"subscribe no topic", []string{"subscribe"}},

		// Invalid flag values
		{"invalid service-url format", []string{"health", "--service-url=not-a-url"}},
		{"empty topic name", []string{"publish", "--topic=", "--message=test"}},
		{"empty message", []string{"publish", "--topic=test", "--message="}},

		// Invalid combinations
		{"publish file and message both", []string{"publish", "--topic=test", "--message=test", "--file=test.txt"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, tt.args...)
			require.Error(t, err, "Expected command to fail but it succeeded. Output: %s", out)
		})
	}
}

func TestInvalidFlagValues(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	tests := []struct {
		name string
		args []string
	}{
		// Service URL issues
		{"malformed URL", []string{"health", "--service-url=://broken"}},
		{"missing protocol", []string{"health", "--service-url=localhost:8080"}},

		// Numeric validation
		{"negative port", []string{"health", "--service-url=http://localhost:-8080"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, tt.args...)
			require.Error(t, err, "Expected command to fail but it succeeded. Output: %s", out)
		})
	}
}
