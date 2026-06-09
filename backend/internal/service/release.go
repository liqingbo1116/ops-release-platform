package service

import (
	"context"
	"errors"
	"time"

	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
)

var (
	ErrInvalidReleaseType         = errors.New("invalid release type")
	ErrInvalidDeployType          = errors.New("invalid deploy type")
	ErrAgentNotFound              = errors.New("agent not found")
	ErrAgentOffline               = errors.New("agent offline")
	ErrAgentEnvironment           = errors.New("agent environment mismatch")
	ErrEnvironmentPermission      = errors.New("environment permission denied")
	ErrBaselineNotFound           = errors.New("baseline not found")
	ErrReleaseBaselineUnsupported = errors.New("release baseline unsupported")
	ErrDeployBaselineRequired     = errors.New("deploy baseline required")
	ErrInvalidServiceSelection    = errors.New("invalid service selection")
	ErrJenkinsTrigger             = errors.New("jenkins trigger failed")
	ErrRegistryImageCheck         = errors.New("registry image check failed")
	ErrRegistryImageSync          = errors.New("registry image sync failed")
	ErrImageNotFound              = errors.New("release image not found")
	ErrWorkloadProbe              = errors.New("kubernetes workload probe failed")
)

type EnqueueFunc func(ctx context.Context, id string, taskType string, action string)

type AgentReader interface {
	GetAgent(id string) (domain.Agent, bool)
}

type DiffReader interface {
	GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool)
}

type PermissionReader interface {
	HasEnvironmentAction(environmentID string, action string) bool
}

type ReleaseCreator struct {
	integrations integration.Suite
	agents       AgentReader
	diffReader   DiffReader
	permissions  PermissionReader
	enqueue      EnqueueFunc
	now          func() time.Time
}

func NewReleaseCreator(integrations integration.Suite, agents AgentReader, diffReader DiffReader, enqueue EnqueueFunc, permissions ...PermissionReader) *ReleaseCreator {
	creator := &ReleaseCreator{
		integrations: integrations,
		agents:       agents,
		diffReader:   diffReader,
		enqueue:      enqueue,
		now:          time.Now,
	}
	if len(permissions) > 0 {
		creator.permissions = permissions[0]
	}
	return creator
}

type CreateReleaseRequest struct {
	Type                string          `json:"type"`
	ReleaseSource       string          `json:"releaseSource"`
	SourceBaselineID    string          `json:"sourceBaselineId"`
	TargetEnvironmentID string          `json:"targetEnvironmentId"`
	AgentID             string          `json:"agentId"`
	ServiceIDs          []string        `json:"serviceIds"`
	Image               ReleaseImage    `json:"image"`
	Jenkins             ReleaseJenkins  `json:"jenkins"`
	Options             map[string]bool `json:"options"`
}

type ReleaseImage struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Digest     string `json:"digest"`
}

type ReleaseJenkins struct {
	JobName    string            `json:"jobName"`
	Branch     string            `json:"branch"`
	Parameters map[string]string `json:"parameters"`
}

