package routes

import (
	"github.com/gin-gonic/gin"
)

// Route specifies an HTTP route.
type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}
