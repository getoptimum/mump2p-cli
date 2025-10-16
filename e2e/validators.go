package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// OutputValidator provides strict validation for CLI output
type OutputValidator struct {
	output string
}

// NewValidator creates a new output validator
func NewValidator(output string) *OutputValidator {
	return &OutputValidator{output: output}
}

// ContainsAll validates that output contains all expected strings
func (v *OutputValidator) ContainsAll(expected ...string) error {
	for _, exp := range expected {
		if !strings.Contains(v.output, exp) {
			return fmt.Errorf("expected output to contain %q, got:\n%s", exp, v.output)
		}
	}
	return nil
}

// MatchesPattern validates output against a regex pattern
func (v *OutputValidator) MatchesPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	if !re.MatchString(v.output) {
		return fmt.Errorf("output does not match pattern %q, got:\n%s", pattern, v.output)
	}
	return nil
}

// ExtractMatch extracts the first match of a regex pattern
func (v *OutputValidator) ExtractMatch(pattern string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}
	matches := re.FindStringSubmatch(v.output)
	if len(matches) < 2 {
		return "", fmt.Errorf("no match found for pattern %q in output:\n%s", pattern, v.output)
	}
	return matches[1], nil
}

// ValidateJSON checks if output is valid JSON and matches structure
func (v *OutputValidator) ValidateJSON(target interface{}) error {
	if err := json.Unmarshal([]byte(v.output), target); err != nil {
		return fmt.Errorf("failed to parse JSON: %w\nOutput:\n%s", err, v.output)
	}
	return nil
}

// NotContains validates that output does NOT contain certain strings
func (v *OutputValidator) NotContains(unwanted ...string) error {
	for _, unw := range unwanted {
		if strings.Contains(v.output, unw) {
			return fmt.Errorf("output should not contain %q, but got:\n%s", unw, v.output)
		}
	}
	return nil
}

// ValidateVersion checks version output format (e.g., "Version: v0.0.1-rc7\nCommit:  8e333bf")
func (v *OutputValidator) ValidateVersion() (*VersionInfo, error) {
	info := &VersionInfo{}

	versionPattern := `Version:\s+(v?\d+\.\d+\.\d+(?:-[\w.]+)?)`
	version, err := v.ExtractMatch(versionPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %w", err)
	}
	info.Version = version

	commitPattern := `Commit:\s+([a-f0-9]{7,40})`
	commit, err := v.ExtractMatch(commitPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid commit format: %w", err)
	}
	info.Commit = commit

	return info, nil
}

// ValidateWhoami checks whoami output format
func (v *OutputValidator) ValidateWhoami() (*WhoamiInfo, error) {
	info := &WhoamiInfo{}

	// Check for authentication status
	if err := v.ContainsAll("Authentication Status:", "Client ID:"); err != nil {
		return nil, err
	}

	// Extract client ID (Auth0 format or email)
	clientPattern := `Client ID:\s+(.+?)(?:\n|$)`
	clientID, err := v.ExtractMatch(clientPattern)
	if err != nil {
		return nil, fmt.Errorf("could not extract client ID: %w", err)
	}
	info.ClientID = strings.TrimSpace(clientID)

	if info.ClientID == "" {
		return nil, fmt.Errorf("client ID is empty")
	}

	return info, nil
}

// ValidatePublishSuccess checks successful publish output
func (v *OutputValidator) ValidatePublishSuccess() error {
	// Must contain published confirmation
	publishedPattern := `(?i)(published|message sent successfully)`
	if err := v.MatchesPattern(publishedPattern); err != nil {
		return fmt.Errorf("publish success not confirmed: %w", err)
	}

	// Should not contain error keywords
	if err := v.NotContains("Error:", "error:", "failed", "Failed"); err != nil {
		return err
	}

	return nil
}

// ValidateHealthCheck validates health check output
func (v *OutputValidator) ValidateHealthCheck() (*HealthInfo, error) {
	info := &HealthInfo{}

	if err := v.ContainsAll("Proxy Health Status:"); err != nil {
		return nil, err
	}

	// Extract status (ok, unhealthy, etc.) - format: "Status:      ok"
	statusPattern := `Status:\s+(\w+)`
	status, err := v.ExtractMatch(statusPattern)
	if err != nil {
		return nil, fmt.Errorf("could not extract health status: %w", err)
	}
	info.Status = strings.TrimSpace(status)

	return info, nil
}

// ValidateUsage validates usage stats output
func (v *OutputValidator) ValidateUsage() (*UsageInfo, error) {
	info := &UsageInfo{}

	if err := v.ContainsAll("Publish (hour):", "Data Used:"); err != nil {
		return nil, err
	}

	// Extract publish count (number format)
	publishPattern := `Publish \(hour\):\s+(\d+)`
	publishCount, err := v.ExtractMatch(publishPattern)
	if err != nil {
		return nil, fmt.Errorf("could not extract publish count: %w", err)
	}
	info.PublishCount = publishCount

	return info, nil
}

// VersionInfo holds parsed version information
type VersionInfo struct {
	Version string
	Commit  string
}

// WhoamiInfo holds parsed whoami information
type WhoamiInfo struct {
	ClientID string
}

// HealthInfo holds parsed health check information
type HealthInfo struct {
	Status string
}

// UsageInfo holds parsed usage statistics
type UsageInfo struct {
	PublishCount string
}
