package cmd

import (
	"fmt"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/formatter"
	"github.com/spf13/cobra"
)

// WhoamiResponse represents structured authentication status
type WhoamiResponse struct {
	ClientID   string         `json:"client_id" yaml:"client_id"`
	Expires    string         `json:"expires,omitempty" yaml:"expires,omitempty"`
	ValidFor   string         `json:"valid_for,omitempty" yaml:"valid_for,omitempty"`
	IsActive   bool           `json:"is_active" yaml:"is_active"`
	IsExpired  bool           `json:"is_expired,omitempty" yaml:"is_expired,omitempty"`
	AuthMode   string         `json:"auth_mode" yaml:"auth_mode"`
	RateLimits *RateLimitInfo `json:"rate_limits,omitempty" yaml:"rate_limits,omitempty"`
}

// RateLimitInfo represents rate limit information
type RateLimitInfo struct {
	PublishPerHour   int     `json:"publish_per_hour" yaml:"publish_per_hour"`
	PublishPerSec    int     `json:"publish_per_sec" yaml:"publish_per_sec"`
	MaxMessageSizeMB float64 `json:"max_message_size_mb" yaml:"max_message_size_mb"`
	DailyQuotaMB     float64 `json:"daily_quota_mb" yaml:"daily_quota_mb"`
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to the P2P service",
	Long:  `Authenticate using the device authorization flow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// create auth client
		authClient := auth.NewClient()
		// login using device authorization flow
		fmt.Println("Initiating authentication...")
		token, err := authClient.Login()
		if err != nil {
			return err
		}

		// store token
		storage := auth.NewStorageWithPath(GetAuthPath())
		if err := storage.SaveToken(token); err != nil {
			return err
		}

		fmt.Println("\n✅ Successfully authenticated")
		fmt.Printf("Token expires at: %s\n", token.ExpiresAt.Format(time.RFC822))
		return nil
	},
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the P2P service",
	Long:  `Remove the stored authentication token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		storage := auth.NewStorageWithPath(GetAuthPath())
		if err := storage.RemoveToken(); err != nil {
			return err
		}

		fmt.Println("✅ Successfully logged out")
		return nil
	},
}

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authentication status",
	Long:  `Display information about the current authentication token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f := formatter.New(GetOutputFormat())

		if IsAuthDisabled() {
			// Display authentication status when auth is disabled
			clientIDToUse := GetClientID()
			if clientIDToUse == "" {
				clientIDToUse = "(not set - use --client-id flag)"
			}

			response := WhoamiResponse{
				ClientID: clientIDToUse,
				AuthMode: "disabled",
				IsActive: true,
			}

			if f.IsTable() {
				fmt.Println("Authentication Status:")
				fmt.Println("----------------------")
				fmt.Printf("Client ID: %s\n", clientIDToUse)
				fmt.Println("Auth Mode: Disabled (using --disable-auth)")
				fmt.Println("Rate Limits: N/A (no limits enforced)")
				fmt.Println("Token: N/A (auth disabled)")
			} else {
				output, err := f.Format(response)
				if err != nil {
					return fmt.Errorf("failed to format output: %v", err)
				}
				fmt.Println(output)
			}
			return nil
		}

		// load token
		storage := auth.NewStorageWithPath(GetAuthPath())
		token, err := storage.LoadToken()
		if err != nil {
			return fmt.Errorf("not authenticated: %v", err)
		}

		// parse token
		parser := auth.NewTokenParser()
		claims, err := parser.ParseToken(token.Token)
		if err != nil {
			return fmt.Errorf("error parsing token: %v", err)
		}

		// Prepare structured response
		isExpired := time.Now().After(claims.ExpiresAt)
		response := WhoamiResponse{
			ClientID:  claims.Subject,
			Expires:   claims.ExpiresAt.Format(time.RFC822),
			ValidFor:  time.Until(claims.ExpiresAt).Round(time.Minute).String(),
			IsActive:  claims.IsActive,
			IsExpired: isExpired,
			AuthMode:  "enabled",
			RateLimits: &RateLimitInfo{
				PublishPerHour:   claims.MaxPublishPerHour,
				PublishPerSec:    claims.MaxPublishPerSec,
				MaxMessageSizeMB: float64(claims.MaxMessageSize) / (1 << 20),
				DailyQuotaMB:     float64(claims.DailyQuota) / (1 << 20),
			},
		}

		if f.IsTable() {
			// display token information (table format)
			fmt.Println("Authentication Status:")
			fmt.Println("----------------------")

			if claims.Subject != "" {
				fmt.Printf("Client ID: %s\n", claims.Subject)
			}

			fmt.Printf("Expires: %s\n", claims.ExpiresAt.Format(time.RFC822))

			if isExpired {
				fmt.Println("Token has expired. Please login again.")
			} else {
				fmt.Printf("Valid for: %s\n", time.Until(claims.ExpiresAt).Round(time.Minute))
			}

			fmt.Printf("Is Active:  %t\n", claims.IsActive)
			// display rate limit information
			fmt.Println("\nRate Limits:")
			fmt.Println("------------")
			fmt.Printf("Publish Rate:  %d per hour\n", claims.MaxPublishPerHour)
			fmt.Printf("Publish Rate:  %d per second\n", claims.MaxPublishPerSec)
			fmt.Printf("Max Message Size:  %.2f MB\n", float64(claims.MaxMessageSize)/(1<<20))
			fmt.Printf("Daily Quota:       %.2f MB\n", float64(claims.DailyQuota)/(1<<20))
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

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the authentication token",
	Long:  `Manually refresh the authentication token before it expires.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsAuthDisabled() {
			fmt.Println("Token refresh skipped (auth disabled)")
			return nil
		}

		// create auth client and storage
		authClient := auth.NewClient()
		storage := auth.NewStorageWithPath(GetAuthPath())

		// load current token
		token, err := storage.LoadToken()
		if err != nil {
			return fmt.Errorf("not authenticated: %v", err)
		}

		// display current token info
		fmt.Println("Current token status:")
		fmt.Printf("Expires at: %s\n", token.ExpiresAt.Format(time.RFC822))
		fmt.Printf("Valid for:  %s\n", time.Until(token.ExpiresAt).Round(time.Minute))

		// check if refresh token exists
		if token.RefreshToken == "" {
			return fmt.Errorf("no refresh token available, please login again")
		}

		// force refresh token
		fmt.Println("Refreshing token...")
		refreshedToken, err := authClient.RefreshToken(token.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to refresh token: %v", err)
		}

		// save refreshed token
		if err := storage.SaveToken(refreshedToken); err != nil {
			return fmt.Errorf("failed to save refreshed token: %v", err)
		}

		// display new token info
		fmt.Println("✅ Token refreshed successfully")
		fmt.Printf("New expiration: %s\n", refreshedToken.ExpiresAt.Format(time.RFC822))
		fmt.Printf("Valid for:      %s\n", time.Until(refreshedToken.ExpiresAt).Round(time.Minute))

		return nil
	},
}

func init() {
	// add commands to root
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(refreshCmd)
}
