package heartbeat

import (
	"context"
	"log"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/reporter"
)

const version = "v1-mock"

func Run(ctx context.Context, cfg config.Config, client *reporter.Client) {
	ticker := newTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		if err := client.Heartbeat(ctx, version, cfg.Capabilities); err != nil && ctx.Err() == nil {
			log.Printf("heartbeat failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
