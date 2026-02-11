package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of ollama-exporter",
	Long:  `Check the health status of the ollama-exporter service.`,
	RunE:  runHealth,
}

var (
	healthURL string
	timeout   string
)

func init() {
	healthCmd.Flags().StringVar(&healthURL, "url", "http://localhost:8000", "URL to check health")
	healthCmd.Flags().StringVar(&timeout, "timeout", "10s", "Timeout for health check")
}

func runHealth(cmd *cobra.Command, args []string) error {
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format: %v", err)
	}

	client := &http.Client{Timeout: timeoutDuration}

	resp, err := client.Get(healthURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	fmt.Println("Health check passed")
	return nil
}
