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
		fmt.Println("Usage Statistics and Rate Limits")
		fmt.Println("================================")
		fmt.Printf("Publish:      %d/%d operations\n", stats["publish_count"], stats["publish_limit"])
		fmt.Printf("Subscribe:    %d/%d operations\n", stats["subscribe_count"], stats["subscribe_limit"])
		fmt.Printf("Data Usage:   %s/%s\n", stats["bytes_published"], stats["daily_quota"])
		fmt.Printf("Next Reset:   %s (%s from now)\n",
			stats["next_reset"].(time.Time).Format(time.RFC822),
			stats["time_until_reset"])

		// token information
		fmt.Println("\nToken Information")
		fmt.Println("=================")
		fmt.Printf("Expires At:   %s\n", token.ExpiresAt.Format(time.RFC822))
		fmt.Printf("Valid For:    %s\n", time.Until(token.ExpiresAt).Round(time.Minute))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
