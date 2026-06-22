package heartbeat

import (
	"context"
	"log"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/reporter"
)

const version = "v1-remote-probe"

type RuntimeStatusCollector func(context.Context) reporter.RuntimeStatus

func Run(ctx context.Context, cfg config.Config, client *reporter.Client, collectRuntimeStatus RuntimeStatusCollector) {
	ticker := newTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		runtimeStatus := reporter.RuntimeStatus{}
		if collectRuntimeStatus != nil {
			runtimeStatus = collectRuntimeStatus(ctx)
		}
		result, err := client.Heartbeat(ctx, version, cfg.Capabilities, runtimeStatus)
		if err != nil && ctx.Err() == nil {
			log.Printf("heartbeat failed: %v", err)
		} else if result.Agent.EnvironmentID != "" && result.Agent.EnvironmentID != client.EnvironmentID() {
			client.SetEnvironmentID(result.Agent.EnvironmentID)
			log.Printf("agent claimed by environment=%s", result.Agent.EnvironmentID)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
