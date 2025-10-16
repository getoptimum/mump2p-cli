package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetupTokenFile creates a temporary auth file for testing
// Uses MUMP2P_E2E_TOKEN_B64 env var (CI) or falls back to ~/.mump2p/auth.yml (local)
func SetupTokenFile() (string, error) {
	if b64 := strings.TrimSpace(os.Getenv("MUMP2P_E2E_TOKEN_B64")); b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return "", fmt.Errorf("failed to decode MUMP2P_E2E_TOKEN_B64: %w", err)
		}
		return writeTokenFile(decoded)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	localAuthPath := filepath.Join(homeDir, ".mump2p", "auth.yml")
	data, err := os.ReadFile(localAuthPath)
	if err == nil {
		return writeTokenFile(data)
	}

	return "", fmt.Errorf("no token available: set MUMP2P_E2E_TOKEN_B64 or login with ./mump2p login")
}

func writeTokenFile(content []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "token")
	if err != nil {
		return "", err
	}

	tmpFile := filepath.Join(tmpDir, "auth.yml")
	if err := os.WriteFile(tmpFile, content, 0600); err != nil {
		return "", err
	}

	return tmpFile, nil
}
