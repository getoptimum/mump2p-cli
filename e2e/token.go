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
	if path := os.Getenv("MUMP2P_E2E_TOKEN_PATH"); path != "" {
		return path, nil
	}

	tmpDir, err := os.MkdirTemp("", "token")
	if err != nil {
		return "", err
	}
	tmpFile := filepath.Join(tmpDir, "auth.yml")

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

	err = os.WriteFile(tmpFile, decoded, 0600)
	if err != nil {
		return "", err
	}
	return tmpFile, nil
}
