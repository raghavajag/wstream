package app

import (
	"backend_task/internal/config"
	"backend_task/internal/handlers"
	"backend_task/internal/ui"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type App struct {
	router *gin.Engine
	config *config.Config
}

func New(cfg *config.Config) *App {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	// Add WebSocket handler
	wsHandler := handlers.NewWebSocketHandler(cfg)
	router.GET("/ws", wsHandler.HandleWebSocket)

	// Add UI routes
	ui.AddRoutes(router)

	return &App{
		router: router,
		config: cfg,
	}
}

func (a *App) Run() error {
	serverAddr := fmt.Sprintf(":%d", a.config.Port)
	log.Printf("Starting server on %s", serverAddr)
	// to ensure Gin binds to all interfaces
	a.router.SetTrustedProxies(nil)
	return a.router.Run(serverAddr)
}
