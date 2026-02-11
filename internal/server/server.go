package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	router     *gin.Engine
	port       string
	addr       string
}

// NewServer creates a new HTTP server instance
func NewServer(addr, port string) *Server {
	router := gin.Default()

	router.Use(gin.Recovery())

	return &Server{
		router: router,
		port:   port,
		addr:   addr,
	}
}

// Router returns the gin router for setting up routes
func (s *Server) Router() *gin.Engine {
	return s.router
}

// NoRoute sets the NoRoute handler for the server
func (s *Server) NoRoute(h gin.HandlerFunc) {
	s.router.NoRoute(h)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := s.getAddr()
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("Ollama exporter listening at %s", addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// WaitForShutdown waits for interrupt signal and gracefully shuts down the server
func (s *Server) WaitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

// getAddr returns the full address for the server
func (s *Server) getAddr() string {
	return fmt.Sprintf("%s:%s", s.addr, s.port)
}