type CreateReleaseResult struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	ExecutionMode string `json:"executionMode"`
	AgentTaskID   string `json:"agentTaskId"`
	ReleaseSource string `json:"releaseSource,omitempty"`
	BuildID       string `json:"buildId,omitempty"`
	BuildStatus   string `json:"buildStatus,omitempty"`
	BuildURL      string `json:"buildUrl,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

func (c *ReleaseCreator) CreateRelease(ctx context.Context, request CreateReleaseRequest) (CreateReleaseResult, error) {
	if request.Type != "SERVICE_RELEASE" {
		return CreateReleaseResult{}, ErrInvalidReleaseType
	}
	if err := c.validateAgent(request.AgentID, request.TargetEnvironmentID); err != nil {
		return CreateReleaseResult{}, err
	}
	if err := c.validateEnvironmentPermission(request.TargetEnvironmentID, "release"); err != nil {
		return CreateReleaseResult{}, err
	}
	if request.SourceBaselineID != "" {
		return CreateReleaseResult{}, ErrReleaseBaselineUnsupported
	}

	id := "REL-20260607-MOCK"
	if request.ReleaseSource == "LOCAL_HARBOR_IMAGE" {
		if c.integrations.Registry != nil {
			image, err := c.integrations.Registry.GetImage(ctx, request.Image.Repository, request.Image.Tag)
			if err != nil {
				return CreateReleaseResult{}, ErrRegistryImageCheck
			}
			if !image.Exists {
				return CreateReleaseResult{}, ErrImageNotFound
			}
			syncResult, err := c.integrations.Registry.SyncImage(ctx, integration.SyncImageRequest{
				SourceImage:    request.Image.Repository,
				SourceTag:      request.Image.Tag,
				TargetRegistry: request.TargetEnvironmentID,
				TargetProject:  request.AgentID,
			})
			if err != nil {
				return CreateReleaseResult{}, ErrRegistryImageSync
			}
			c.enqueueIfNeeded(ctx, id, "release", "harbor-image-sync")
			return CreateReleaseResult{
				ID:            id,
				Status:        "RUNNING",
				ExecutionMode: "AGENT_IMAGE_SYNC",
				AgentTaskID:   id,
				ReleaseSource: request.ReleaseSource,
				BuildID:       syncResult.TaskID,
				BuildStatus:   syncResult.Status,
				CreatedAt:     c.now().Format(time.RFC3339),
			}, nil
		}
		c.enqueueIfNeeded(ctx, id, "release", "harbor-image-sync")
		return CreateReleaseResult{
			ID:            id,
			Status:        "PENDING_IMAGE_SYNC",
			ExecutionMode: "AGENT_IMAGE_SYNC",
			AgentTaskID:   id,
			ReleaseSource: request.ReleaseSource,
			CreatedAt:     c.now().Format(time.RFC3339),
		}, nil
	}

	if c.integrations.Jenkins != nil {
		jobName := request.Jenkins.JobName
		if jobName == "" {
			jobName = "mock-service-release"
		}
		build, err := c.integrations.Jenkins.TriggerBuild(ctx, integration.BuildRequest{
			JobName:    jobName,
			Branch:     request.Jenkins.Branch,
			Parameters: request.Jenkins.Parameters,
		})
		if err != nil {
			return CreateReleaseResult{}, ErrJenkinsTrigger
		}
		c.enqueueIfNeeded(ctx, id, "release", "project-agent-sync")
		return CreateReleaseResult{
			ID:            id,
			Status:        "JENKINS_QUEUED",
			ExecutionMode: "JENKINS_AGENT",
			AgentTaskID:   id,
			ReleaseSource: request.ReleaseSource,
			BuildID:       build.BuildID,
			BuildStatus:   build.Status,
			BuildURL:      build.URL,
			CreatedAt:     c.now().Format(time.RFC3339),
		}, nil
	}

	c.enqueueIfNeeded(ctx, id, "release", "create")
	return CreateReleaseResult{
		ID:            id,
		Status:        "PENDING_CONFIRM",
		ExecutionMode: "AGENT",
		AgentTaskID:   id,
		CreatedAt:     c.now().Format(time.RFC3339),
	}, nil
}

type CreateDeployTaskRequest struct {
	Type                string          `json:"type"`
	SourceBaselineID    string          `json:"sourceBaselineId"`
	TargetEnvironmentID string          `json:"targetEnvironmentId"`
	AgentID             string          `json:"agentId"`
	ServiceIDs          []string        `json:"serviceIds"`
	Options             map[string]bool `json:"options"`
}

type CreateDeployTaskResult struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	ExecutionMode string `json:"executionMode"`
	AgentTaskID   string `json:"agentTaskId"`
	CreatedAt     string `json:"createdAt"`
}

func (c *ReleaseCreator) CreateDeployTask(ctx context.Context, request CreateDeployTaskRequest) (CreateDeployTaskResult, error) {
	if request.Type != "" && request.Type != "SERVICE_DEPLOYMENT" {
		return CreateDeployTaskResult{}, ErrInvalidDeployType
	}
	if err := c.validateAgent(request.AgentID, request.TargetEnvironmentID); err != nil {
		return CreateDeployTaskResult{}, err
	}
	if err := c.validateEnvironmentPermission(request.TargetEnvironmentID, "deploy"); err != nil {
		return CreateDeployTaskResult{}, err
	}
	if request.SourceBaselineID == "" {
		return CreateDeployTaskResult{}, ErrDeployBaselineRequired
	}
	allowedServiceIDs, err := c.resolveServiceSelection(request.SourceBaselineID, request.TargetEnvironmentID, request.ServiceIDs, "MISSING_IN_TARGET")
	if err != nil {
		return CreateDeployTaskResult{}, err
	}
	request.ServiceIDs = allowedServiceIDs

	if c.integrations.Kubernetes != nil {
		if _, err := c.integrations.Kubernetes.ListWorkloads(ctx, request.TargetEnvironmentID); err != nil {
			return CreateDeployTaskResult{}, ErrWorkloadProbe
		}
	}

	id := "DEP-20260607-MOCK"
	c.enqueueIfNeeded(ctx, id, "deploy", "create")
	return CreateDeployTaskResult{
		ID:            id,
		Status:        "PENDING",
		ExecutionMode: "AGENT",
		AgentTaskID:   id,
		CreatedAt:     c.now().Format(time.RFC3339),
	}, nil
}

func (c *ReleaseCreator) enqueueIfNeeded(ctx context.Context, id string, taskType string, action string) {
	if c.enqueue == nil {
		return
	}
	c.enqueue(ctx, id, taskType, action)
}

func (c *ReleaseCreator) validateAgent(agentID string, environmentID string) error {
	if c.agents == nil {
		return nil
	}
	agent, ok := c.agents.GetAgent(agentID)
	if !ok {
		return ErrAgentNotFound
	}
	if agent.EnvironmentID != environmentID {
		return ErrAgentEnvironment
	}
	if agent.Status != "ONLINE" {
		return ErrAgentOffline
	}
	return nil
}

func (c *ReleaseCreator) validateEnvironmentPermission(environmentID string, action string) error {
	if c.permissions == nil {
		return nil
	}
	if c.permissions.HasEnvironmentAction(environmentID, action) {
		return nil
	}
	return ErrEnvironmentPermission
}

func (c *ReleaseCreator) resolveServiceSelection(sourceBaselineID string, targetEnvironmentID string, requestedServiceIDs []string, expectedStatus string) ([]string, error) {
	if c.diffReader == nil {
		return nil, ErrBaselineNotFound
	}
	diff, ok := c.diffReader.GetDiffResult(sourceBaselineID, targetEnvironmentID)
	if !ok {
		return nil, ErrBaselineNotFound
	}
	allowed := make(map[string]struct{})
	defaultSelection := make([]string, 0)
	for _, item := range diff.Items {
		if item.DiffStatus != expectedStatus {
			continue
		}
		allowed[item.ServiceID] = struct{}{}
		defaultSelection = append(defaultSelection, item.ServiceID)
	}
	if len(requestedServiceIDs) == 0 {
		return defaultSelection, nil
	}
	selected := make([]string, 0, len(requestedServiceIDs))
	seen := make(map[string]struct{}, len(requestedServiceIDs))
	for _, serviceID := range requestedServiceIDs {
		if _, ok := allowed[serviceID]; !ok {
			return nil, ErrInvalidServiceSelection
		}
		if _, duplicated := seen[serviceID]; duplicated {
			continue
		}
		seen[serviceID] = struct{}{}
		selected = append(selected, serviceID)
	}
	return selected, nil
}
