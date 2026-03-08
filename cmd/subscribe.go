package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/entities"
	"github.com/getoptimum/mump2p-cli/internal/node"
	"github.com/getoptimum/mump2p-cli/internal/session"
	"github.com/getoptimum/mump2p-cli/internal/webhook"
	"github.com/spf13/cobra"
)

var (
	subTopic           string
	persistPath        string
	webhookURL         string
	webhookSchema      string
	webhookQueueSize   int
	webhookTimeoutSecs int
	subServiceURL      string
	subExposeAmount    uint32
)

func printDebugReceiveInfo(message []byte, receiverAddr string, topic string, messageNum int32, protocol string) {
	currentTime := time.Now().UnixNano()
	messageSize := len(message)
	sum := sha256.Sum256(message)
	hash := hex.EncodeToString(sum[:])

	sendInfoRegex := regexp.MustCompile(`sender_addr:\d+\.\d+\.\d+\.\d+\t\[send_time, size\]:\[\d+,\s*\d+\]`)
	sendInfo := sendInfoRegex.FindString(string(message))

	fmt.Printf("Recv:\t[%d]\treceiver_addr:%s\t[recv_time, size]:[%d, %d]\t%s\ttopic:%s\thash:%s\tprotocol:%s\n",
		messageNum, receiverAddr, currentTime, messageSize, sendInfo, topic, hash[:8], protocol)
}

func decodeMessage(rawMsg []byte) (decoded []byte, topic string) {
	p2pMsg, err := entities.UnmarshalP2PMessage(rawMsg)
	if err != nil {
		return rawMsg, ""
	}
	return p2pMsg.Message, p2pMsg.Topic
}

func isReadable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 && c != '\n' && c != '\r' && c != '\t' {
			return false
		}
	}
	return len(b) > 0 && utf8.Valid(b)
}

