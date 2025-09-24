package cmd

import (
	"fmt"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/ratelimit"
	"github.com/spf13/cobra"
)

// usageCmd represents the usage command
var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Display usage statistics and rate limits",
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsAuthDisabled() {
			// Display mock usage statistics when auth is disabled
			fmt.Println("Usage Statistics (Auth Disabled):")
			fmt.Println("  Publish (hour):     0 / unlimited")
			fmt.Println("  Publish (second):   0 / unlimited")
			fmt.Println("  Data Used:          0.0000 MB / unlimited MB")
			fmt.Println("  Next Reset:         N/A (auth disabled)")
			fmt.Println("  Last Publish:       N/A")
			fmt.Println("  Last Subscribe:     N/A")
			return nil
		}

		// get valid token (refreshes if needed)
		authClient := auth.NewClient()
		storage := auth.NewStorageWithPath(GetAuthPath())
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
		limiter, err := ratelimit.NewRateLimiterWithDir(claims, GetAuthDir())
		if err != nil {
			return fmt.Errorf("error initializing rate limiter: %v", err)
		}

		// get usage statistics
		stats := limiter.GetUsageStats()

		// display usage statistics
		fmt.Printf("  Publish (hour):     %d / %d\n", stats.PublishCount, stats.PublishLimitPerHour)
		fmt.Printf("  Publish (second):   %d / %d\n", stats.SecondPublishCount, stats.PublishLimitPerSec)
		fmt.Printf("  Data Used:          %.4f MB / %.4f MB\n", float64(stats.BytesPublished)/(1<<20), float64(stats.DailyQuota)/(1<<20))
		fmt.Printf("  Next Reset:         %s (%s from now)\n", stats.NextReset.Format(time.RFC822), stats.TimeUntilReset)

		if !stats.LastPublishTime.IsZero() {
			fmt.Printf("  Last Publish:       %s\n", stats.LastPublishTime.Format(time.RFC822))
		}
		if !stats.LastSubscribeTime.IsZero() {
			fmt.Printf("  Last Subscribe:     %s\n", stats.LastSubscribeTime.Format(time.RFC822))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
