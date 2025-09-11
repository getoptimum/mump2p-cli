package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/auth"
	"github.com/getoptimum/mump2p-cli/internal/config"
	grpcclient "github.com/getoptimum/mump2p-cli/internal/grpc"
	"github.com/getoptimum/mump2p-cli/internal/ratelimit"
	"github.com/spf13/cobra"
)

var (
	pubTopic   string
	pubMessage string
	file       string
	//optional
	serviceURL string
	useGRPCPub bool // gRPC flag for publish
)

// PublishPayload matches the expected JSON body on the server
type PublishRequest struct {
	ClientID  string `json:"client_id"`
	Topic     string `json:"topic"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// addDebugPrefix adds debug information prefix to message data
func addDebugPrefix(data []byte, proxyAddr string) []byte {
	currentTime := time.Now().UnixNano()
	prefix := fmt.Sprintf("sender_addr:%s\t[send_time, size]:[%d, %d]\t", proxyAddr, currentTime, len(data))
	prefixBytes := []byte(prefix)
	return append(prefixBytes, data...)
}

// printDebugInfo prints debug information for publish operations
func printDebugInfo(data []byte, proxyAddr string, topic string, isGRPC bool) {
	currentTime := time.Now().UnixNano()
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	protocol := "HTTP"
	if isGRPC {
		protocol = "gRPC"
	}
	fmt.Printf("Publish:\tsender_info:%s, [send_time, size]:[%d, %d]\ttopic:%s\tmsg_hash:%s\tprotocol:%s\n",
		proxyAddr, currentTime, len(data), topic, hash[:8], protocol)
}

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to the OptimumP2P via HTTP or gRPC",
	RunE: func(cmd *cobra.Command, args []string) error {
		if pubMessage == "" && file == "" {
			return errors.New("either --message or --file must be provided")
		}
		if pubMessage != "" && file != "" {
			return errors.New("only one of --message or --file should be used at a time")
		}

		authClient := auth.NewClient()
		storage := auth.NewStorageWithPath(GetAuthPath())
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
		// message size
		messageSize := int64(len(data))

		limiter, err := ratelimit.NewRateLimiterWithDir(claims, GetAuthDir())
		if err != nil {
			return fmt.Errorf("rate limiter setup failed: %v", err)
		}

		// check all rate limits: size, quota, per-hr, per-sec
		if err := limiter.CheckPublishAllowed(messageSize); err != nil {
			return err
		}

		// use custom service URL if provided, otherwise use the default
		baseURL := config.LoadConfig().ServiceUrl
		if serviceURL != "" {
			baseURL = serviceURL
		}

		if useGRPCPub {
			// gRPC publish logic
			grpcAddr := strings.Replace(baseURL, "http://", "", 1)
			grpcAddr = strings.Replace(grpcAddr, "https://", "", 1)
			// Replace the port with 50051 for gRPC (default gRPC port)
			if strings.Contains(grpcAddr, ":") {
				// Extract host part and append gRPC port
				host := strings.Split(grpcAddr, ":")[0]
				grpcAddr = host + ":50051"
			} else {
				grpcAddr += ":50051" // default port if not specified
			}

			// Extract proxy IP for debug mode
			proxyAddr := extractIPFromURL(grpcAddr)
			if proxyAddr == "" {
				proxyAddr = grpcAddr // fallback to full address if no IP found
			}

			// Add debug prefix to data if debug mode is enabled
			publishData := data
			if IsDebugMode() {
				publishData = addDebugPrefix(data, proxyAddr)
			}

			ctx := context.Background()
			client, err := grpcclient.NewProxyClient(grpcAddr)
			if err != nil {
				return fmt.Errorf("failed to connect to gRPC proxy: %v", err)
			}
			defer client.Close()

			err = client.Publish(ctx, claims.ClientID, pubTopic, publishData)
			if err != nil {
				return fmt.Errorf("gRPC publish failed: %v", err)
			}

			// Print debug information if debug mode is enabled
			if IsDebugMode() {
				printDebugInfo(publishData, proxyAddr, pubTopic, true)
			}

			fmt.Println("✅ Published via gRPC", source)
		} else {
			// HTTP publish logic (existing)
			// Extract proxy IP for debug mode
			proxyAddr := extractIPFromURL(baseURL)
			if proxyAddr == "" {
				proxyAddr = baseURL // fallback to full URL if no IP found
			}

			// Add debug prefix to data if debug mode is enabled
			publishData := data
			if IsDebugMode() {
				publishData = addDebugPrefix(data, proxyAddr)
			}

			reqData := PublishRequest{
				ClientID:  claims.ClientID,
				Topic:     pubTopic,
				Message:   string(publishData), // use modified data with debug prefix if enabled
				Timestamp: time.Now().UnixMilli(),
			}
			reqBytes, err := json.Marshal(reqData)
			if err != nil {
				return fmt.Errorf("failed to marshal publish request: %v", err)
			}

			url := baseURL + "/api/v1/publish"
			req, err := http.NewRequest("POST", url, strings.NewReader(string(reqBytes)))
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token.Token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("HTTP publish failed: %v", err)
			}
			defer resp.Body.Close() //nolint:errcheck
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 {
				return fmt.Errorf("publish error: %s", string(body))
			}

			// Print debug information if debug mode is enabled
			if IsDebugMode() {
				printDebugInfo(publishData, proxyAddr, pubTopic, false)
			}

			fmt.Println("✅ Published via HTTP", source)
			fmt.Println(string(body))
		}

		if limiter, err := ratelimit.NewRateLimiterWithDir(claims, GetAuthDir()); err == nil {
			_ = limiter.RecordPublish(messageSize) // update quota (fail silently)
		}
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVar(&pubTopic, "topic", "", "Topic to publish to")
	publishCmd.Flags().StringVar(&pubMessage, "message", "", "Message string (should be more than allowed size)")
	publishCmd.Flags().StringVar(&file, "file", "", "File (should be more than allowed size)")
	publishCmd.Flags().StringVar(&serviceURL, "service-url", "", "Override the default service URL")
	publishCmd.Flags().BoolVar(&useGRPCPub, "grpc", false, "Use gRPC for publishing instead of HTTP")
	publishCmd.MarkFlagRequired("topic") //nolint:errcheck
	rootCmd.AddCommand(publishCmd)
}
