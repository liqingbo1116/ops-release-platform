package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/heartbeat"
	"ops-release-platform/agent/internal/reporter"
	"ops-release-platform/agent/internal/runtime"
	"ops-release-platform/agent/internal/task"
)

func main() {
	configFile := flag.String("f", "", "path to agent config file")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("load agent config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := reporter.NewClient(cfg.PlatformURL, cfg.AgentID, cfg.EnvironmentID, cfg.Token, cfg.HTTPTimeout)
	if cfg.Token == "" {
		result, err := client.Register(ctx, cfg.RegisterToken, "v1-remote-probe", cfg.Capabilities)
		if err != nil {
			log.Fatalf("register agent: %v", err)
		}
		client.SetRuntimeIdentity(result.Agent.ID, result.Agent.EnvironmentID, result.AgentToken)
		cfg.AgentID = result.Agent.ID
		cfg.EnvironmentID = result.Agent.EnvironmentID
		cfg.Token = result.AgentToken
		if *configFile == "" {
			log.Printf("agent registered id=%s claimStatus=%s; no config file was provided, AGENT_TOKEN cannot be persisted automatically", result.Agent.ID, result.Agent.ClaimStatus)
		} else if err := config.PersistRuntimeToken(*configFile, result.AgentToken); err != nil {
			log.Fatalf("persist agent token: %v", err)
		} else {
			log.Printf("agent registered id=%s claimStatus=%s; runtime token persisted to config file", result.Agent.ID, result.Agent.ClaimStatus)
		}
	}
	probeExecutor := runtime.NewProbeExecutor(cfg, client)
	fallbackExecutor := runtime.Executor(runtime.NewUnsupportedExecutor(client))
	executor := runtime.NewRouterExecutor(probeExecutor, fallbackExecutor)
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

	log.Printf("agent started id=%s environment=%s platform=%s mode=%s capabilities=%s", cfg.AgentID, cfg.EnvironmentID, cfg.PlatformURL, cfg.Mode, strings.Join(cfg.Capabilities, ","))
	log.Printf("agent runtime config kubernetesConfigured=%t harborConfigured=%t harborURL=%s", cfg.Kubeconfig != "", cfg.HarborURL != "" && cfg.HarborUsername != "" && cfg.HarborPassword != "", cfg.HarborURL)
	go heartbeat.Run(ctx, cfg, client, probeExecutor.RuntimeStatus)
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
