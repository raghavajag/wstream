package app

import (
	"backend_task/internal/config"
	"backend_task/internal/handlers"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type App struct {
	router *gin.Engine
	config *config.Config
}

func New(cfg *config.Config) *App {
	router := gin.New()
	router.Use(gin.Recovery())

	wsHandler := handlers.NewWebSocketHandler(cfg)

	router.GET("/ws", wsHandler.HandleWebSocket)

	return &App{
		router: router,
		config: cfg,
	}
}

func (a *App) Run() error {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	serverAddr := fmt.Sprintf(":%d", a.config.Port)
	log.Printf("Starting server on %s", serverAddr)
	return a.router.Run(serverAddr)
}
