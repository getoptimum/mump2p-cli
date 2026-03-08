package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/node"
	"github.com/getoptimum/mump2p-cli/internal/ratelimit"
	"github.com/getoptimum/mump2p-cli/internal/session"
	"github.com/spf13/cobra"
)

var (
	pubTopic        string
	pubMessage      string
	file            string
	serviceURL      string
	pubExposeAmount uint32
)

func addDebugPrefix(data []byte, addr string) []byte {
	currentTime := time.Now().UnixNano()
	prefix := fmt.Sprintf("sender_addr:%s\t[send_time, size]:[%d, %d]\t", addr, currentTime, len(data))
	return append([]byte(prefix), data...)
}

func printDebugInfo(data []byte, addr string, topic string) {
	currentTime := time.Now().UnixNano()
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	fmt.Printf("Publish:\tsender_info:%s, [send_time, size]:[%d, %d]\ttopic:%s\tmsg_hash:%s\tprotocol:gRPC-direct\n",
		addr, currentTime, len(data), topic, hash[:8])
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to the Optimum Network",
	RunE: func(cmd *cobra.Command, args []string) error {
		if pubMessage == "" && file == "" {
			return errors.New("either --message or --file must be provided")
		}
		if pubMessage != "" && file != "" {
			return errors.New("only one of --message or --file should be used at a time")
		}

		var claims *auth.TokenClaims
		var clientIDToUse string

		if !IsAuthDisabled() {
			authClient := auth.NewClient()
			storage := auth.NewStorageWithPath(GetAuthPath())
			token, err := authClient.GetValidToken(storage)
			if err != nil {
				return fmt.Errorf("authentication required: %v", err)
			}
			parser := auth.NewTokenParser()
			claims, err = parser.ParseToken(token.Token)
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

		var (
			data   []byte
			source string
		)

		if file != "" {
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read file: %v", err)
			}
			data = content
			source = file
		} else {
			data = []byte(pubMessage)
			source = "inline message"
		}

		messageSize := int64(len(data))

		if !IsAuthDisabled() {
			limiter, err := ratelimit.NewRateLimiterWithDir(claims, GetAuthDir())
			if err != nil {
				return fmt.Errorf("rate limiter setup failed: %v", err)
			}
			if err := limiter.CheckPublishAllowed(messageSize); err != nil {
				return err
			}
		}

		proxyURL := config.LoadConfig().ServiceUrl
		if serviceURL != "" {
			proxyURL = serviceURL
		}

		fmt.Printf("Requesting session from %s...\n", proxyURL)

		sess, err := session.CreateSession(
			proxyURL,
			clientIDToUse,
			[]string{pubTopic},
			[]string{"publish"},
			pubExposeAmount,
		)
		if err != nil {
			return fmt.Errorf("session creation failed: %v", err)
		}

		bestNode := sess.Nodes[0]
		fmt.Printf("Session: %s | Node: %s (score: %.2f)\n",
			sess.SessionID, bestNode.Address, bestNode.Score)

		nodeAddr := extractIPFromURL(bestNode.Address)
		if nodeAddr == "" {
			nodeAddr = bestNode.Address
		}

		publishData := data
		if IsDebugMode() {
			publishData = addDebugPrefix(data, nodeAddr)
		}

		nodeClient, err := node.NewClient(bestNode.Address)
		if err != nil {
			return fmt.Errorf("failed to connect to node: %v", err)
		}
		defer nodeClient.Close()

		ctx := context.Background()
		_, err = nodeClient.Publish(ctx, bestNode.Ticket, pubTopic, publishData)
		if err != nil {
			return fmt.Errorf("publish failed: %v", err)
		}

		if IsDebugMode() {
			printDebugInfo(publishData, nodeAddr, pubTopic)
		}

		fmt.Printf("Published (%s)\n", source)

		if !IsAuthDisabled() {
			if limiter, err := ratelimit.NewRateLimiterWithDir(claims, GetAuthDir()); err == nil {
				_ = limiter.RecordPublish(messageSize)
			}
		}
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVar(&pubTopic, "topic", "", "Topic to publish to")
	publishCmd.Flags().StringVar(&pubMessage, "message", "", "Message string to publish")
	publishCmd.Flags().StringVar(&file, "file", "", "Path of the file to publish")
	publishCmd.Flags().StringVar(&serviceURL, "service-url", "", "Override the default proxy URL")
	publishCmd.Flags().Uint32Var(&pubExposeAmount, "expose-amount", 1, "Number of nodes to request from proxy")
	publishCmd.MarkFlagRequired("topic") //nolint:errcheck
	rootCmd.AddCommand(publishCmd)
}
