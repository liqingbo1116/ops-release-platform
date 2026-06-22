package runtime

import (
	"context"

	"ops-release-platform/agent/internal/reporter"
)

type RouterExecutor struct {
	probe    Executor
	fallback Executor
}

func NewRouterExecutor(probe Executor, fallback Executor) *RouterExecutor {
	return &RouterExecutor{probe: probe, fallback: fallback}
}

func (e *RouterExecutor) Execute(ctx context.Context, task reporter.Task) error {
	if task.Type == "probe" || task.Action == "remote_resource_probe" {
		return e.probe.Execute(ctx, task)
	}
	return e.fallback.Execute(ctx, task)
}
