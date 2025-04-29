package cmd

import (
	"fmt"

	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display the current version and Git commit used to build this binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", config.Version)
		fmt.Println("Commit: ", config.CommitHash)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
