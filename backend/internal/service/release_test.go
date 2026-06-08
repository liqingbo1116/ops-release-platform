package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
)

func TestCreateReleaseWithLocalHarborImageUsesRegistry(t *testing.T) {
	creator := NewReleaseCreator(integration.NewMockSuite(), newMockAgentReader(), nil)

	result, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "LOCAL_HARBOR_IMAGE",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
		Image: ReleaseImage{
			Repository: "harbor.local/project-x/user-service",
			Tag:        "20260607-a1b2c3",
		},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if result.ExecutionMode != "AGENT_IMAGE_SYNC" {
		t.Fatalf("expected AGENT_IMAGE_SYNC, got %s", result.ExecutionMode)
	}
	if result.BuildID == "" || result.BuildStatus != "SUCCESS" {
		t.Fatalf("expected sync task metadata, got %+v", result)
	}
}

func TestCreateDeployTaskUsesKubernetesProbe(t *testing.T) {
	creator := NewReleaseCreator(integration.NewMockSuite(), newMockAgentReader(), nil)

	result, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if err != nil {
		t.Fatalf("create deploy task: %v", err)
	}
	if result.ExecutionMode != "AGENT" || result.AgentTaskID == "" {
		t.Fatalf("unexpected deploy result: %+v", result)
	}
}

func TestCreateDeployTaskReturnsWorkloadProbeError(t *testing.T) {
	creator := NewReleaseCreator(integration.Suite{
		Kubernetes: failingKubernetesAdapter{err: errors.New("boom")},
	}, newMockAgentReader(), nil)

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if !errors.Is(err, ErrWorkloadProbe) {
		t.Fatalf("expected ErrWorkloadProbe, got %v", err)
	}
}

func TestCreateReleaseReturnsAgentNotFound(t *testing.T) {
	creator := NewReleaseCreator(integration.NewMockSuite(), newMockAgentReader(), nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-missing",
	})
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestCreateReleaseReturnsAgentOffline(t *testing.T) {
	creator := NewReleaseCreator(integration.NewMockSuite(), newMockAgentReader(), nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-project-z-prod",
		AgentID:             "agent-project-z",
	})
	if !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected ErrAgentOffline, got %v", err)
	}
}

func TestCreateDeployTaskReturnsAgentEnvironmentMismatch(t *testing.T) {
	creator := NewReleaseCreator(integration.NewMockSuite(), newMockAgentReader(), nil)

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-y",
	})
	if !errors.Is(err, ErrAgentEnvironment) {
		t.Fatalf("expected ErrAgentEnvironment, got %v", err)
	}
}

type mockAgentReader map[string]string

func newMockAgentReader() mockAgentReader {
	return mockAgentReader{
		"agent-project-x": "env-project-x-prod:ONLINE",
		"agent-project-y": "env-project-y-pre:ONLINE",
		"agent-project-z": "env-project-z-prod:OFFLINE",
	}
}

func (m mockAgentReader) GetAgent(id string) (domain.Agent, bool) {
	raw, ok := m[id]
	if !ok {
		return domain.Agent{}, false
	}
	parts := strings.SplitN(raw, ":", 2)
	return domain.Agent{
		ID:            id,
		EnvironmentID: parts[0],
		Status:        parts[1],
	}, true
}

type failingKubernetesAdapter struct {
	err error
}

func (f failingKubernetesAdapter) CheckConnection(ctx context.Context, environmentID string) (integration.IntegrationCheck, error) {
	return integration.IntegrationCheck{}, f.err
}

func (f failingKubernetesAdapter) ListWorkloads(ctx context.Context, environmentID string) ([]integration.Workload, error) {
	return nil, f.err
}

func (f failingKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req integration.SetImageRequest) error {
	return f.err
}

func (f failingKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (integration.RolloutStatus, error) {
	return integration.RolloutStatus{}, f.err
}
