package main

import (
	"encoding/base64"
	_ "errors"
	"fmt"
	"os"
	_ "os/exec"
	"path/filepath"
	"strings"
)

func SetupTokenFile() (string, error) {
	// 1. CI: use a pre-set secret file path
	//if path := os.Getenv("MUMP2P_E2E_TOKEN_PATH"); path != "" {
	//	return path, nil
	//}
	fmt.Printf("Trying to set token file MUMP2P_E2E_TOKEN_YAML ")
	// 2. CI: use raw YAML token provided directly via env
	if raw := os.Getenv("MUMP2P_E2E_TOKEN_YAML"); strings.TrimSpace(raw) != "" {
		return writeTokenFile([]byte(raw))
	}
	fmt.Printf("Trying to set MUMP2P_E2E_TOKEN_YAML ")
	// 3. CI: support legacy env name used in workflows
	//if raw := os.Getenv("AUTH0_TOKEN"); strings.TrimSpace(raw) != "" {
	//	return writeTokenFile([]byte(raw))
	//}
	fmt.Printf("Trying to set toekn file from MUMP2P_E2E_TOKEN_B64 ")

	// 4. CI: use a base64-encoded secret string
	if b64 := strings.TrimSpace(os.Getenv("MUMP2P_E2E_TOKEN_B64")); b64 != "" {
		fmt.Printf("MUMP2P_E2E_TOKEN_B64 is not empty and is read from env")
		decoded, err := base64.StdEncoding.DecodeString(b64)

		if err != nil {
			return "", fmt.Errorf("failed to decode MUMP2P_E2E_TOKEN_B64: %w", err)
		}

		return writeTokenFile(decoded)
	}

	// 5. Local fallback: encode from ~/.mump2p/auth.yml
	//cmd := exec.Command("sh", "-c", "base64 < ~/.mump2p/auth.yml | tr -d '\\n'")
	//raw, err := cmd.Output()
	//if err != nil {
	//	return "", fmt.Errorf("failed to encode token: %w", err)
	//}
	//
	//b64 := strings.TrimSpace(string(raw))
	//if b64 == "" {
	//	return "", errors.New("failed to encode token from ~/.mump2p/auth.yml")
	//}

	//decoded, err := base64.StdEncoding.DecodeString(b64)
	//if err != nil {
	//	return "", err
	//}

	//return writeTokenFile(decoded)
	return "", fmt.Errorf("MUMP2P_E2E_TOKEN_B64 is not empty")
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
