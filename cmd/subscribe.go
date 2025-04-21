package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getoptimum/optcli/internal/auth"
	"github.com/getoptimum/optcli/internal/config"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var (
	subTopic    string
	persistPath string
	webhookURL  string
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to a topic via WebSocket",
	RunE: func(cmd *cobra.Command, args []string) error {
		// auth
		authClient := auth.NewClient()
		storage := auth.NewStorage()
		token, err := authClient.GetValidToken(storage)
		if err != nil {
			return fmt.Errorf("authentication required: %v", err)
		}
		// parse token to check if the account is active
		parser := auth.NewTokenParser()
		claims, err := parser.ParseToken(token.Token)
		if err != nil {
			return fmt.Errorf("error parsing token: %v", err)
		}
		// check if the account is active
		if !claims.IsActive {
			return fmt.Errorf("your account is inactive, please contact support")
		}

		// setup persistence if path is provided
		var persistFile *os.File
		if persistPath != "" {
			// create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(persistPath), 0755); err != nil {
				return fmt.Errorf("failed to create persistence directory: %v", err)
			}

			// open file for appending
			persistFile, err = os.OpenFile(persistPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open persistence file: %v", err)
			}
			defer persistFile.Close()

			fmt.Printf("Persisting data to: %s\n", persistPath)
		}

		// validate webhook URL if provided
		if webhookURL != "" {
			if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
				return fmt.Errorf("webhook URL must start with http:// or https://")
			}
			fmt.Printf("Forwarding messages to webhook: %s\n", webhookURL)
		}

		//signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		srcUrl := config.LoadConfig().ServiceUrl
		// setup ws connection
		fmt.Println("Opening WebSocket connection...")

		// convert HTTP URL to WebSocket URL
		wsURL := strings.Replace(srcUrl, "http://", "ws://", 1)
		wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
		wsURL = fmt.Sprintf("%s/ws/api/v1/subscribe/%s", wsURL, subTopic)

		// setup ws headers for authentication
		header := http.Header{}
		header.Set("Authorization", "Bearer "+token.Token)

		// connect
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			return fmt.Errorf("websocket connection failed: %v", err)
		}
		defer conn.Close()

		fmt.Printf("Listening for messages on topic '%s'... Press Ctrl+C to exit\n", subTopic)

		// message receiver goroutine
		doneChan := make(chan struct{})
		go func() {
			defer close(doneChan)
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						fmt.Printf("WebSocket read error: %v\n", err)
					}
					return
				}
				msgStr := string(msg)
				fmt.Println(msgStr)

				// handle persistence to file
				if persistFile != nil {
					timestamp := time.Now().Format(time.RFC3339)
					if _, err := persistFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msgStr)); err != nil {
						fmt.Printf("Error writing to persistence file: %v\n", err)
					}
				}

				// forward to webhook if configured
				if webhookURL != "" {
					go func(payload []byte) {
						resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
						if err != nil {
							fmt.Printf("Webhook forwarding error: %v\n", err)
							return
						}
						defer resp.Body.Close()

						if resp.StatusCode >= 400 {
							fmt.Printf("Webhook responded with status code: %d\n", resp.StatusCode)
						}
					}(msg)
				}
			}
		}()

		// wait for interrupt signal or connection close
		select {
		case <-sigChan:
			fmt.Println("\nClosing connection...")
			// cleanly close the connection by sending a close message
			err := conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return fmt.Errorf("error closing connection: %v", err)
			}
		case <-doneChan:
			fmt.Println("\nConnection closed by server")
		}

		return nil
	},
}

func init() {
	subscribeCmd.Flags().StringVar(&subTopic, "topic", "", "Topic to subscribe to")
	subscribeCmd.MarkFlagRequired("topic")
	subscribeCmd.Flags().StringVar(&persistPath, "persist", "", "Path to file where messages will be stored")
	subscribeCmd.Flags().StringVar(&webhookURL, "webhook", "", "URL to forward messages to")
	rootCmd.AddCommand(subscribeCmd)
}
