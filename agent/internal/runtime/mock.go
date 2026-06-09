package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"ops-release-platform/agent/internal/reporter"
)

type Executor interface {
	Execute(ctx context.Context, task reporter.Task) error
}

type MockExecutor struct {
	client *reporter.Client
}

func NewMockExecutor(client *reporter.Client) *MockExecutor {
	return &MockExecutor{client: client}
}

func (e *MockExecutor) Execute(ctx context.Context, task reporter.Task) error {
	steps := []string{"receive-task", "prepare-runtime", "mock-execute", "collect-result"}
	for _, step := range steps {
		if err := e.client.ReportStep(ctx, task.ID, step, "RUNNING"); err != nil {
			return err
		}
		if err := e.client.AppendLog(ctx, task.ID, fmt.Sprintf("mock agent task=%s action=%s step=%s", task.ID, task.Action, step)); err != nil {
			return err
		}
		log.Printf("executed mock step task=%s step=%s", task.ID, step)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return e.client.ReportResult(ctx, task.ID, "SUCCESS", "mock agent execution finished")
}
