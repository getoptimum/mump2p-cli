package main

import (
	"bytes"
	"os"
	"os/exec"
)

// RunCommand executes the CLI binary with given arguments and returns output
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
