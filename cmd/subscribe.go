package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	grpcsub "github.com/getoptimum/mump2p-cli/internal/grpc"
	"github.com/getoptimum/mump2p-cli/internal/webhook"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var (
	subTopic           string
	persistPath        string
	webhookURL         string
	webhookSchema      string
	webhookQueueSize   int
	webhookTimeoutSecs int
	subThreshold       float32
	//optional
	subServiceURL  string
	useGRPC        bool // <-- new flag
	grpcBufferSize int  // gRPC message buffer size
)

// SubscribeRequest represents the HTTP POST payload
type SubscribeRequest struct {
	ClientID  string  `json:"client_id"`
	Topic     string  `json:"topic"`
	Threshold float32 `json:"threshold,omitempty"`
}

// printDebugReceiveInfo prints debug information for received messages
func printDebugReceiveInfo(message []byte, receiverAddr string, topic string, messageNum int32, protocol string) {
	currentTime := time.Now().UnixNano()
	messageSize := len(message)
	sum := sha256.Sum256(message)
	hash := hex.EncodeToString(sum[:])

	// Extract sender info from message if it contains debug prefix
	sendInfoRegex := regexp.MustCompile(`sender_addr:\d+\.\d+\.\d+\.\d+\t\[send_time, size\]:\[\d+,\s*\d+\]`)
	sendInfo := sendInfoRegex.FindString(string(message))

	fmt.Printf("Recv:\t[%d]\treceiver_addr:%s\t[recv_time, size]:[%d, %d]\t%s\ttopic:%s\thash:%s\tprotocol:%s\n",
		messageNum, receiverAddr, currentTime, messageSize, sendInfo, topic, hash[:8], protocol)
}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to a topic via WebSocket or gRPC stream",
	RunE: func(cmd *cobra.Command, args []string) error {
		var claims *auth.TokenClaims
		var token *auth.StoredToken
		var clientIDToUse string

		if !IsAuthDisabled() {
			// auth
			authClient := auth.NewClient()
			storage := auth.NewStorageWithPath(GetAuthPath())
			var err error
			token, err = authClient.GetValidToken(storage)
			if err != nil {
				return fmt.Errorf("authentication required: %v", err)
			}
			// parse token to check if the account is active
			parser := auth.NewTokenParser()
			claims, err = parser.ParseToken(token.Token)
			if err != nil {
				return fmt.Errorf("error parsing token: %v", err)
			}
			// check if the account is active
			if !claims.IsActive {
				return fmt.Errorf("your account is inactive, please contact support")
			}
			clientIDToUse = claims.ClientID
		} else {
			// When auth is disabled, require client-id flag
			clientIDToUse = GetClientID()
			if clientIDToUse == "" {
				return fmt.Errorf("--client-id is required when using --disable-auth")
			}
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
			defer persistFile.Close() //nolint:errcheck

			fmt.Printf("Persisting data to: %s\n", persistPath)
		}

		// validate webhook URL and schema if provided
		var webhookFormatter *webhook.TemplateFormatter
		if webhookURL != "" {
			if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
				return fmt.Errorf("webhook URL must start with http:// or https://")
			}

			// Create template formatter
			formatter, err := webhook.NewTemplateFormatter(webhookSchema)
			if err != nil {
				return fmt.Errorf("invalid webhook schema: %v", err)
			}
			webhookFormatter = formatter

			if webhookSchema == "" {
				fmt.Printf("Forwarding messages to webhook (raw format): %s\n", webhookURL)
			} else {
				fmt.Printf("Forwarding messages to webhook (custom schema): %s\n", webhookURL)
			}
		}

		//signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		srcUrl := config.LoadConfig().ServiceUrl
		// use custom service URL if provided, otherwise use the default
		if subServiceURL != "" {
			srcUrl = subServiceURL
		}

		// Prepare gRPC address if needed
		var grpcAddr string
		if useGRPC {
			grpcAddr = strings.Replace(srcUrl, "http://", "", 1)
			grpcAddr = strings.Replace(grpcAddr, "https://", "", 1)
			// Replace the port with 50051 for gRPC (default gRPC port)
			if strings.Contains(grpcAddr, ":") {
				// Extract host part and append gRPC port
				host := strings.Split(grpcAddr, ":")[0]
				grpcAddr = host + ":50051"
			} else {
				grpcAddr += ":50051" // default port if not specified
			}
			fmt.Printf("Using gRPC service URL: %s\n", grpcAddr)
		} else {
			fmt.Printf("Using HTTP service URL: %s\n", srcUrl)
		}

		// send subscription request (HTTP or gRPC based on useGRPC flag)
		if useGRPC {
			// Use gRPC for subscription request
			fmt.Println("Sending gRPC subscription request...")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client, err := grpcsub.NewProxyClient(grpcAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to gRPC proxy: %v", err)
			}
			defer client.Close()

			err = client.SubscribeTopic(ctx, clientIDToUse, subTopic, subThreshold)
			if err != nil {
				return fmt.Errorf("gRPC subscribe failed: %v", err)
			}

			fmt.Printf("gRPC subscription successful: subscribed to topic '%s'\n", subTopic)
		} else {
			// Use HTTP for subscription request
			fmt.Println("Sending HTTP POST subscription request...")
			httpEndpoint := fmt.Sprintf("%s/api/v1/subscribe", srcUrl)
			reqData := SubscribeRequest{
				ClientID:  clientIDToUse,
				Topic:     subTopic,
				Threshold: subThreshold,
			}
			reqBytes, err := json.Marshal(reqData)
			if err != nil {
				return fmt.Errorf("failed to marshal subscription request: %v", err)
			}

			req, err := http.NewRequest("POST", httpEndpoint, bytes.NewBuffer(reqBytes))
			if err != nil {
				return fmt.Errorf("failed to create HTTP request: %v", err)
			}
			// Only set Authorization header if auth is enabled
			if !IsAuthDisabled() && token != nil {
				req.Header.Set("Authorization", "Bearer "+token.Token)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("HTTP POST subscribe failed: %v", err)
			}

			defer resp.Body.Close() //nolint:errcheck
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != 200 {
				return fmt.Errorf("HTTP POST subscribe error: %s", string(body))
			}

			fmt.Printf("HTTP POST subscription successful: %s\n", string(body))
		}

		if useGRPC {
			// gRPC streaming logic (reuse the connection from subscription)
			// Extract receiver IP for debug mode
			receiverAddr := extractIPFromURL(grpcAddr)
			if receiverAddr == "" {
				receiverAddr = grpcAddr // fallback to full address if no IP found
			}

			// Create a new context for streaming (separate from subscription context)
			streamCtx, streamCancel := context.WithCancel(context.Background())
			defer streamCancel()

			// Create a new client connection for streaming
			streamClient, err := grpcsub.NewProxyClient(grpcAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to gRPC proxy for streaming: %v", err)
			}
			defer streamClient.Close()

			msgChan, err := streamClient.Subscribe(streamCtx, clientIDToUse, grpcBufferSize)
			if err != nil {
				return fmt.Errorf("gRPC stream subscribe failed: %v", err)
			}

			fmt.Printf("Listening for messages on topic '%s' via gRPC... Press Ctrl+C to exit\n", subTopic)

			// webhook queue and worker (same as before)
			type webhookMsg struct {
				data []byte
			}
			webhookQueue := make(chan webhookMsg, webhookQueueSize)
			go func() {
				for msg := range webhookQueue {
					go func(payload []byte) {
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(webhookTimeoutSecs)*time.Second)
						defer cancel()

						// Format the payload using template
						formattedPayload, err := webhookFormatter.FormatMessage(payload, subTopic, clientIDToUse, "grpc-msg")
						if err != nil {
							fmt.Printf("Failed to format webhook payload: %v\n", err)
							return
						}

						req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(formattedPayload))
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
						defer resp.Body.Close() //nolint:errcheck
						if resp.StatusCode >= 400 {
							fmt.Printf("Webhook responded with status code: %d\n", resp.StatusCode)
						}
					}(msg.data)
				}
			}()

			// receiver
			doneChan := make(chan struct{})
			var messageCount int32
			go func() {
				defer close(doneChan)
				for msg := range msgChan {
					msgStr := string(msg.Message)

					// Print debug information if debug mode is enabled
					if IsDebugMode() {
						n := atomic.AddInt32(&messageCount, 1)
						printDebugReceiveInfo(msg.Message, receiverAddr, subTopic, n, "gRPC")
					} else {
						fmt.Println(msgStr)
					}

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
						case webhookQueue <- webhookMsg{data: msg.Message}:
						default:
							fmt.Println("⚠️ Webhook queue full, message dropped")
						}
					}
				}
			}()

			select {
			case <-sigChan:
				fmt.Println("\nClosing connection...")
				streamCancel()
			case <-doneChan:
				fmt.Println("\nConnection closed by server")
			}
			return nil
		}

		// setup ws connection
		fmt.Println("Opening WebSocket connection...")

		// convert HTTP URL to WebSocket URL
		wsURL := strings.Replace(srcUrl, "http://", "ws://", 1)
		wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
		wsURL = fmt.Sprintf("%s/api/v1/ws?client_id=%s", wsURL, clientIDToUse)

		// Extract receiver IP for debug mode
		receiverAddr := extractIPFromURL(srcUrl)
		if receiverAddr == "" {
			receiverAddr = srcUrl // fallback to full URL if no IP found
		}

		// setup ws headers for authentication
		header := http.Header{}
		// Only set Authorization header if auth is enabled
		if !IsAuthDisabled() && token != nil {
			header.Set("Authorization", "Bearer "+token.Token)
		}

		// connect
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			return fmt.Errorf("websocket connection failed: %v", err)
		}
		defer conn.Close() //nolint:errcheck

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

					// Format the payload using template
					formattedPayload, err := webhookFormatter.FormatMessage(payload, subTopic, clientIDToUse, "ws-msg")
					if err != nil {
						fmt.Printf("Failed to format webhook payload: %v\n", err)
						return
					}

					req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(formattedPayload))
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
					defer resp.Body.Close() //nolint:errcheck

					if resp.StatusCode >= 400 {
						fmt.Printf("Webhook responded with status code: %d\n", resp.StatusCode)
					}
				}(msg.data)
			}
		}()

		// receiver
		doneChan := make(chan struct{})
		var messageCount int32
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

				// Print debug information if debug mode is enabled
				if IsDebugMode() {
					n := atomic.AddInt32(&messageCount, 1)
					printDebugReceiveInfo(msg, receiverAddr, subTopic, n, "WebSocket")
				} else {
					fmt.Println(msgStr)
				}

				// persist
				if persistFile != nil {
					timestamp := time.Now().Format(time.RFC3339)
					if _, err := persistFile.WriteString(fmt.Sprintf("[%s] [%s] %s\n", timestamp, subTopic, msgStr)); err != nil { //nolint:staticcheck
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
	subscribeCmd.MarkFlagRequired("topic") //nolint:errcheck
	subscribeCmd.Flags().StringVar(&persistPath, "persist", "", "Path to file where messages will be stored")
	subscribeCmd.Flags().StringVar(&webhookURL, "webhook", "", "URL to forward messages to")
	subscribeCmd.Flags().StringVar(&webhookSchema, "webhook-schema", "", "JSON template for webhook payload (e.g., '{\"content\":\"{{.Message}}\"}')")
	subscribeCmd.Flags().IntVar(&webhookQueueSize, "webhook-queue-size", 100, "Max number of webhook messages to queue before dropping")
	subscribeCmd.Flags().IntVar(&webhookTimeoutSecs, "webhook-timeout", 3, "Timeout in seconds for each webhook POST request")
	subscribeCmd.Flags().Float32Var(&subThreshold, "threshold", 0.1, "Delivery threshold (0.1 to 1.0)")
	subscribeCmd.Flags().StringVar(&subServiceURL, "service-url", "", "Override the default service URL")
	subscribeCmd.Flags().BoolVar(&useGRPC, "grpc", false, "Use gRPC stream for subscription instead of WebSocket")
	subscribeCmd.Flags().IntVar(&grpcBufferSize, "grpc-buffer-size", 100, "gRPC message buffer size (default: 100)")
	rootCmd.AddCommand(subscribeCmd)
}
