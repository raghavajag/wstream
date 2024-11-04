package main

import (
	"backend_task/internal/config"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	port := cfg.Server.Port
	log.Printf("Starting server on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatal(err)
	}
}
