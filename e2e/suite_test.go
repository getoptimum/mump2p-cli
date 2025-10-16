package main

import (
	"fmt"
	"os"
	"testing"
)

var (
	cliBinaryPath string
	cliCleanup    func()
)

func TestMain(m *testing.M) {
	if os.Getenv("MUMP2P_E2E_SKIP") == "1" {
		fmt.Fprintln(os.Stderr, "[e2e] skipping CLI smoke tests (MUMP2P_E2E_SKIP=1)")
		os.Exit(0)
	}

	var err error
	cliBinaryPath, cliCleanup, err = PrepareCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[e2e] failed to prepare CLI: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	if cliCleanup != nil {
		cliCleanup()
	}
	os.Exit(code)
}
