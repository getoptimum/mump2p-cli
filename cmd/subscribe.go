package cmd

import (
	"bytes"
	"context"
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
	subTopic           string
	persistPath        string
	webhookURL         string
	webhookQueueSize   int
	webhookTimeoutSecs int
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
			// check if persistPath is a directory or ends with a directory separator
			fileInfo, err := os.Stat(persistPath)
			if err == nil && fileInfo.IsDir() || strings.HasSuffix(persistPath, "/") || strings.HasSuffix(persistPath, "\\") {
				// If it's a directory, append a default filename
				persistPath = filepath.Join(persistPath, "messages.log")
			}

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
		// webhook queue and worker
		type webhookMsg struct {
			data []byte
		}
		webhookQueue := make(chan webhookMsg, webhookQueueSize)

		go func() {
			for msg := range webhookQueue {
				go func(payload []byte) {
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(webhookTimeoutSecs)*time.Second)
					defer cancel()

					req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payload))
					if err != nil {
						fmt.Printf("Failed to create webhook request: %v\n", err)
						return
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						fmt.Printf("Webhook request error: %v\n", err)
						return
					}
					defer resp.Body.Close()

					if resp.StatusCode >= 400 {
						fmt.Printf("Webhook responded with status code: %d\n", resp.StatusCode)
					}
				}(msg.data)
			}
		}()

		// receiver
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

				// persist
				if persistFile != nil {
					timestamp := time.Now().Format(time.RFC3339)
					if _, err := persistFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, msgStr)); err != nil {
						fmt.Printf("Error writing to persistence file: %v\n", err)
					}
				}

				// forward
				if webhookURL != "" {
					select {
					case webhookQueue <- webhookMsg{data: msg}:
					default:
						fmt.Println("⚠️ Webhook queue full, message dropped")
					}
				}
			}
		}()

		select {
		case <-sigChan:
			fmt.Println("\nClosing connection...")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
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
	subscribeCmd.Flags().IntVar(&webhookQueueSize, "webhook-queue-size", 100, "Max number of webhook messages to queue before dropping")
	subscribeCmd.Flags().IntVar(&webhookTimeoutSecs, "webhook-timeout", 3, "Timeout in seconds for each webhook POST request")
	rootCmd.AddCommand(subscribeCmd)
}
