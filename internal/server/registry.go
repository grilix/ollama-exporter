package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ollama-exporter/internal/routes"
)

// Router wraps gin.Engine and provides a registration helper.
type Router struct{ engine *gin.Engine }

func NewRouter(engine *gin.Engine) *Router { return &Router{engine: engine} }

func (r *Router) RegisterRoutes(routes []routes.Route) {
	for _, rt := range routes {
		switch rt.Method {
		case http.MethodGet:
			r.engine.GET(rt.Path, rt.Handler)
		case http.MethodPost:
			r.engine.POST(rt.Path, rt.Handler)
		case http.MethodPut:
			r.engine.PUT(rt.Path, rt.Handler)
		case http.MethodDelete:
			r.engine.DELETE(rt.Path, rt.Handler)
		}
	}
}

func (r *Router) NoRoute(h gin.HandlerFunc) { r.engine.NoRoute(h) }
