package app

import (
	"backend_task/internal/config"

	"github.com/gin-gonic/gin"
)

type App struct {
	config *config.Config
	router *gin.Engine
}

func New(cfg *config.Config) *App {
	return &App{
		config: cfg,
		router: gin.Default(),
	}
}

func (a *App) Run() error {
	a.setupRoutes()
	return a.router.Run(a.config.Server.Port)
}

func (a *App) setupRoutes() {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
