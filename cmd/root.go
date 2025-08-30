package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	authPath string // Global flag for custom authentication file path
)

var rootCmd = &cobra.Command{
	Use:   "mump2p",
	Short: "CLI to interact with OptimumP2P directly via Go",
	Long: `mump2p is a developer tool for interacting with OptimumP2P
without relying on the HTTP server. It directly invokes Go services.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func init() {
	// Add global flag for custom authentication path
	rootCmd.PersistentFlags().StringVar(&authPath, "auth-path", os.Getenv("MUMP2P_AUTH_PATH"), "Custom path for authentication file (default: ~/.optimum/auth.yml, env: MUMP2P_AUTH_PATH)")

	// disable completion option
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// GetAuthPath returns the custom auth path if set, otherwise empty string
func GetAuthPath() string {
	return authPath
}

// GetAuthDir returns the directory for auth files (either custom or default ~/.optimum)
func GetAuthDir() string {
	if authPath != "" {
		return filepath.Dir(authPath)
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".optimum")
}
