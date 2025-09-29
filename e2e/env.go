package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads the environment from .env in local runs; fails if required vars are not set
func LoadEnv() error {
	_ = godotenv.Load() // silently load .env; ignore error bc in CI it's expected to fail and env is from repo secrets
	// When running `go test ./e2e -v` Load seen .env in directory it's invoked from; so its e2e/ for test harness
	// Try project root if .env isn't in ./e2e
	if os.Getenv("SERVICE_URL") == "" {
		_ = godotenv.Load("../.env")
	}

	required := []string{"SERVICE_URL"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			return fmt.Errorf("env var %s must be set", key)
		}
	}
	return nil
}