func formatMessage(data []byte) string {
	if isReadable(data) {
		return string(data)
	}
	if len(data) > 256 {
		return fmt.Sprintf("[binary %d bytes] %x...", len(data), data[:128])
	}
	return fmt.Sprintf("[binary %d bytes] %x", len(data), data)
}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Subscribe to a topic and stream messages from the P2P network",
	RunE: func(cmd *cobra.Command, args []string) error {
		var clientIDToUse string

		if !IsAuthDisabled() {
			authClient := auth.NewClient()
			storage := auth.NewStorageWithPath(GetAuthPath())
			token, err := authClient.GetValidToken(storage)
			if err != nil {
				return fmt.Errorf("authentication required: %v", err)
			}
			parser := auth.NewTokenParser()
			claims, err := parser.ParseToken(token.Token)
			if err != nil {
				return fmt.Errorf("error parsing token: %v", err)
			}
			if !claims.IsActive {
				return fmt.Errorf("your account is inactive, please contact support")
			}
			clientIDToUse = claims.ClientID
		} else {
			clientIDToUse = GetClientID()
			if clientIDToUse == "" {
				return fmt.Errorf("--client-id is required when using --disable-auth")
			}
		}

		var persistFile *os.File
		if persistPath != "" {
			fileInfo, err := os.Stat(persistPath)
			if err == nil && fileInfo.IsDir() || strings.HasSuffix(persistPath, "/") || strings.HasSuffix(persistPath, "\\") {
				persistPath = filepath.Join(persistPath, "messages.log")
			}
			if err := os.MkdirAll(filepath.Dir(persistPath), 0755); err != nil {
				return fmt.Errorf("failed to create persistence directory: %v", err)
			}
			persistFile, err = os.OpenFile(persistPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open persistence file: %v", err)
			}
			defer persistFile.Close()
			fmt.Printf("Persisting data to: %s\n", persistPath)
		}

		var webhookFormatter *webhook.TemplateFormatter
		if webhookURL != "" {
			if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
				return fmt.Errorf("webhook URL must start with http:// or https://")
			}
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

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		proxyURL := config.LoadConfig().ServiceUrl
		if subServiceURL != "" {
			proxyURL = subServiceURL
		}

		fmt.Printf("Requesting session from %s...\n", proxyURL)

		sess, err := session.CreateSession(
			proxyURL,
			clientIDToUse,
			[]string{subTopic},
			[]string{"publish", "subscribe"},
			subExposeAmount,
		)
		if err != nil {
			return fmt.Errorf("session creation failed: %v", err)
		}

		bestNode := sess.Nodes[0]
		fmt.Printf("Session: %s | Node: %s (%s, score: %.2f)\n",
			sess.SessionID, bestNode.Address, bestNode.Region, bestNode.Score)
		if len(sess.Nodes) > 1 {
			fmt.Printf("Available nodes: ")
			for i := 1; i < len(sess.Nodes); i++ {
				if i > 1 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s (%s, score: %.2f)", sess.Nodes[i].Address, sess.Nodes[i].Region, sess.Nodes[i].Score)
			}
			fmt.Println()
		}

		nodeClient, err := node.NewClient(bestNode.Address)
		if err != nil {
			return fmt.Errorf("failed to connect to node: %v", err)
		}
		defer nodeClient.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgChan, err := nodeClient.Subscribe(ctx, bestNode.Ticket, subTopic, 100)
		if err != nil {
			return fmt.Errorf("subscribe failed: %v", err)
		}

		fmt.Printf("Subscribed to '%s' — listening for messages. Press Ctrl+C to exit\n", subTopic)

		receiverAddr := extractIPFromURL(bestNode.Address)
		if receiverAddr == "" {
			receiverAddr = bestNode.Address
		}

		type webhookMsg struct {
			data []byte
		}
		wq := make(chan webhookMsg, webhookQueueSize)
		if webhookURL != "" {
			go func() {
				for msg := range wq {
					go func(payload []byte) {
						wctx, wcancel := context.WithTimeout(context.Background(), time.Duration(webhookTimeoutSecs)*time.Second)
						defer wcancel()

						formattedPayload, fmtErr := webhookFormatter.FormatMessage(payload, subTopic, clientIDToUse, "grpc-msg")
						if fmtErr != nil {
							fmt.Printf("Failed to format webhook payload: %v\n", fmtErr)
							return
						}

						req, reqErr := http.NewRequestWithContext(wctx, "POST", webhookURL, bytes.NewBuffer(formattedPayload))
						if reqErr != nil {
							fmt.Printf("Failed to create webhook request: %v\n", reqErr)
							return
						}
						req.Header.Set("Content-Type", "application/json")
						resp, doErr := http.DefaultClient.Do(req)
						if doErr != nil {
							fmt.Printf("Webhook request error: %v\n", doErr)
							return
						}
						defer resp.Body.Close()
						if resp.StatusCode >= 400 {
							fmt.Printf("Webhook responded with status code: %d\n", resp.StatusCode)
						}
					}(msg.data)
				}
			}()
		}

		doneChan := make(chan struct{})
		var messageCount int32
		go func() {
			defer close(doneChan)
			for resp := range msgChan {
				decodedMsg, msgTopic := decodeMessage(resp.Data)

				if msgTopic != "" && msgTopic != subTopic {
					continue
				}

				if IsDebugMode() {
					n := atomic.AddInt32(&messageCount, 1)
					printDebugReceiveInfo(decodedMsg, receiverAddr, subTopic, n, "gRPC-direct")
				} else {
					displayTopic := subTopic
					if msgTopic != "" {
						displayTopic = msgTopic
					}
					fmt.Printf("[%s] %s\n", displayTopic, formatMessage(decodedMsg))
				}

				msgStr := formatMessage(decodedMsg)

				if persistFile != nil {
					timestamp := time.Now().Format(time.RFC3339)
					if _, writeErr := fmt.Fprintf(persistFile, "[%s] %s\n", timestamp, msgStr); writeErr != nil {
						fmt.Printf("Error writing to persistence file: %v\n", writeErr)
					}
				}

				if webhookURL != "" {
					select {
					case wq <- webhookMsg{data: decodedMsg}:
					default:
						fmt.Println("Webhook queue full, message dropped")
					}
				}
			}
		}()

		select {
		case <-sigChan:
			fmt.Println("\nClosing connection...")
			cancel()
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
	subscribeCmd.Flags().StringVar(&webhookSchema, "webhook-schema", "", "JSON template for webhook payload")
	subscribeCmd.Flags().IntVar(&webhookQueueSize, "webhook-queue-size", 100, "Max number of webhook messages to queue before dropping")
	subscribeCmd.Flags().IntVar(&webhookTimeoutSecs, "webhook-timeout", 3, "Timeout in seconds for each webhook POST request")
	subscribeCmd.Flags().StringVar(&subServiceURL, "service-url", "", "Override the default proxy URL")
	subscribeCmd.Flags().Uint32Var(&subExposeAmount, "expose-amount", 1, "Number of nodes to request from proxy (enables failover if >1)")
	rootCmd.AddCommand(subscribeCmd)
}
