package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	authPath     string
	debug        bool
	disableAuth  bool
	clientID     string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "mump2p",
	Short: "Direct P2P publish/subscribe on the Optimum Network",
	Long: `mump2p connects you directly to the Optimum P2P network.
Publish and subscribe with direct node connections for real-time, low-latency messaging.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add global flag for custom authentication path
	rootCmd.PersistentFlags().StringVar(&authPath, "auth-path", os.Getenv("MUMP2P_AUTH_PATH"), "Custom path for authentication file (default: ~/.mump2p/auth.yml, env: MUMP2P_AUTH_PATH)")

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode with session detail, node scores, message IDs, and peer paths")
	rootCmd.PersistentFlags().BoolVar(&disableAuth, "disable-auth", false, "Disable authentication checks (for testing/development)")

	// Add global client ID flag
	rootCmd.PersistentFlags().StringVar(&clientID, "client-id", "", "Client ID to use (required when --disable-auth is enabled)")

	// Add global output format flag
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")

	// disable completion option
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// GetAuthPath returns the custom auth path if set, otherwise empty string
func GetAuthPath() string {
	return authPath
}

// GetAuthDir returns the directory for auth files (either custom or default ~/.mump2p)
func GetAuthDir() string {
	if authPath != "" {
		return filepath.Dir(authPath)
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".mump2p")
}

// IsDebugMode returns true if debug mode is enabled
func IsDebugMode() bool {
	return debug
}

// IsAuthDisabled returns true if authentication is disabled
func IsAuthDisabled() bool {
	return disableAuth
}

// GetClientID returns the client ID when auth is disabled
func GetClientID() string {
	return clientID
}

// GetOutputFormat returns the output format
func GetOutputFormat() string {
	return outputFormat
}

func humanDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}
