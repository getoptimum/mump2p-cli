package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/getoptimum/optcli/internal/auth"
	"github.com/getoptimum/optcli/internal/config"
	"github.com/getoptimum/optcli/internal/ratelimit"
	"github.com/spf13/cobra"
)

var (
	pubTopic   string
	pubMessage string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to the OptimumP2P via HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		authClient := auth.NewClient()
		storage := auth.NewStorage()
		token, err := authClient.GetValidToken(storage)
		if err != nil {
			return fmt.Errorf("authentication required: %v", err)
		}
		srcUrl := config.LoadConfig().ServiceUrl
		// TODO:: change the API, use only optimump2p and message size based on the message itself
		reqBody := fmt.Sprintf(`{"topic": "%s", "protocol": ["%s"], "message_size": %d}`, pubTopic, "optimump2p", config.DefaultMaxMessageSize)

		request, err := http.NewRequest("POST", srcUrl+"/api/publish", strings.NewReader(reqBody))
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+token.Token)
		request.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("HTTP publish failed: %v", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			return fmt.Errorf("❌ publish error: %s", string(body))
		}

		fmt.Println("✅ Published via HTTP")
		fmt.Println(string(body))
		// Record local usage
		parser := auth.NewTokenParser()
		claims, err := parser.ParseToken(token.Token)
		if err == nil {
			limiter, err := ratelimit.NewRateLimiter(claims)
			if err == nil {
				// TODO:: as per the message itself.
				_ = limiter.RecordPublish(config.DefaultMaxMessageSize) // ignore error silently
			}
		}
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVar(&pubTopic, "topic", "", "Topic to publish to")
	publishCmd.Flags().StringVar(&pubMessage, "message", "", "Message string (used to estimate message size)")
	publishCmd.MarkFlagRequired("topic")     //nolint:errcheck
	publishCmd.MarkFlagRequired("algorithm") //nolint:errcheck
	rootCmd.AddCommand(publishCmd)
}
