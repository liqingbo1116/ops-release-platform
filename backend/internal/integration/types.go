package integration

import (
	"context"
	"errors"
	"strings"
	"time"

	"ops-release-platform/backend/internal/domain"
)

var ErrUnsupportedMode = errors.New("unsupported integration mode")
var ErrMissingRealConfig = errors.New("missing real integration config")

type Config struct {
	Mode          string
	Registries    map[string]RegistryConfig
	Clusters      map[string]ClusterConfig
	HTTPTimeoutMS string
}

type RegistryConfig struct {
	URL      string
	Username string
	Password string
}

type ClusterConfig struct {
	Kubeconfig string
}

type Suite struct {
	Jenkins    JenkinsAdapter
	Registry   RegistryAdapter
	Kubernetes KubernetesAdapter
}

func (s Suite) IsMock() bool {
	_, jenkinsMock := s.Jenkins.(MockJenkinsAdapter)
	_, registryMock := s.Registry.(MockRegistryAdapter)
	_, kubernetesMock := s.Kubernetes.(MockKubernetesAdapter)
	return jenkinsMock || registryMock || kubernetesMock
}

type JenkinsAdapter interface {
	TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error)
	GetBuildStatus(ctx context.Context, buildID string) (BuildStatus, error)
}

type RegistryAdapter interface {
	CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error)
	ListImageTags(ctx context.Context, environment domain.Environment, repository string) ([]ImageInfo, error)
	GetImage(ctx context.Context, image string, tag string) (ImageInfo, error)
	SyncImage(ctx context.Context, req SyncImageRequest) (SyncImageResult, error)
}

type KubernetesAdapter interface {
	CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error)
	ListWorkloads(ctx context.Context, environment domain.Environment) ([]Workload, error)
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
	Namespace     string              `json:"namespace"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Replicas      int                 `json:"replicas"`
	ReadyReplicas int                 `json:"readyReplicas"`
	Containers    []WorkloadContainer `json:"containers"`
}

type WorkloadContainer struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Image string `json:"image"`
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
	mode := strings.TrimSpace(cfg.Mode)
	if mode == "" {
		mode = "mock"
	}
	switch mode {
	case "mock":
		return NewMockSuite(), nil
	case "real":
		return NewRealSuite(cfg, parseTimeout(cfg.HTTPTimeoutMS))
	default:
		return Suite{}, ErrUnsupportedMode
	}
}

func parseTimeout(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 10 * time.Second
	}
	duration, err := time.ParseDuration(value)
	if err == nil && duration > 0 {
		return duration
	}
	duration, err = time.ParseDuration(value + "ms")
	if err == nil && duration > 0 {
		return duration
	}
	return 10 * time.Second
}
