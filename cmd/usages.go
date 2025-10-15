package cmd

import (
	"fmt"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/formatter"
	"github.com/getoptimum/mump2p-cli/internal/ratelimit"
	"github.com/spf13/cobra"
)

// UsageResponse represents structured usage statistics
type UsageResponse struct {
	PublishCount        int     `json:"publish_count" yaml:"publish_count"`
	PublishLimitPerHour int     `json:"publish_limit_per_hour" yaml:"publish_limit_per_hour"`
	SecondPublishCount  int     `json:"second_publish_count" yaml:"second_publish_count"`
	PublishLimitPerSec  int     `json:"publish_limit_per_sec" yaml:"publish_limit_per_sec"`
	BytesPublishedMB    float64 `json:"bytes_published_mb" yaml:"bytes_published_mb"`
	DailyQuotaMB        float64 `json:"daily_quota_mb" yaml:"daily_quota_mb"`
	NextReset           string  `json:"next_reset" yaml:"next_reset"`
	TimeUntilReset      string  `json:"time_until_reset" yaml:"time_until_reset"`
	LastPublishTime     string  `json:"last_publish_time,omitempty" yaml:"last_publish_time,omitempty"`
	LastSubscribeTime   string  `json:"last_subscribe_time,omitempty" yaml:"last_subscribe_time,omitempty"`
}

// usageCmd represents the usage command
var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Display usage statistics and rate limits",
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatter.New(GetOutputFormat())

		if IsAuthDisabled() {
			// When auth is disabled, usage tracking is not available
			if f.IsTable() {
				fmt.Println("Usage Statistics:")
				fmt.Println("  Status: Usage tracking disabled (using --disable-auth)")
				fmt.Println("  No rate limits or quotas are enforced in this mode")
			} else {
				response := map[string]string{
					"status":  "disabled",
					"message": "Usage tracking disabled (using --disable-auth)",
				}
				output, err := f.Format(response)
				if err != nil {
					return fmt.Errorf("failed to format output: %v", err)
				}
				fmt.Println(output)
			}
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

		// Prepare structured response
		response := UsageResponse{
			PublishCount:        stats.PublishCount,
			PublishLimitPerHour: stats.PublishLimitPerHour,
			SecondPublishCount:  stats.SecondPublishCount,
			PublishLimitPerSec:  stats.PublishLimitPerSec,
			BytesPublishedMB:    float64(stats.BytesPublished) / (1 << 20),
			DailyQuotaMB:        float64(stats.DailyQuota) / (1 << 20),
			NextReset:           stats.NextReset.Format(time.RFC822),
			TimeUntilReset:      stats.TimeUntilReset.String(),
		}

		if !stats.LastPublishTime.IsZero() {
			response.LastPublishTime = stats.LastPublishTime.Format(time.RFC822)
		}
		if !stats.LastSubscribeTime.IsZero() {
			response.LastSubscribeTime = stats.LastSubscribeTime.Format(time.RFC822)
		}

		if f.IsTable() {
			// display usage statistics (table format)
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
		} else {
			// JSON or YAML format
			output, err := f.Format(response)
			if err != nil {
				return fmt.Errorf("failed to format output: %v", err)
			}
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
