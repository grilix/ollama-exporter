package routes_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"ollama-exporter/internal/routes"
)

func noopHandler(c *gin.Context) {}

func TestRouteInitialization(t *testing.T) {
	// Test that Route struct can be created without panic
	r := routes.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: noopHandler,
	}

	// If we reach here, initialization succeeded
	if r.Method == "" {
		t.Error("Expected Method to be set")
	}
}

func TestRouteFieldAccessors(t *testing.T) {
	r := routes.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: noopHandler,
	}

	if r.Method != "GET" {
		t.Errorf("Expected Method to be 'GET', got '%s'", r.Method)
	}

	if r.Path != "/test" {
		t.Errorf("Expected Path to be '/test', got '%s'", r.Path)
	}

	if r.Handler == nil {
		t.Error("Expected Handler to be non-nil")
	}
}

func TestRouteMethodVariations(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			r := routes.Route{
				Method:  method,
				Path:    "/test",
				Handler: noopHandler,
			}

			if r.Method != method {
				t.Errorf("Expected Method to be '%s', got '%s'", method, r.Method)
			}
		})
	}
}

func TestRoutePathPatterns(t *testing.T) {
	paths := []string{
		"/users",
		"/users/:id",
		"/products/:category/:id",
		"/search/*term",
		"/api/v1/*",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			r := routes.Route{
				Method:  "GET",
				Path:    path,
				Handler: noopHandler,
			}

			if r.Path != path {
				t.Errorf("Expected Path to be '%s', got '%s'", path, r.Path)
			}
		})
	}
}

func TestRouteNilHandler(t *testing.T) {
	r := routes.Route{
		Method:  "GET",
		Path:    "/test",
		Handler: nil,
	}

	// Should not panic and handler should be nil
	if r.Handler != nil {
		t.Error("Expected Handler to be nil")
	}
}

func TestRouteEmptyStrings(t *testing.T) {
	r := routes.Route{
		Method:  "",
		Path:    "",
		Handler: noopHandler,
	}

	if r.Method != "" {
		t.Errorf("Expected Method to be empty, got '%s'", r.Method)
	}

	if r.Path != "" {
		t.Errorf("Expected Path to be empty, got '%s'", r.Path)
	}
}
