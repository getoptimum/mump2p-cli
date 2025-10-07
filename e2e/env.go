package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv loads the environment from .env in local runs; fails if required vars are not set
func LoadEnv() error {
	_ = godotenv.Load() // silently load .env; ignore error bc in CI it's expected to fail and env is from repo secrets
	if os.Getenv("SERVICE_URL") == "" {
		if root, err := findRepoRoot(); err == nil {
			_ = godotenv.Load(filepath.Join(root, ".env"))
		}
	}

	required := []string{"SERVICE_URL"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			return fmt.Errorf("env var %s must be set", key)
		}
	}
	return nil
}
