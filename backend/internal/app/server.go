package app

import (
	"context"
	"fmt"
	"log"

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

func validateRuntimeConfig(cfg config.Config) error {
	if cfg.DatabaseDSN == "" {
		return fmt.Errorf("DATABASE_DSN is required for V1 mainline runtime")
	}
	if cfg.RedisAddr == "" {
		return fmt.Errorf("REDIS_ADDR is required for V1 mainline runtime")
	}
	return nil
}

func (s *Server) Run() error {
	if err := validateRuntimeConfig(s.config); err != nil {
		return err
	}
	database, err := repository.ConnectAndMigrate(s.config.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	log.Println("database migration completed")

	queue, err := agent.NewQueue(s.config.RedisAddr)
	if err != nil {
		return fmt.Errorf("redis queue init failed: %w", err)
	}
	defer queue.Close()
	queue.StartMockWorker(context.Background())
	log.Println("mock agent worker started")

	integrations, err := integration.NewSuite(integration.Config{
		Mode: s.config.IntegrationMode,
		Registries: map[string]integration.RegistryConfig{
			"local": {
				URL:      s.config.LocalHarborURL,
				Username: s.config.LocalHarborUsername,
				Password: s.config.LocalHarborPassword,
			},
			"remote": {
				URL:      s.config.RemoteHarborURL,
				Username: s.config.RemoteHarborUsername,
				Password: s.config.RemoteHarborPassword,
			},
		},
		Clusters: map[string]integration.ClusterConfig{
			"local": {
				Kubeconfig: s.config.LocalKubeconfig,
			},
			"remote": {
				Kubeconfig: s.config.RemoteKubeconfig,
			},
		},
		HTTPTimeoutMS: s.config.IntegrationHTTPTimeoutMS,
	})
	if err != nil {
		return fmt.Errorf("integration init failed: %w", err)
	}

	mockRepo, err := repository.NewMockRepository()
	if err != nil {
		return fmt.Errorf("load mock repository failed: %w", err)
	}
	repo := repository.NewDatabaseStore(database, mockRepo)
	protocolStore := repository.NewDatabaseProtocolStore(database)
	router := api.NewRouter(repo, queue, protocolStore, integrations)
	return router.Run(fmt.Sprintf(":%s", s.config.Port))
}
