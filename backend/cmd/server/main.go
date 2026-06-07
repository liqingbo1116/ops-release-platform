package main

import (
	"log"

	"ops-release-platform/backend/internal/app"
	"ops-release-platform/backend/internal/config"
)

func main() {
	cfg := config.Load()
	server := app.NewServer(cfg)

	if err := server.Run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
