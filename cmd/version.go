package cmd

import (
	"fmt"

	"github.com/getoptimum/optimum-common/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display the current version and Git commit used to build this binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", version.Version)
		fmt.Println("Commit:", version.CommitHash)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
