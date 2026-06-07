package service

import (
	"context"
	"errors"
	"time"

	"ops-release-platform/backend/internal/integration"
)

var (
	ErrInvalidReleaseType = errors.New("invalid release type")
	ErrInvalidDeployType  = errors.New("invalid deploy type")
	ErrJenkinsTrigger     = errors.New("jenkins trigger failed")
)

type EnqueueFunc func(ctx context.Context, id string, taskType string, action string)

type ReleaseCreator struct {
	integrations integration.Suite
	enqueue      EnqueueFunc
	now          func() time.Time
}

func NewReleaseCreator(integrations integration.Suite, enqueue EnqueueFunc) *ReleaseCreator {
	return &ReleaseCreator{
		integrations: integrations,
		enqueue:      enqueue,
		now:          time.Now,
	}
}

type CreateReleaseRequest struct {
	Type                string            `json:"type"`
	ReleaseSource       string            `json:"releaseSource"`
	TargetEnvironmentID string            `json:"targetEnvironmentId"`
	AgentID             string            `json:"agentId"`
	ServiceIDs          []string          `json:"serviceIds"`
	Image               ReleaseImage      `json:"image"`
	Jenkins             ReleaseJenkins    `json:"jenkins"`
	Options             map[string]bool   `json:"options"`
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

	id := "REL-20260607-MOCK"
	if request.ReleaseSource == "LOCAL_HARBOR_IMAGE" {
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
