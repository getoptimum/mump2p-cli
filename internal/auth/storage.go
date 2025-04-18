package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

// Storage handles token persistence
type Storage struct {
	tokenDir  string
	tokenFile string
}

// NewStorage creates a new token storage
func NewStorage() *Storage {
	homeDir, _ := os.UserHomeDir()
	tokenDir := filepath.Join(homeDir, ".optimum")
	tokenFile := filepath.Join(tokenDir, "auth.yml")

	return &Storage{
		tokenDir:  tokenDir,
		tokenFile: tokenFile,
	}
}

// SaveToken persists a token to disk
func (s *Storage) SaveToken(token *StoredToken) error {
	// create directory if it doesn't exist
	if err := os.MkdirAll(s.tokenDir, 0700); err != nil {
		return fmt.Errorf("error creating token directory: %v", err)
	}

	tokenData, err := yaml.Marshal(token)
	if err != nil {
		return fmt.Errorf("error encoding token: %v", err)
	}

	// write token to file
	if err := os.WriteFile(s.tokenFile, tokenData, 0600); err != nil {
		return fmt.Errorf("error saving token: %v", err)
	}

	return nil
}

// LoadToken retrieves a token from disk if valid
func (s *Storage) LoadToken() (*StoredToken, error) {
	// check if token file exists
	if _, err := os.Stat(s.tokenFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("not authenticated, please login first")
	}

	data, err := os.ReadFile(s.tokenFile)
	if err != nil {
		return nil, fmt.Errorf("error reading token: %v", err)
	}

	var token StoredToken
	if err := yaml.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	// check if token has expired
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token has expired, please login again")
	}

	return &token, nil
}

// RemoveToken deletes the stored token
func (s *Storage) RemoveToken() error {
	if _, err := os.Stat(s.tokenFile); os.IsNotExist(err) {
		return fmt.Errorf("not logged in")
	}

	if err := os.Remove(s.tokenFile); err != nil {
		return fmt.Errorf("error removing token: %v", err)
	}

	return nil
}
