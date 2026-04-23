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
	pb "github.com/getoptimum/mump2p-cli/proto"
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

func decodeMessage(rawMsg []byte) (decoded []byte, topic string, p2pMsg *entities.P2PMessage) {
	msg, err := entities.UnmarshalP2PMessage(rawMsg)
	if err != nil {
		return rawMsg, "", nil
	}
	return msg.Message, msg.Topic, msg
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
		var accessToken string

		if !IsAuthDisabled() {
			authClient := auth.NewClient()
			storage := auth.NewStorageWithPath(GetAuthPath())
			token, err := authClient.GetValidToken(storage)
			if err != nil {
				return fmt.Errorf("authentication required: %v", err)
			}
			accessToken = token.Token
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

		sessionStart := time.Now()
		sess, reused, err := session.GetOrCreateSession(
			proxyURL,
			clientIDToUse,
			accessToken,
			[]string{subTopic},
			[]string{"subscribe"},
			subExposeAmount,
		)
		if err != nil {
			return fmt.Errorf("session creation failed: %v", err)
		}
		sessionDur := time.Since(sessionStart)

		if IsDebugMode() {
			if reused {
				fmt.Printf("Reusing session %s | %d node(s) available\n", sess.SessionID, len(sess.Nodes))
			} else {
				fmt.Printf("New session %s from %s (%s) | %d node(s) available\n",
					sess.SessionID, proxyURL, humanDuration(sessionDur), len(sess.Nodes))
			}
		}

		var (
			nodeClient    *node.Client
			connectedNode session.Node
			msgChan       <-chan *pb.Response
		)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		connectStart := time.Now()
		for i, n := range sess.Nodes {
			if IsDebugMode() {
				fmt.Printf("  Trying node %d/%d: %s (%s, score: %.2f)...\n",
					i+1, len(sess.Nodes), n.Address, n.Region, n.Score)
			}

			nc, connErr := node.NewClient(n.Address)
			if connErr != nil {
				fmt.Printf("  Node %s unreachable, falling back...\n", n.Address)
				continue
			}

			ch, subErr := nc.Subscribe(ctx, n.Ticket, subTopic, 100)
			if subErr != nil {
				fmt.Printf("  Node %s subscribe failed, falling back...\n", n.Address)
				nc.Close()
				continue
			}

			nodeClient = nc
			connectedNode = n
			msgChan = ch
			break
		}

		if nodeClient == nil {
			return fmt.Errorf("all %d node(s) failed to connect", len(sess.Nodes))
		}
		defer nodeClient.Close()
		connectDur := time.Since(connectStart)

		region := connectedNode.Region
		if region == "" {
			region = "unknown"
		}

		var backupNodes []session.Node
		for _, n := range sess.Nodes {
			if n.Address != connectedNode.Address {
				backupNodes = append(backupNodes, n)
			}
		}

		backupSuffix := ""
		if len(backupNodes) == 1 {
			backupSuffix = " — 1 backup node ready"
		} else if len(backupNodes) > 1 {
			backupSuffix = fmt.Sprintf(" — %d backup nodes ready", len(backupNodes))
		}

		fmt.Printf("Subscribed to '%s' on %s (%s) in %s%s\n",
			subTopic, connectedNode.Address, region, humanDuration(connectDur), backupSuffix)

		for _, bn := range backupNodes {
			r := bn.Region
			if r == "" {
				r = "unknown"
			}
			fmt.Printf("  backup: %s (%s)\n", bn.Address, r)
		}

		receiverAddr := extractIPFromURL(connectedNode.Address)
		if receiverAddr == "" {
			receiverAddr = connectedNode.Address
		}

		type webhookMsg struct {
			data []byte
		}

		var wq chan webhookMsg
		if webhookURL != "" && webhookQueueSize > 0 {
			wq = make(chan webhookMsg, webhookQueueSize)
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
						if webhookSchema != "" {
							req.Header.Set("Content-Type", "application/json")
						}
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
		subscribeStart := time.Now()

		go func() {
			defer close(doneChan)
			for resp := range msgChan {
				if !IsDebugMode() {
					switch resp.GetCommand() {
					case pb.ResponseType_MessageTraceMumP2P, pb.ResponseType_MessageTraceGossipSub:
						continue
					}
				}

				decodedMsg, msgTopic, p2pMsg := decodeMessage(resp.Data)

				if msgTopic != "" && msgTopic != subTopic {
					continue
				}

				if IsDebugMode() {
					n := atomic.AddInt32(&messageCount, 1)
					printDebugReceiveInfo(decodedMsg, receiverAddr, subTopic, n, "gRPC-direct")
					if p2pMsg != nil {
						if p2pMsg.SourceNodeID != "" {
							fmt.Printf("  from: %s\n", p2pMsg.SourceNodeID)
						}
						fmt.Printf("  via:  %s (%s)\n", connectedNode.Address, region)
						if p2pMsg.MessageID != "" {
							id := p2pMsg.MessageID
							if len(id) > 12 {
								id = id[:12] + "..."
							}
							fmt.Printf("  id:   %s\n", id)
						}
					}
				} else {
					if !isReadable(decodedMsg) {
						continue
					}
					atomic.AddInt32(&messageCount, 1)

					displayTopic := subTopic
					if msgTopic != "" {
						displayTopic = msgTopic
					}
					fmt.Printf("[%s] %s\n", displayTopic, string(decodedMsg))
				}

				// Unreadable payloads are filtered from stdout in non-debug mode above; skip
				// persistence and webhook for them in debug mode too (same as filtered topics).
				if !isReadable(decodedMsg) {
					continue
				}

				msgStr := formatMessage(decodedMsg)

				if persistFile != nil {
					timestamp := time.Now().Format(time.RFC3339)
					if _, writeErr := fmt.Fprintf(persistFile, "[%s] %s\n", timestamp, msgStr); writeErr != nil {
						fmt.Printf("Error writing to persistence file: %v\n", writeErr)
					}
				}

				if wq != nil {
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
			cancel()
		case <-doneChan:
		}

		elapsed := time.Since(subscribeStart)
		count := atomic.LoadInt32(&messageCount)

		throughput := ""
		if elapsed > time.Second && count > 0 {
			rate := float64(count) / elapsed.Seconds()
			throughput = fmt.Sprintf(" (%.1f msg/s)", rate)
		}

		if count == 1 {
			fmt.Printf("\nDisconnected — 1 message in %s%s\n", humanDuration(elapsed), throughput)
		} else {
			fmt.Printf("\nDisconnected — %d messages in %s%s\n", count, humanDuration(elapsed), throughput)
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
	subscribeCmd.Flags().Uint32Var(&subExposeAmount, "expose-amount", 3, "Number of nodes to request from proxy (enables failover if >1)")
	rootCmd.AddCommand(subscribeCmd)
}
