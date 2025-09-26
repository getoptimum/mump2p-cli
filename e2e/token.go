package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func SetupTokenFile() (string, error) {
	// 1. CI: use a pre-set secret file path
	if path := os.Getenv("MUMP2P_E2E_TOKEN_PATH"); path != "" {
		return path, nil
	}

	// 2. CI: use a base64-encoded secret string
	if b64 := strings.TrimSpace(os.Getenv("MUMP2P_E2E_TOKEN_B64")); b64 != "" {
		tmpDir, err := os.MkdirTemp("", "token")
		if err != nil {
			return "", err
		}
		tmpFile := filepath.Join(tmpDir, "auth.yml")

		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return "", fmt.Errorf("failed to decode MUMP2P_E2E_TOKEN_B64: %w", err)
		}

		if err := os.WriteFile(tmpFile, decoded, 0600); err != nil {
			return "", err
		}
		return tmpFile, nil
	}

	// 3. Local fallback: encode from ~/.mump2p/auth.yml
	cmd := exec.Command("sh", "-c", "base64 < ~/.mump2p/auth.yml | tr -d '\\n'")
	raw, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %w", err)
	}

	b64 := strings.TrimSpace(string(raw))
	if b64 == "" {
		return "", errors.New("failed to encode token from ~/.mump2p/auth.yml")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "token")
	if err != nil {
		return "", err
	}
	tmpFile := filepath.Join(tmpDir, "auth.yml")

	if err := os.WriteFile(tmpFile, decoded, 0600); err != nil {
		return "", err
	}

	return tmpFile, nil
}
