package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	ConfigPath string // shared across CLI commands
)

var rootCmd = &cobra.Command{
	Use:   "mump2p",
	Short: "CLI to interact with OptimumP2P directly via Go",
	Long: `optcli is a developer tool for interacting with OptimumP2P
without relying on the HTTP server. It directly invokes Go services.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&ConfigPath, "config", "app_conf.yml", "Path to YAML config file")
}
