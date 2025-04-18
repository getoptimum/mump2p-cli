package cmd

import (
	"fmt"
	"time"

	"github.com/getoptimum/optcli/internal/auth"
	"github.com/spf13/cobra"
)

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
		storage := auth.NewStorage()
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
		storage := auth.NewStorage()
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
		// load token
		storage := auth.NewStorage()
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

		// display token information
		fmt.Println("Authentication Status:")
		fmt.Println("----------------------")

		if claims.Subject != "" {
			fmt.Printf("Client ID: %s\n", claims.Subject)
		}

		fmt.Printf("Expires: %s\n", claims.ExpiresAt.Format(time.RFC822))

		if time.Now().After(claims.ExpiresAt) {
			fmt.Println("Token has expired. Please login again.")
		} else {
			fmt.Printf("Valid for: %s\n", time.Until(claims.ExpiresAt).Round(time.Minute))
		}

		// display rate limit information
		fmt.Println("\nRate Limits:")
		fmt.Println("------------")
		fmt.Printf("Publish Rate:  %d per hour\n", claims.MaxPublishRate)
		fmt.Printf("Max Message Size: %s\n", claims.MaxMessageSize)
		fmt.Printf("Daily Quota: %s\n", claims.DailyQuota)

		return nil
	},
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the authentication token",
	Long:  `Manually refresh the authentication token before it expires.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// create auth client and storage
		authClient := auth.NewClient()
		storage := auth.NewStorage()

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
