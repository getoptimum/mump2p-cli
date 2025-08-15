package cmd

import (
	"fmt"
	"os"

	"github.com/getoptimum/mump2p-cli/internal/config"
	ocauth "github.com/getoptimum/optimum-common/auth"
	"github.com/spf13/cobra"
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
	// disable completion option
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	ocauth.SetDefaultLimits(ocauth.LimitsDefaults{
		MaxPublishPerHour: config.DefaultMaxPublishPerHour,
		MaxPublishPerSec:  config.DefaultMaxPublishPerSec,
		MaxMessageSize:    config.DefaultMaxMessageSize,
		DailyQuota:        config.DefaultDailyQuota,
	})
}
