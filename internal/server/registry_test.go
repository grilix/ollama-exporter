package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"ollama-exporter/internal/routes"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}

	if router.engine != engine {
		t.Error("NewRouter() should return router with the given engine")
	}
}

func TestRegisterRoutes_GET(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Create a test handler
	testHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "GET route")
	}

	// Register a GET route
	routes := []routes.Route{
		{
			Method:  http.MethodGet,
			Path:    "/test-get",
			Handler: testHandler,
		},
	}
	router.RegisterRoutes(routes)

	// Create a request to test the route
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-get", nil)

	// Perform the request
	engine.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "GET route" {
		t.Errorf("Expected body 'GET route', got '%s'", w.Body.String())
	}
}

func TestRegisterRoutes_POST(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Create a test handler
	testHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "POST route")
	}

	// Register a POST route
	routes := []routes.Route{
		{
			Method:  http.MethodPost,
			Path:    "/test-post",
			Handler: testHandler,
		},
	}
	router.RegisterRoutes(routes)

	// Create a request to test the route
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test-post", nil)

	// Perform the request
	engine.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "POST route" {
		t.Errorf("Expected body 'POST route', got '%s'", w.Body.String())
	}
}

func TestRegisterRoutes_PUT(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Create a test handler
	testHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "PUT route")
	}

	// Register a PUT route
	routes := []routes.Route{
		{
			Method:  http.MethodPut,
			Path:    "/test-put",
			Handler: testHandler,
		},
	}
	router.RegisterRoutes(routes)

	// Create a request to test the route
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/test-put", nil)

	// Perform the request
	engine.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "PUT route" {
		t.Errorf("Expected body 'PUT route', got '%s'", w.Body.String())
	}
}

func TestRegisterRoutes_DELETE(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Create a test handler
	testHandler := func(c *gin.Context) {
		c.String(http.StatusOK, "DELETE route")
	}

	// Register a DELETE route
	routes := []routes.Route{
		{
			Method:  http.MethodDelete,
			Path:    "/test-delete",
			Handler: testHandler,
		},
	}
	router.RegisterRoutes(routes)

	// Create a request to test the route
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/test-delete", nil)

	// Perform the request
	engine.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "DELETE route" {
		t.Errorf("Expected body 'DELETE route', got '%s'", w.Body.String())
	}
}

func TestRegisterRoutes_EmptySlice(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Register empty routes slice - should not panic
	router.RegisterRoutes([]routes.Route{})

	// Create a request to check that no routes are registered
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

	// Perform the request - should return 404 as no routes are registered
	engine.ServeHTTP(w, req)

	// Check the response is 404 (no route found)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d for non-existent route, got %d", http.StatusNotFound, w.Code)
	}
}

func TestNoRouteHandler(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router := NewRouter(engine)

	// Create a custom no-route handler
	noRouteHandler := func(c *gin.Context) {
		c.String(http.StatusNotFound, "Custom 404")
	}

	// Set the no-route handler
	router.NoRoute(noRouteHandler)

	// Create a request to a non-existent route
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)

	// Perform the request
	engine.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	if w.Body.String() != "Custom 404" {
		t.Errorf("Expected body 'Custom 404', got '%s'", w.Body.String())
	}
}
