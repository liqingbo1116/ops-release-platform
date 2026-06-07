package app

import (
	"fmt"

	"ops-release-platform/backend/internal/api"
	"ops-release-platform/backend/internal/config"
)

type Server struct {
	config config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) Run() error {
	router := api.NewRouter()
	return router.Run(fmt.Sprintf(":%s", s.config.Port))
}
