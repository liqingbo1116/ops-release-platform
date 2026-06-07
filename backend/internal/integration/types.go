package integration

import (
	"context"
	"errors"
)

var ErrUnsupportedMode = errors.New("unsupported integration mode")

type Config struct {
	Mode string
}

type Suite struct {
	Jenkins    JenkinsAdapter
	Registry   RegistryAdapter
	Kubernetes KubernetesAdapter
}

type JenkinsAdapter interface {
	TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error)
	GetBuildStatus(ctx context.Context, buildID string) (BuildStatus, error)
}

type RegistryAdapter interface {
	CheckConnection(ctx context.Context, environmentID string) (IntegrationCheck, error)
	GetImage(ctx context.Context, image string, tag string) (ImageInfo, error)
	SyncImage(ctx context.Context, req SyncImageRequest) (SyncImageResult, error)
}

type KubernetesAdapter interface {
	CheckConnection(ctx context.Context, environmentID string) (IntegrationCheck, error)
	ListWorkloads(ctx context.Context, environmentID string) ([]Workload, error)
	SetImage(ctx context.Context, environmentID string, req SetImageRequest) error
	GetRolloutStatus(ctx context.Context, environmentID string, workload string) (RolloutStatus, error)
}

type BuildRequest struct {
	JobName    string            `json:"jobName"`
	Branch     string            `json:"branch"`
	Parameters map[string]string `json:"parameters"`
}

type BuildResult struct {
	BuildID string `json:"buildId"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

type BuildStatus struct {
	BuildID    string `json:"buildId"`
	Status     string `json:"status"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
	LogURL     string `json:"logUrl"`
}

type ImageInfo struct {
	Image     string `json:"image"`
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
	Exists    bool   `json:"exists"`
	UpdatedAt string `json:"updatedAt"`
}

type SyncImageRequest struct {
	SourceImage    string `json:"sourceImage"`
	SourceTag      string `json:"sourceTag"`
	TargetRegistry string `json:"targetRegistry"`
	TargetProject  string `json:"targetProject"`
}

type SyncImageResult struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
	Digest string `json:"digest"`
}

type Workload struct {
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Image         string `json:"image"`
	Tag           string `json:"tag"`
	Replicas      int    `json:"replicas"`
	ReadyReplicas int    `json:"readyReplicas"`
}

type SetImageRequest struct {
	Namespace string `json:"namespace"`
	Workload  string `json:"workload"`
	Container string `json:"container"`
	Image     string `json:"image"`
	Tag       string `json:"tag"`
}

type RolloutStatus struct {
	Namespace     string `json:"namespace"`
	Workload      string `json:"workload"`
	Status        string `json:"status"`
	Replicas      int    `json:"replicas"`
	ReadyReplicas int    `json:"readyReplicas"`
	Message       string `json:"message"`
}

type IntegrationCheck struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CheckedAt string `json:"checkedAt"`
}

func NewSuite(cfg Config) (Suite, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = "mock"
	}
	if mode != "mock" {
		return Suite{}, ErrUnsupportedMode
	}
	return NewMockSuite(), nil
}
