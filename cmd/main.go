package main

import (
	"backend_task/internal/app"
	"backend_task/internal/config"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
