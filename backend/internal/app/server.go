package app

import (
	"context"
	"fmt"
	"log"

	"ops-release-platform/backend/internal/agent"
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

	var queue *agent.Queue
	if s.config.RedisAddr != "" {
		redisQueue, err := agent.NewQueue(s.config.RedisAddr)
		if err != nil {
			return fmt.Errorf("redis queue init failed: %w", err)
		}
		defer redisQueue.Close()
		queue = redisQueue
		queue.StartMockWorker(context.Background())
		log.Println("mock agent worker started")
	}

	router := api.NewRouter(queue)
	return router.Run(fmt.Sprintf(":%s", s.config.Port))
}
