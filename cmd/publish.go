package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/node"
	"github.com/getoptimum/mump2p-cli/internal/ratelimit"
	"github.com/getoptimum/mump2p-cli/internal/session"
	pb "github.com/getoptimum/mump2p-cli/proto"
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

func shortMsgID(resp *pb.Response) string {
	if resp == nil || len(resp.Data) == 0 {
		return ""
	}
	var trace map[string]interface{}
	if err := json.Unmarshal(resp.Data, &trace); err != nil {
		return ""
	}
	if mid, ok := trace["messageID"].(string); ok && mid != "" {
		if len(mid) > 8 {
			return mid[:8]
		}
		return mid
	}
	if mid, ok := trace["message_id"].(string); ok && mid != "" {
		if len(mid) > 8 {
			return mid[:8]
		}
		return mid
	}
	return ""
}

// payloadSource describes where the publish payload should come from, as
// derived from the command flags.
type payloadSource struct {
	message    string
	messageSet bool // --message was given explicitly (possibly empty)
	filePath   string
	fileSet    bool // --file was given explicitly (possibly empty)
}

// validatePayloadFlags rejects flag combinations that cannot be resolved.
func validatePayloadFlags(src payloadSource) error {
	if src.messageSet && src.fileSet {
		return errors.New("only one of --message or --file should be used at a time")
	}
	if src.fileSet && src.filePath == "" {
		return errors.New("--file requires a path (use - for stdin)")
	}
	return nil
}

// resolvePublishPayload returns the payload from --file, stdin, or --message.
// Stdin (a pipe or redirect, never an interactive terminal) takes precedence
// over --message when it carries data, as requested in #92; --file=- reads
// stdin explicitly. Reads are capped at maxBytes to bound memory use.
func resolvePublishPayload(src payloadSource, stdin *os.File, maxBytes int64) ([]byte, error) {
	if err := validatePayloadFlags(src); err != nil {
		return nil, err
	}

	if src.fileSet && src.filePath != "-" {
		content, err := os.ReadFile(src.filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		return content, nil
	}

	stdinAvailable := src.filePath == "-" || !isTerminal(stdin)
	if stdinAvailable {
		content, err := readStdinBounded(stdin, maxBytes)
		if err != nil {
			return nil, err
		}
		if len(content) > 0 {
			return content, nil
		}
		if src.filePath == "-" || !src.messageSet {
			return nil, errors.New("stdin is empty: nothing to publish")
		}
	}

	if src.messageSet {
		if src.message == "" {
			return nil, errors.New("--message is empty: nothing to publish")
		}
		return []byte(src.message), nil
	}

	return nil, errors.New("no message provided: use --message, --file, or pipe data via stdin")
}

// readStdinBounded reads stdin to EOF, failing if it exceeds maxBytes.
func readStdinBounded(stdin *os.File, maxBytes int64) ([]byte, error) {
	var r io.Reader = stdin
	if maxBytes > 0 {
		r = io.LimitReader(stdin, maxBytes+1)
	}
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %v", err)
	}
	if maxBytes > 0 && int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("stdin exceeds the maximum message size of %d bytes", maxBytes)
	}
	return content, nil
}

