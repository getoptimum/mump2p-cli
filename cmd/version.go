package cmd

import (
	"fmt"

	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/formatter"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long:  `Display the current version and Git commit used to build this binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		f := formatter.New(GetOutputFormat())

		if f.IsTable() {
			// Table format (default)
			fmt.Println("Version:", config.Version)
			fmt.Println("Commit: ", config.CommitHash)
		} else {
			// JSON or YAML format
			data := map[string]string{
				"version":     config.Version,
				"commit_hash": config.CommitHash,
			}
			output, err := f.Format(data)
			if err != nil {
				fmt.Printf("Error formatting output: %v\n", err)
				return
			}
			fmt.Println(output)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
