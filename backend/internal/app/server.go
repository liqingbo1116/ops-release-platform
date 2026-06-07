package app

import (
	"fmt"
	"log"

	"ops-release-platform/backend/internal/api"
	"ops-release-platform/backend/internal/config"
	"ops-release-platform/backend/internal/repository"
)

type Server struct {
	config config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) Run() error {
	if s.config.DatabaseDSN != "" {
		if _, err := repository.ConnectAndMigrate(s.config.DatabaseDSN); err != nil {
			return fmt.Errorf("database migration failed: %w", err)
		}
		log.Println("database migration completed")
	}

	router := api.NewRouter()
	return router.Run(fmt.Sprintf(":%s", s.config.Port))
}
