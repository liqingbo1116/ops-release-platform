package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	ErrInvalidReleaseSource       = errors.New("invalid release source")
	ErrEnvironmentNotFound        = errors.New("environment not found")
	ErrReleaseOrderCreate         = errors.New("release order create failed")
	ErrJenkinsTrigger             = errors.New("jenkins trigger failed")
	ErrRegistryImageCheck         = errors.New("registry image check failed")
	ErrRegistryImageSync          = errors.New("registry image sync failed")
	ErrImageNotFound              = errors.New("release image not found")
	ErrWorkloadProbe              = errors.New("kubernetes workload probe failed")
)

type EnqueueFunc func(ctx context.Context, id string, taskType string, action string, agentID string, environmentID string)

type AgentReader interface {
	GetAgent(id string) (domain.Agent, bool)
}

type DiffReader interface {
	GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool)
}

type PermissionReader interface {
	HasEnvironmentAction(environmentID string, action string) bool
}

type EnvironmentReader interface {
	GetEnvironment(id string) (domain.Environment, bool)
}

type ReleaseOrderWriter interface {
	CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error)
}

type ReleaseCreator struct {
	integrations integration.Suite
	agents       AgentReader
	diffReader   DiffReader
	environments EnvironmentReader
	orders       ReleaseOrderWriter
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
	if environmentReader, ok := agents.(EnvironmentReader); ok {
		creator.environments = environmentReader
	}
	if orderWriter, ok := agents.(ReleaseOrderWriter); ok {
		creator.orders = orderWriter
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

	releaseSource := strings.TrimSpace(request.ReleaseSource)
	if releaseSource == "" {
		releaseSource = "LOCAL_HARBOR_IMAGE"
	}

	id := c.nextID("REL")
	createdAt := c.now().Format(time.RFC3339)
	if releaseSource == "LOCAL_HARBOR_IMAGE" {
		if strings.TrimSpace(request.Image.Repository) == "" || strings.TrimSpace(request.Image.Tag) == "" {
			return CreateReleaseResult{}, ErrRegistryImageCheck
		}
		environment, err := c.targetEnvironment(request.TargetEnvironmentID)
		if err != nil {
			return CreateReleaseResult{}, err
		}
		image, err := c.findImageTag(ctx, environment, request.Image.Repository, request.Image.Tag)
		if err != nil {
			return CreateReleaseResult{}, err
		}
		order, err := c.createReleaseOrder(domain.CreateReleaseOrderInput{
			ID:                   id,
			Type:                 request.Type,
			ReleaseSource:        releaseSource,
			ExecutionMode:        "AGENT_IMAGE_SYNC",
			ImageRepository:      request.Image.Repository,
			ImageTag:             request.Image.Tag,
			ImageDigest:          firstNonEmpty(request.Image.Digest, image.Digest),
			TargetEnvironmentID:  request.TargetEnvironmentID,
			AgentID:              request.AgentID,
			Status:               "PENDING_IMAGE_SYNC",
			Progress:             0,
			SelectedServiceCount: len(request.ServiceIDs),
		})
		if err != nil {
			return CreateReleaseResult{}, err
		}
		c.enqueueIfNeeded(ctx, id, "release", "harbor-image-sync", request.AgentID, request.TargetEnvironmentID)
		return CreateReleaseResult{
			ID:            order.ID,
			Status:        order.Status,
			ExecutionMode: order.ExecutionMode,
			AgentTaskID:   order.ID,
			ReleaseSource: order.ReleaseSource,
			CreatedAt:     createdAt,
		}, nil
	}

	if releaseSource == "JENKINS_JOB" {
		jobName := strings.TrimSpace(request.Jenkins.JobName)
		if jobName == "" {
			return CreateReleaseResult{}, ErrJenkinsTrigger
		}
		if c.integrations.Jenkins == nil {
			return CreateReleaseResult{}, ErrJenkinsTrigger
		}
		build, err := c.integrations.Jenkins.TriggerBuild(ctx, integration.BuildRequest{
			JobName:    jobName,
			Branch:     request.Jenkins.Branch,
			Parameters: request.Jenkins.Parameters,
		})
		if err != nil {
			return CreateReleaseResult{}, ErrJenkinsTrigger
		}
		order, err := c.createReleaseOrder(domain.CreateReleaseOrderInput{
			ID:                   id,
			Type:                 request.Type,
			ReleaseSource:        releaseSource,
			ExecutionMode:        "JENKINS_AGENT",
			BuildID:              build.BuildID,
			BuildStatus:          build.Status,
			BuildURL:             build.URL,
			TargetEnvironmentID:  request.TargetEnvironmentID,
			AgentID:              request.AgentID,
			Status:               "JENKINS_QUEUED",
			Progress:             0,
			SelectedServiceCount: len(request.ServiceIDs),
		})
		if err != nil {
			return CreateReleaseResult{}, err
		}
		c.enqueueIfNeeded(ctx, id, "release", "project-agent-sync", request.AgentID, request.TargetEnvironmentID)
		return CreateReleaseResult{
			ID:            order.ID,
			Status:        order.Status,
			ExecutionMode: order.ExecutionMode,
			AgentTaskID:   order.ID,
			ReleaseSource: order.ReleaseSource,
			BuildID:       build.BuildID,
			BuildStatus:   build.Status,
			BuildURL:      build.URL,
			CreatedAt:     createdAt,
		}, nil
	}

	return CreateReleaseResult{}, ErrInvalidReleaseSource
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
		environment, err := c.targetEnvironment(request.TargetEnvironmentID)
		if err != nil {
			return CreateDeployTaskResult{}, err
		}
		if _, err := c.integrations.Kubernetes.ListWorkloads(ctx, environment); err != nil {
			return CreateDeployTaskResult{}, ErrWorkloadProbe
		}
	}

	id := "DEP-20260607-MOCK"
	c.enqueueIfNeeded(ctx, id, "deploy", "create", request.AgentID, request.TargetEnvironmentID)
	return CreateDeployTaskResult{
		ID:            id,
		Status:        "PENDING",
		ExecutionMode: "AGENT",
		AgentTaskID:   id,
		CreatedAt:     c.now().Format(time.RFC3339),
	}, nil
}

