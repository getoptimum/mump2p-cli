package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLISmokeCommands(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	for _, tc := range smokeTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			output := runCLICommand(t, tc.Args...)
			require.NoError(t, tc.Validate(output))
		})
	}
}

func runCLICommand(t *testing.T, args ...string) string {
	t.Helper()

	output, err := RunCommand(cliBinaryPath, args...)
	require.NoError(t, err, "command %v failed: %s", args, output)
	return output
}
