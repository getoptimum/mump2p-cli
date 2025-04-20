package cmd

import (
	"fmt"
	"time"

	"github.com/getoptimum/optcli/internal/auth"
	"github.com/getoptimum/optcli/internal/ratelimit"
	"github.com/spf13/cobra"
)

// usageCmd represents the usage command
var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Display usage statistics and rate limits",
	RunE: func(cmd *cobra.Command, args []string) error {
		// get valid token (refreshes if needed)
		authClient := auth.NewClient()
		storage := auth.NewStorage()
		token, err := authClient.GetValidToken(storage)
		if err != nil {
			return fmt.Errorf("authentication required: %v", err)
		}

		// parse token to get rate limits
		parser := auth.NewTokenParser()
		claims, err := parser.ParseToken(token.Token)
		if err != nil {
			return fmt.Errorf("error parsing token: %v", err)
		}

		// initialize rate limiter
		limiter, err := ratelimit.NewRateLimiter(claims)
		if err != nil {
			return fmt.Errorf("error initializing rate limiter: %v", err)
		}

		// get usage statistics
		stats := limiter.GetUsageStats()

		// display usage statistics
		fmt.Printf("Publish:    %d/%d operations\n", stats.PublishCount, stats.PublishLimit)
		fmt.Printf("Data Usage: %.2f MB / %.2f MB\n", float64(stats.BytesPublished)/(1<<20), float64(stats.DailyQuota)/(1<<20))
		fmt.Printf("Next Reset: %s (%s from now)\n", stats.NextReset.Format(time.RFC822), stats.TimeUntilReset)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