func (c *ReleaseCreator) enqueueIfNeeded(ctx context.Context, id string, taskType string, action string, agentID string, environmentID string) {
	if c.enqueue == nil {
		return
	}
	c.enqueue(ctx, id, taskType, action, agentID, environmentID)
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

func (c *ReleaseCreator) targetEnvironment(id string) (domain.Environment, error) {
	if c.environments == nil {
		return domain.Environment{ID: id}, nil
	}
	environment, ok := c.environments.GetEnvironment(id)
	if !ok {
		return domain.Environment{}, ErrEnvironmentNotFound
	}
	return environment, nil
}

func (c *ReleaseCreator) findImageTag(ctx context.Context, environment domain.Environment, repository string, tag string) (integration.ImageInfo, error) {
	if c.integrations.Registry == nil {
		return integration.ImageInfo{}, ErrRegistryImageCheck
	}
	tags, err := c.integrations.Registry.ListImageTags(ctx, environment, repository)
	if err != nil {
		return integration.ImageInfo{}, ErrRegistryImageCheck
	}
	for _, item := range tags {
		if item.Tag == tag {
			item.Exists = true
			return item, nil
		}
	}
	return integration.ImageInfo{}, ErrImageNotFound
}

func (c *ReleaseCreator) createReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error) {
	if c.orders == nil {
		return domain.ReleaseOrder{
			ID:                    input.ID,
			Type:                  input.Type,
			SourceBaselineID:      input.SourceBaselineID,
			ReleaseSource:         input.ReleaseSource,
			ExecutionMode:         input.ExecutionMode,
			BuildID:               input.BuildID,
			BuildStatus:           input.BuildStatus,
			BuildURL:              input.BuildURL,
			ImageRepository:       input.ImageRepository,
			ImageTag:              input.ImageTag,
			ImageDigest:           input.ImageDigest,
			TargetEnvironmentName: input.TargetEnvironmentID,
			Status:                input.Status,
			Progress:              input.Progress,
			AgentName:             input.AgentID,
		}, nil
	}
	order, err := c.orders.CreateReleaseOrder(input)
	if err != nil {
		return domain.ReleaseOrder{}, ErrReleaseOrderCreate
	}
	return order, nil
}

func (c *ReleaseCreator) nextID(prefix string) string {
	now := c.now()
	return fmt.Sprintf("%s-%s-%06d", prefix, now.Format("20060102-150405"), now.Nanosecond()/1000)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
