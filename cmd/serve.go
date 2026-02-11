package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"ollama-exporter/internal/app"
	"ollama-exporter/internal/config"
)

var (
	ollamaHost    string
	ollamaTimeout string
	exporterAddr  string
	exporterPort  string
	verbose       bool
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the ollama-exporter server",
	Long: `Start the ollama-exporter server that provides Prometheus metrics
and acts as a proxy for Ollama API requests. This is the default command
and maintains backward compatibility with the original CLI behavior.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&ollamaHost, "ollama-host", "", "Ollama server host (default: localhost:11434, env: OLLAMA_HOST)")
	serveCmd.Flags().StringVar(&ollamaTimeout, "ollama-timeout", "", "Ollama request timeout (default: 50m, env: OLLAMA_TIMEOUT)")
	serveCmd.Flags().StringVar(&exporterAddr, "addr", "", "Address to bind to (default: '', env: EXPORTER_ADDR)")
	serveCmd.Flags().StringVar(&exporterPort, "port", "", "Port to bind to (default: 8000, env: EXPORTER_PORT)")
	serveCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
}

func runServe(cmd *cobra.Command, args []string) error {
	env := config.NewEnvConfig()
	cfg := config.NewConfig(env)
	cfg.Version = GetVersion()

	if cmd.Flags().Changed("ollama-host") {
		cfg.OllamaHost = ollamaHost
	}
	if cmd.Flags().Changed("ollama-timeout") {
		if timeout, err := time.ParseDuration(ollamaTimeout); err == nil {
			cfg.OllamaTimeout = timeout
		} else {
			return fmt.Errorf("invalid timeout format: %v", err)
		}
	}
	if cmd.Flags().Changed("addr") {
		cfg.ExporterAddr = exporterAddr
	}
	if cmd.Flags().Changed("port") {
		cfg.ExporterPort = exporterPort
	}

	cfg.Initialize()

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if verbose {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	application, err := app.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return application.Start()
}
