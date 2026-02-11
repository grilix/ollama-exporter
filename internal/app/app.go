package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ollama-exporter/internal/config"
	"ollama-exporter/internal/handlers"
	"ollama-exporter/internal/metrics"
	"ollama-exporter/internal/proxy"
	"ollama-exporter/internal/routes"
	"ollama-exporter/internal/server"
)

// App represents the main application
type App struct {
	config        *config.Config
	metrics       *metrics.Metrics
	collector     *metrics.Collector
	server        *server.Server
	healthChecker *HealthChecker
}

// New creates a new application instance
func New(cfg *config.Config) (*App, error) {
	metricsInstance := metrics.NewMetrics()
	collector := metrics.NewCollector(metricsInstance)

	metricsInstance.SetVersion(cfg.Version)

	srv := server.NewServer(cfg.ExporterAddr, cfg.ExporterPort)

	metricsHandler := handlers.NewMetricsHandler()
	forwarder := proxy.NewForwarder(cfg.OllamaURLBase, cfg.OllamaTimeout)
	ollamaHandler := handlers.NewOllamaHandler(collector, forwarder)
	openaiHandler := handlers.NewOpenAIHandler(collector, forwarder)
	proxyHandler, err := proxy.NewProxy(collector, cfg.OllamaURLBase)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy handler: %w", err)
	}

	router := server.NewRouter(srv.Router())

	var routeList []routes.Route
	routeList = append(routeList, routes.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		},
	})

	metricsRoutes := metricsHandler.Routes()
	for _, r := range metricsRoutes {
		routeList = append(routeList, r)
	}

	ollamaRoutes := ollamaHandler.Routes()
	for _, r := range ollamaRoutes {
		routeList = append(routeList, r)
	}

	openaiRoutes := openaiHandler.Routes()
	for _, r := range openaiRoutes {
		routeList = append(routeList, r)
	}

	router.RegisterRoutes(routeList)
	router.NoRoute(proxyHandler.Handler())

	healthChecker := NewHealthChecker(cfg.OllamaURLBase)

	return &App{
		config:        cfg,
		metrics:       metricsInstance,
		collector:     collector,
		server:        srv,
		healthChecker: healthChecker,
	}, nil
}

// Start starts the application
func (a *App) Start() error {
	if err := a.healthChecker.VerifyOllamaConnection(); err != nil {
		log.Printf("Failed to connect to Ollama server: %v", err)
		log.Println("Please ensure Ollama is running and accessible at the configured host")
		return fmt.Errorf("ollama connection failed: %w", err)
	}

	startErrCh := make(chan error, 1)

	go func() {
		if err := a.server.Start(); err != nil && err != http.ErrServerClosed {
			startErrCh <- err
			return
		}
		startErrCh <- nil
	}()

	startErr := <-startErrCh
	if startErr != nil {
		return fmt.Errorf("server startup failed: %w", startErr)
	}

	a.server.WaitForShutdown()

	return nil
}

// Stop stops the application gracefully
func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return a.server.Shutdown(ctx)
}
