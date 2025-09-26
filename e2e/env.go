package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	_ = godotenv.Load() // silently load .env

	required := []string{"SERVICE_URL"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			return fmt.Errorf("env var %s must be set", key)
		}
	}
	return nil
}
