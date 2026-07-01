package runtime

import (
	"context"

	"ops-release-platform/agent/internal/reporter"
)

type Executor interface {
	Execute(ctx context.Context, task reporter.Task) error
}

type UnsupportedExecutor struct {
	client *reporter.Client
}

func NewUnsupportedExecutor(client *reporter.Client) *UnsupportedExecutor {
	return &UnsupportedExecutor{client: client}
}

func (e *UnsupportedExecutor) Execute(ctx context.Context, task reporter.Task) error {
	message := "当前 Agent 仅支持环境远程探测，发布/部署真实执行器尚未接入"
	if err := e.client.AppendLog(ctx, task.ID, message); err != nil {
		return err
	}
	return e.client.ReportResult(ctx, task.ID, "FAILED", message)
}
