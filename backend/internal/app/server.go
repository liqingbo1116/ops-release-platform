package app

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/api"
	"ops-release-platform/backend/internal/config"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/repository"
)

type Server struct {
	config config.Config
}

func NewServer(cfg config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) Run() error {
	var database *gorm.DB
	if s.config.DatabaseDSN != "" {
		db, err := repository.ConnectAndMigrate(s.config.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("database migration failed: %w", err)
		}
		database = db
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

	integrations, err := integration.NewSuite(integration.Config{Mode: s.config.IntegrationMode})
	if err != nil {
		return fmt.Errorf("integration init failed: %w", err)
	}

	repo := api.BuildRepository(database)
	router := api.NewRouter(repo, queue, agent.NewProtocolStore(), integrations)
	return router.Run(fmt.Sprintf(":%s", s.config.Port))
}