// isTerminal reports whether f is an interactive terminal. A nil or
// unreadable file counts as one so we never block on input that cannot arrive.
func isTerminal(f *os.File) bool {
	if f == nil {
		return true
	}
	info, err := f.Stat()
	if err != nil {
		return true
	}
	return info.Mode()&os.ModeCharDevice != 0
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to the Optimum Network",
	Long: `Publish a message to the Optimum Network.

The payload comes from --file, stdin, or --message. Piped stdin is used
whenever it carries data (it takes precedence over --message), so the
command composes in pipes:

  echo "hello" | mump2p publish --topic=test
  curl -s https://api.example.com/status | mump2p publish --topic=status

Use --file=- to read from stdin explicitly.`,
	Example: `  mump2p publish --topic=test --message="Hello World"
  mump2p publish --topic=test/data --file=./payload.json
  cat payload.json | mump2p publish --topic=test/data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := payloadSource{
			message:    pubMessage,
			messageSet: cmd.Flags().Changed("message"),
			filePath:   file,
			fileSet:    cmd.Flags().Changed("file"),
		}
		if err := validatePayloadFlags(src); err != nil {
			return err
		}

		var claims *auth.TokenClaims
		var clientIDToUse string
		var accessToken string
		maxMessageSize := int64(config.DefaultMaxMessageSize)

		if !IsAuthDisabled() {
			authClient := auth.NewClient()
			storage := auth.NewStorageWithPath(GetAuthPath())
			token, err := authClient.GetValidToken(storage)
			if err != nil {
				return fmt.Errorf("authentication required: %v", err)
			}
			accessToken = token.Token
			parser := auth.NewTokenParser()
			claims, err = parser.ParseToken(token.Token)
			if err != nil {
				return fmt.Errorf("error parsing token: %v", err)
			}
			if !claims.IsActive {
				return fmt.Errorf("your account is inactive, please contact support")
			}
			clientIDToUse = claims.ClientID
			if claims.MaxMessageSize > 0 {
				maxMessageSize = claims.MaxMessageSize
			}
		} else {
			clientIDToUse = GetClientID()
			if clientIDToUse == "" {
				return fmt.Errorf("--client-id is required when using --disable-auth")
			}
		}

		// Read the payload only after the size limit is known so stdin is bounded.
		data, err := resolvePublishPayload(src, os.Stdin, maxMessageSize)
		if err != nil {
			return err
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

		sessionStart := time.Now()
		sess, reused, err := session.GetOrCreateSession(
			proxyURL,
			clientIDToUse,
			accessToken,
			[]string{pubTopic},
			[]string{"publish"},
			pubExposeAmount,
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

		var published bool
		for i, n := range sess.Nodes {
			if IsDebugMode() {
				fmt.Printf("  Trying node %d/%d: %s (%s, score: %.2f)...\n",
					i+1, len(sess.Nodes), n.Address, n.Region, n.Score)
			}

			nodeAddr := extractIPFromURL(n.Address)
			if nodeAddr == "" {
				nodeAddr = n.Address
			}

			publishData := data
			if IsDebugMode() {
				publishData = addDebugPrefix(data, nodeAddr)
			}

			connectStart := time.Now()
			nc, connErr := node.NewClient(n.Address)
			if connErr != nil {
				fmt.Printf("  Node %s unreachable: %v\n", n.Address, connErr)
				continue
			}

			ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
			resp, pubErr := nc.Publish(ctx, n.Ticket, pubTopic, publishData)
			rpcDur := time.Since(connectStart)
			ctxCancel()
			nc.Close()

			if pubErr != nil {
				fmt.Printf("  Node %s failed: %v\n", n.Address, pubErr)
				continue
			}

			if IsDebugMode() {
				printDebugInfo(publishData, nodeAddr, pubTopic)
			}

			region := n.Region
			if region == "" {
				region = "unknown"
			}

			msgID := shortMsgID(resp)
			suffix := ""
			if msgID != "" {
				suffix = fmt.Sprintf(" [msg: %s]", msgID)
			}

			if IsDebugMode() && !reused {
				fmt.Printf("Session: %s | Published: %s | Total: %s\n",
					humanDuration(sessionDur), humanDuration(rpcDur),
					humanDuration(sessionDur+rpcDur))
			}

			fmt.Printf("Published to %s (%s) in %s%s\n",
				n.Address, region, humanDuration(rpcDur), suffix)

			if IsDebugMode() && resp != nil && msgID != "" {
				var trace map[string]interface{}
				if json.Unmarshal(resp.Data, &trace) == nil {
					if mid, ok := trace["messageID"].(string); ok && mid != "" {
						fmt.Printf("  message-id: %s\n", mid)
					} else if mid, ok := trace["message_id"].(string); ok && mid != "" {
						fmt.Printf("  message-id: %s\n", mid)
					}
				}
			}

			published = true
			break
		}

		if !published {
			return fmt.Errorf("all %d node(s) failed to publish", len(sess.Nodes))
		}

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
	publishCmd.Flags().StringVar(&pubMessage, "message", "", "Message string to publish (reads stdin if neither --message nor --file is set)")
	publishCmd.Flags().StringVar(&file, "file", "", "Path of the file to publish (use - for stdin)")
	publishCmd.Flags().StringVar(&serviceURL, "service-url", "", "Override the default proxy URL")
	publishCmd.Flags().Uint32Var(&pubExposeAmount, "expose-amount", 1, "Number of nodes to request from proxy")
	publishCmd.MarkFlagRequired("topic") //nolint:errcheck
	rootCmd.AddCommand(publishCmd)
}
