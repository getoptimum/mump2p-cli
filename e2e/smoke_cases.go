package main

import (
	"fmt"
	"os"
)

type cliCommandCase struct {
	Name           string
	Args           []string
	StrictValidate func(string) error // New: strict validation function
}

func (c cliCommandCase) Validate(output string) error {
	if c.StrictValidate != nil {
		return c.StrictValidate(output)
	}
	return nil
}

func smokeTestCases() []cliCommandCase {
	serviceURL := os.Getenv("SERVICE_URL")
	if serviceURL == "" {
		serviceURL = GetDefaultProxy()
	}

	return []cliCommandCase{
		{
			Name: "version",
			Args: []string{"version"},
			StrictValidate: func(output string) error {
				validator := NewValidator(output)
				versionInfo, err := validator.ValidateVersion()
				if err != nil {
					return err
				}
				// Additional validation: version should not be empty
				if versionInfo.Version == "" {
					return fmt.Errorf("version is empty")
				}
				if versionInfo.Commit == "" {
					return fmt.Errorf("commit hash is empty")
				}
				return nil
			},
		},
		{
			Name: "health",
			Args: []string{"health", "--service-url=" + serviceURL},
			StrictValidate: func(output string) error {
				validator := NewValidator(output)
				healthInfo, err := validator.ValidateHealthCheck()
				if err != nil {
					return err
				}
				// Validate status is not empty
				if healthInfo.Status == "" {
					return fmt.Errorf("health status is empty")
				}
				return nil
			},
		},
		{
			Name: "whoami",
			Args: []string{"whoami"},
			StrictValidate: func(output string) error {
				validator := NewValidator(output)
				whoamiInfo, err := validator.ValidateWhoami()
				if err != nil {
					return err
				}
				// Validate client ID is not empty
				if whoamiInfo.ClientID == "" {
					return fmt.Errorf("client ID is empty")
				}
				return nil
			},
		},
		{
			Name: "usage",
			Args: []string{"usage"},
			StrictValidate: func(output string) error {
				validator := NewValidator(output)
				usageInfo, err := validator.ValidateUsage()
				if err != nil {
					return err
				}
				// Validate publish count exists (can be "0")
				if usageInfo.PublishCount == "" {
					return fmt.Errorf("publish count is empty")
				}
				return nil
			},
		},
		{
			Name: "list-topics",
			Args: []string{"list-topics", "--service-url=" + serviceURL},
			StrictValidate: func(output string) error {
				validator := NewValidator(output)
				return validator.ContainsAll("Subscribed Topics", "Client:")
			},
		},
	}
}
