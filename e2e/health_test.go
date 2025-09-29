package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthCommand(t *testing.T) {
	cli, cleanup, err := PrepareCLI()
	require.NoError(t, err, "failed to prepare CLI")
	defer cleanup()

	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectOut   string
	}{
		{"basic", []string{"health"}, false, "ok"},
		{"bad flag", []string{"health", "--bad"}, true, ""},
		// in case empty string is provided for --service_url -> defaults to .env
		{"missing flag", []string{"health", "--service-url="}, false, "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RunCommand(cli, tt.args...)

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
