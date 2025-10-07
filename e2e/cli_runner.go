package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func RunE2ETests() error {
	if os.Getenv("MUMP2P_E2E_SKIP") == "1" {
		fmt.Println("[e2e] skipping CLI smoke tests (MUMP2P_E2E_SKIP=1)")
		return nil
	}

	cli, cleanup, err := PrepareCLI()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	for _, test := range smokeTestCases() {
		fmt.Printf("[e2e] running %s\n", test.Name)
		output, err := RunCommand(cli, test.Args...)
		if err != nil {
			return fmt.Errorf("test %s failed: %w\nOutput: %s", test.Name, err, output)
		}
		if err := test.Validate(output); err != nil {
			return fmt.Errorf("test %s failed: %w", test.Name, err)
		}
	}

	return nil
}

func RunCommand(bin string, args ...string) (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	return out.String(), nil
}
