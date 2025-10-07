package main

import (
	"fmt"
	"strings"
)

type cliCommandCase struct {
	Name           string
	Args           []string
	ExpectContains []string
}

func (c cliCommandCase) Validate(output string) error {
	for _, expected := range c.ExpectContains {
		if !strings.Contains(output, expected) {
			return fmt.Errorf("expected output to contain %q, got %q", expected, output)
		}
	}
	return nil
}

func smokeTestCases() []cliCommandCase {
	return []cliCommandCase{
		{
			Name:           "health",
			Args:           []string{"health"},
			ExpectContains: []string{"Proxy Health Status:"},
		},
		{
			Name:           "whoami",
			Args:           []string{"whoami"},
			ExpectContains: []string{"Authentication Status:"},
		},
	}
}
