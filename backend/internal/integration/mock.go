package integration

import (
	"context"
	"fmt"
	"time"

	"ops-release-platform/backend/internal/domain"
)

func NewMockSuite() Suite {
	return Suite{
		Jenkins:    MockJenkinsAdapter{},
		Registry:   MockRegistryAdapter{},
		Kubernetes: MockKubernetesAdapter{},
	}
}

type MockJenkinsAdapter struct{}

func (MockJenkinsAdapter) TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error) {
	if err := ctx.Err(); err != nil {
		return BuildResult{}, err
	}
	buildID := "BUILD-MOCK-20260607"
	return BuildResult{
		BuildID: buildID,
		Status:  "QUEUED",
		URL:     "https://jenkins.mock/job/" + req.JobName + "/" + buildID,
	}, nil
}

func (MockJenkinsAdapter) GetBuildStatus(ctx context.Context, buildID string) (BuildStatus, error) {
	if err := ctx.Err(); err != nil {
		return BuildStatus{}, err
	}
	now := time.Now()
	return BuildStatus{
		BuildID:    buildID,
		Status:     "SUCCESS",
		StartedAt:  now.Add(-2 * time.Minute).Format(time.RFC3339),
		FinishedAt: now.Format(time.RFC3339),
		LogURL:     "https://jenkins.mock/builds/" + buildID + "/console",
	}, nil
}

type MockRegistryAdapter struct{}

func (MockRegistryAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return IntegrationCheck{}, err
	}
	return IntegrationCheck{
		Component: "harbor",
		Status:    "HEALTHY",
		Message:   "mock registry connection is available for " + environment.ID,
		CheckedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (MockRegistryAdapter) GetImage(ctx context.Context, image string, tag string) (ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return ImageInfo{}, err
	}
	return ImageInfo{
		Image:     image,
		Tag:       tag,
		Digest:    "sha256:mock-" + tag,
		Exists:    true,
		UpdatedAt: time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
	}, nil
}

func (MockRegistryAdapter) ListImageTags(ctx context.Context, environment domain.Environment, repository string) ([]ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []ImageInfo{
		{
			Image:     repository,
			Tag:       "20260607-a1b2c3",
			Digest:    "sha256:mock-20260607-a1b2c3",
			Exists:    true,
			UpdatedAt: time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
		},
	}, nil
}

func (MockRegistryAdapter) SyncImage(ctx context.Context, req SyncImageRequest) (SyncImageResult, error) {
	if err := ctx.Err(); err != nil {
		return SyncImageResult{}, err
	}
	return SyncImageResult{
		TaskID: "IMG-SYNC-MOCK-20260607",
		Status: "SUCCESS",
		Digest: fmt.Sprintf("sha256:mock-%s-%s", req.TargetProject, req.SourceTag),
	}, nil
}

type MockKubernetesAdapter struct{}

func (MockKubernetesAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return IntegrationCheck{}, err
	}
	return IntegrationCheck{
		Component: "kubernetes",
		Status:    "HEALTHY",
		Message:   "mock kubernetes connection is available for " + environment.ID,
		CheckedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (MockKubernetesAdapter) ListWorkloads(ctx context.Context, environment domain.Environment) ([]Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []Workload{
		{
			Namespace:     "project-x",
			Name:          "user-service",
			Type:          "Deployment",
			Replicas:      4,
			ReadyReplicas: 4,
			Containers: []WorkloadContainer{
				{Name: "user-service", Type: "APP", Image: "harbor.local/project-x/user-service:20260607-a1b2c3"},
			},
		},
		{
			Namespace:     "project-x",
			Name:          "order-service",
			Type:          "Deployment",
			Replicas:      3,
			ReadyReplicas: 3,
			Containers: []WorkloadContainer{
				{Name: "order-service", Type: "APP", Image: "harbor.local/project-x/order-service:20260606-111aaa"},
			},
		},
	}, nil
}

func (MockKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req SetImageRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (MockKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (RolloutStatus, error) {
	if err := ctx.Err(); err != nil {
		return RolloutStatus{}, err
	}
	return RolloutStatus{
		Namespace:     "project-x",
		Workload:      workload,
		Status:        "SUCCESS",
		Replicas:      4,
		ReadyReplicas: 4,
		Message:       "mock rollout completed",
	}, nil
}
