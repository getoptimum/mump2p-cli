package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthCommand(t *testing.T) {
	require.NotEmpty(t, cliBinaryPath, "CLI binary path must be set by TestMain")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   string
	}{
		{"basic", []string{"health"}, false, "Proxy Health Status:"},
		{"bad flag", []string{"health", "--bad"}, true, ""},
		// in case empty string is provided for --service_url -> defaults to .env
		{"missing flag", []string{"health", "--service-url="}, false, "Proxy Health Status:"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out, err := RunCommand(cliBinaryPath, tt.args...)

			if tt.expectError {
				require.Error(t, err, out)
			} else {
				require.NoError(t, err, out)
				if tt.expectOut != "" {
					require.Contains(t, strings.ToLower(out), strings.ToLower(tt.expectOut))
				}
			}
		})
	}
}
