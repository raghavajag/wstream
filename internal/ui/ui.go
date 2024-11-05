package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static
var staticFS embed.FS

// AddRoutes serves the static files for the UI
func AddRoutes(router *gin.Engine) {
	// Create a file system from the embedded files
	staticFiles, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create static file system: %v", err)
	}

	// Create a handler for serving static files
	staticHandler := http.FileServer(http.FS(staticFiles))

	// Serve static files with correct paths
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Serve other static files
	router.GET("./static/*filepath", func(c *gin.Context) {
		// Create a handler that serves the specific file
		handler := http.StripPrefix("/static", staticHandler)

		// Serve the file
		handler.ServeHTTP(c.Writer, c.Request)
	})

	// Optional: Add CORS middleware if needed
	router.Use(corsMiddleware())
}

// CORS middleware to allow WebSocket connections
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
