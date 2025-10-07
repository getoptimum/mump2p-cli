package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	listServiceURL string // optional service URL override
)

// ListResponse represents the response from the /api/v1/topics endpoint
type ListResponse struct {
	ClientID string   `json:"client_id"`
	Topics   []string `json:"topics"`
	Count    int      `json:"count"`
}

var listTopicsCmd = &cobra.Command{
	Use:   "list-topics",
	Short: "List subscribed topics for the authenticated client",
	Long: `List all topics that the authenticated client is currently subscribed to.
This command shows your active topics and their count.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsAuthDisabled() {
			// When auth is disabled, require client-id flag
			clientIDToUse := GetClientID()
			if clientIDToUse == "" {
				return fmt.Errorf("--client-id is required when using --disable-auth")
			}

			// Determine service URL
			serviceURL := config.LoadConfig().ServiceUrl
			if listServiceURL != "" {
				serviceURL = listServiceURL
				fmt.Printf("Using custom service URL: %s\n", serviceURL)
			}

			// Create HTTP GET request to /api/v1/topics with client_id query parameter
			endpoint := fmt.Sprintf("%s/api/v1/topics?client_id=%s", serviceURL, clientIDToUse)
			req, err := http.NewRequest("GET", endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to create HTTP request: %v", err)
			}

			// Set headers (no auth needed for disabled auth)
			req.Header.Set("Content-Type", "application/json")

			// Execute the request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("HTTP GET request failed: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %v", err)
			}

			// Check for HTTP errors
			if resp.StatusCode != 200 {
				return fmt.Errorf("HTTP GET request error (status %d): %s", resp.StatusCode, string(body))
			}

			// Parse the JSON response
			var listResponse ListResponse
			if err := json.Unmarshal(body, &listResponse); err != nil {
				return fmt.Errorf("failed to parse response JSON: %v", err)
			}

			// Display results in a formatted table
			fmt.Printf("\n📋 Subscribed Topics for Client: %s (Auth Disabled)\n", listResponse.ClientID)
			fmt.Printf("═══════════════════════════════════════════════════════════════\n")

			if listResponse.Count == 0 {
				fmt.Printf("   No active topics found.\n")
				fmt.Printf("   Use './mump2p subscribe --topic=<topic-name>' to subscribe to a topic.\n")
			} else {
				fmt.Printf("   Total Topics: %d\n\n", listResponse.Count)
				for i, topic := range listResponse.Topics {
					fmt.Printf("   %d. %s\n", i+1, topic)
				}
			}

			fmt.Printf("═══════════════════════════════════════════════════════════════\n")
			return nil
		}

		// Authenticate
		authClient := auth.NewClient()
		storage := auth.NewStorageWithPath(GetAuthPath())
		token, err := authClient.GetValidToken(storage)
		if err != nil {
			return fmt.Errorf("authentication required: %v", err)
		}

		// Parse token to get client ID and check if account is active
		parser := auth.NewTokenParser()
		claims, err := parser.ParseToken(token.Token)
		if err != nil {
			return fmt.Errorf("error parsing token: %v", err)
		}

		// Check if the account is active
		if !claims.IsActive {
			return fmt.Errorf("your account is inactive, please contact support")
		}

		// Determine service URL
		serviceURL := config.LoadConfig().ServiceUrl
		if listServiceURL != "" {
			serviceURL = listServiceURL
			fmt.Printf("Using custom service URL: %s\n", serviceURL)
		}

		// Create HTTP GET request to /api/v1/topics with client_id query parameter
		// This provides fallback for servers that don't extract client_id from JWT claims
		endpoint := fmt.Sprintf("%s/api/v1/topics?client_id=%s", serviceURL, claims.ClientID)
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %v", err)
		}

		// Set authorization header
		req.Header.Set("Authorization", "Bearer "+token.Token)
		req.Header.Set("Content-Type", "application/json")

		// Execute the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP GET request failed: %v", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %v", err)
		}

		// Check for HTTP errors
		if resp.StatusCode != 200 {
			return fmt.Errorf("HTTP GET request error (status %d): %s", resp.StatusCode, string(body))
		}

		// Parse the JSON response
		var listResponse ListResponse
		if err := json.Unmarshal(body, &listResponse); err != nil {
			return fmt.Errorf("failed to parse response JSON: %v", err)
		}

		// Display results in a formatted table
		fmt.Printf("\n📋 Subscribed Topics for Client: %s\n", listResponse.ClientID)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")

		if listResponse.Count == 0 {
			fmt.Printf("   No active topics found.\n")
			fmt.Printf("   Use './mump2p subscribe --topic=<topic-name>' to subscribe to a topic.\n")
		} else {
			fmt.Printf("   Total Topics: %d\n\n", listResponse.Count)
			for i, topic := range listResponse.Topics {
				fmt.Printf("   %d. %s\n", i+1, topic)
			}
		}

		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		return nil
	},
}

func init() {
	listTopicsCmd.Flags().StringVar(&listServiceURL, "service-url", "", "Override the default service URL")
	rootCmd.AddCommand(listTopicsCmd)
}
