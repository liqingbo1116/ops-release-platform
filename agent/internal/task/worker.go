package task

import (
	"context"
	"log"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/reporter"
	"ops-release-platform/agent/internal/runtime"
)

const leaseSeconds = 300

type Worker struct {
	cfg      config.Config
	client   *reporter.Client
	executor runtime.Executor
}

func NewWorker(cfg config.Config, client *reporter.Client, executor runtime.Executor) *Worker {
	return &Worker{cfg: cfg, client: client, executor: executor}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := newTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		w.pollOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) pollOnce(ctx context.Context) {
	lease, err := w.client.Lease(ctx, leaseSeconds)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("lease task failed: %v", err)
		}
		return
	}
	if !lease.Leased || lease.Task == nil {
		return
	}
	log.Printf("leased task id=%s type=%s action=%s", lease.Task.ID, lease.Task.Type, lease.Task.Action)
	if err := w.executor.Execute(ctx, *lease.Task); err != nil {
		log.Printf("execute task failed id=%s: %v", lease.Task.ID, err)
		_ = w.client.ReportResult(context.Background(), lease.Task.ID, "FAILED", err.Error())
	}
}
