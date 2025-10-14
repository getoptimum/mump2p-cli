package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/formatter"
	"github.com/spf13/cobra"
)

var (
	healthServiceURL string
)

// HealthResponse represents the response from the health endpoint
type HealthResponse struct {
	Status     string `json:"status" yaml:"status"`
	MemoryUsed string `json:"memory_used" yaml:"memory_used"`
	CPUUsed    string `json:"cpu_used" yaml:"cpu_used"`
	DiskUsed   string `json:"disk_used" yaml:"disk_used"`
	Country    string `json:"country,omitempty" yaml:"country,omitempty"`
	CountryISO string `json:"country_iso,omitempty" yaml:"country_iso,omitempty"`
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health status of the proxy server",
	Long:  `Check the health status and system metrics of the proxy server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// use custom service URL if provided, otherwise use the default
		baseURL := config.LoadConfig().ServiceUrl
		if healthServiceURL != "" {
			baseURL = healthServiceURL
		}

		url := baseURL + "/api/v1/health"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("health check failed: %v", err)
		}
		defer resp.Body.Close() //nolint:errcheck

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("health check error (status %d): %s", resp.StatusCode, string(body))
		}

		var healthResp HealthResponse
		if err := json.Unmarshal(body, &healthResp); err != nil {
			// If JSON parsing fails, just display raw response
			fmt.Println("Health Status:")
			fmt.Println(string(body))
			return nil
		}

		f := formatter.New(GetOutputFormat())

		if f.IsTable() {
			// Display formatted health information (default table format)
			fmt.Println("Proxy Health Status:")
			fmt.Println("-------------------")
			fmt.Printf("Status:      %s\n", healthResp.Status)
			fmt.Printf("Memory Used: %s%%\n", healthResp.MemoryUsed)
			fmt.Printf("CPU Used:    %s%%\n", healthResp.CPUUsed)
			fmt.Printf("Disk Used:   %s%%\n", healthResp.DiskUsed)
			if healthResp.Country != "" {
				fmt.Printf("Country:     %s (%s)\n", healthResp.Country, healthResp.CountryISO)
			}
		} else {
			// JSON or YAML format
			output, err := f.Format(healthResp)
			if err != nil {
				return fmt.Errorf("failed to format output: %v", err)
			}
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	healthCmd.Flags().StringVar(&healthServiceURL, "service-url", "", "Override the default service URL")
	rootCmd.AddCommand(healthCmd)
}
