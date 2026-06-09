package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/heartbeat"
	"ops-release-platform/agent/internal/reporter"
	"ops-release-platform/agent/internal/runtime"
	"ops-release-platform/agent/internal/task"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load agent config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := reporter.NewClient(cfg.PlatformURL, cfg.AgentID, cfg.EnvironmentID, cfg.Token, cfg.HTTPTimeout)
	executor := runtime.NewMockExecutor(client)
	worker := task.NewWorker(cfg, client, executor)

	server := &http.Server{
		Addr:              ":" + cfg.HealthPort,
		Handler:           healthHandler(cfg),
		ReadHeaderTimeout: 3 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("health server stopped: %v", err)
		}
	}()

	log.Printf("agent started id=%s environment=%s platform=%s", cfg.AgentID, cfg.EnvironmentID, cfg.PlatformURL)
	go heartbeat.Run(ctx, cfg, client)
	worker.Run(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	log.Println("agent stopped")
}

func healthHandler(cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","agentId":"` + cfg.AgentID + `"}`))
	})
	return mux
}
