package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables and validates required vars are set
func LoadEnv() error {
	_ = godotenv.Load()
	if os.Getenv("SERVICE_URL") == "" {
		if root, err := findRepoRoot(); err == nil {
			_ = godotenv.Load(filepath.Join(root, ".env"))
		}
	}

	if os.Getenv("SERVICE_URL") == "" {
		return fmt.Errorf("env var SERVICE_URL must be set")
	}
	return nil
}
