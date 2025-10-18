package cmd

// Batch commands for publishing multiple messages from files
import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	batchFile     string
	batchFormat   string
	batchRate     int
	batchProgress bool
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch operations for publishing messages",
	Long:  `Batch publish multiple messages from files.`,
}

var batchPublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish messages from a file",
	Long: `Publish multiple messages from a file (JSON, TSV, or plain text).

Supported formats:
  - json: JSON array with multiple objects
  - tsv: Tab-separated values with headers (converted to JSON)
  - text: Plain text (one message per line)

Examples:
  # Publish from JSON array file
  mump2p batch publish --file=messages.json --topic=orders

  # From TSV file with headers
  mump2p batch publish --file=data.tsv --topic=users --format=tsv

  # From plain text file with rate limiting (50 msg/s)
  mump2p batch publish --file=messages.txt --topic=logs --format=text --rate=50`,
	RunE: runBatchPublish,
}

func init() {
	rootCmd.AddCommand(batchCmd)
	batchCmd.AddCommand(batchPublishCmd)

	batchPublishCmd.Flags().StringVar(&batchFile, "file", "", "File path to read messages from (required)")
	batchPublishCmd.Flags().StringVar(&batchFormat, "format", "json", "File format: json, tsv, text")
	batchPublishCmd.Flags().IntVar(&batchRate, "rate", 0, "Rate limit in messages per second (0 = unlimited)")
	batchPublishCmd.Flags().BoolVar(&batchProgress, "progress", true, "Show progress updates")

	batchPublishCmd.Flags().StringVar(&pubTopic, "topic", "", "Topic to publish to (required)")
	batchPublishCmd.Flags().BoolVar(&useGRPC, "grpc", false, "Use gRPC instead of HTTP")

	batchPublishCmd.MarkFlagRequired("file")
	batchPublishCmd.MarkFlagRequired("topic")
}

// runBatchPublish executes the batch publish command
func runBatchPublish(cmd *cobra.Command, args []string) error {
	// Read all messages from file based on format
	messages, err := readMessagesFromFile(batchFile, batchFormat)
	if err != nil {
		return fmt.Errorf("failed to read messages: %v", err)
	}

	if batchProgress {
		fmt.Printf("Publishing %d messages\n", len(messages))
	}

	// Process each message with rate limiting
	failedCount := 0
	for i, message := range messages {
		// Apply custom rate limiting if specified
		if batchRate > 0 && i > 0 {
			time.Sleep(time.Second / time.Duration(batchRate))
		}

		// Publish using existing logic
		err := publishSingleMessage(message)
		if err != nil {
			failedCount++
			fmt.Printf("Message %d error: %v\n", i+1, err)
		}
	}

	if failedCount > 0 {
		return fmt.Errorf("batch publish completed with %d errors", failedCount)
	}
	return nil
}

// readMessagesFromFile reads messages from file based on format
func readMessagesFromFile(filepath, format string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	switch format {
	case "json":
		return readJSONArray(file)
	case "tsv":
		return readTSV(file)
	case "text":
		return readText(file)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, tsv, text)", format)
	}
}

// readJSONArray reads JSON array and returns individual JSON objects
func readJSONArray(file *os.File) ([]string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %v", err)
	}

	result := make([]string, len(messages))
	for i, msg := range messages {
		result[i] = string(msg)
	}
	return result, nil
}

// readTSV reads TSV file and converts rows to JSON objects
func readTSV(file *os.File) ([]string, error) {
	reader := csv.NewReader(file)
	reader.Comma = '\t'

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var messages []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		jsonObj := make(map[string]string)
		for i, header := range headers {
			if i < len(record) {
				jsonObj[header] = record[i]
			}
		}

		jsonBytes, err := json.Marshal(jsonObj)
		if err != nil {
			return nil, err
		}
		messages = append(messages, string(jsonBytes))
	}
	return messages, nil
}

// readText reads text file and returns non-empty lines
func readText(file *os.File) ([]string, error) {
	var messages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			messages = append(messages, line)
		}
	}
	return messages, scanner.Err()
}

// publishSingleMessage publishes one message using the existing publish command logic
func publishSingleMessage(message string) error {
	pubMessage = message
	file = ""
	useGRPCPub = useGRPC // Set the gRPC flag for the publish command
	return publishCmd.RunE(publishCmd, []string{})
}
