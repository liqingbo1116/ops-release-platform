package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
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

type JenkinsTriggerError struct {
	Reason string
}

func (e JenkinsTriggerError) Error() string {
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		return ErrJenkinsTrigger.Error()
	}
	return ErrJenkinsTrigger.Error() + ": " + reason
}

func NewJenkinsTriggerError(reason string) error {
	return JenkinsTriggerError{Reason: reason}
}

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

type JenkinsReader interface {
	GetJenkinsInstance(id string) (domain.JenkinsInstance, bool)
}

type ReleaseOrderWriter interface {
	CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error)
}

type ReleaseOrderBuildUpdater interface {
	UpdateReleaseBuildStatus(id string, buildID string, buildStatus string, buildURL string, status string, progress int) (domain.ReleaseOrder, bool, error)
}

type ManagedServiceReader interface {
	ListManagedServices(productID string) []domain.ManagedService
}

type ReleaseCreator struct {
	integrations integration.Suite
	agents       AgentReader
	diffReader   DiffReader
	environments EnvironmentReader
	jenkins      JenkinsReader
	orders       ReleaseOrderWriter
	managed      ManagedServiceReader
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
	if jenkinsReader, ok := agents.(JenkinsReader); ok {
		creator.jenkins = jenkinsReader
	}
	if orderWriter, ok := agents.(ReleaseOrderWriter); ok {
		creator.orders = orderWriter
	}
	if managedReader, ok := agents.(ManagedServiceReader); ok {
		creator.managed = managedReader
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
	JobURL     string            `json:"jobUrl"`
	Branch     string            `json:"branch"`
	Parameters map[string]string `json:"parameters"`
}

type CreateReleaseResult struct {
	ID            string   `json:"id"`
	Status        string   `json:"status"`
	ExecutionMode string   `json:"executionMode"`
	AgentTaskID   string   `json:"agentTaskId"`
	ReleaseSource string   `json:"releaseSource,omitempty"`
	BuildID       string   `json:"buildId,omitempty"`
	BuildStatus   string   `json:"buildStatus,omitempty"`
	BuildURL      string   `json:"buildUrl,omitempty"`
	ServiceIDs    []string `json:"serviceIds,omitempty"`
	ServiceNames  []string `json:"serviceNames,omitempty"`
	CreatedAt     string   `json:"createdAt"`
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
		serviceIDs, serviceNames := c.resolveReleaseServices(request.TargetEnvironmentID, request.ServiceIDs)
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
			SelectedServiceCount: len(serviceIDs),
			ServiceIDs:           serviceIDs,
			ServiceNames:         serviceNames,
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
			ServiceIDs:    order.ServiceIDs,
			ServiceNames:  order.ServiceNames,
			CreatedAt:     createdAt,
		}, nil
	}

	if releaseSource == "JENKINS_JOB" {
		serviceIDs, serviceNames := c.resolveReleaseServices(request.TargetEnvironmentID, request.ServiceIDs)
		jobName := strings.TrimSpace(request.Jenkins.JobName)
		jobURL := strings.TrimSpace(request.Jenkins.JobURL)
		if jobName == "" {
			return CreateReleaseResult{}, NewJenkinsTriggerError("服务未绑定 Jenkins Pipeline")
		}
		environment, err := c.targetEnvironment(request.TargetEnvironmentID)
		if err != nil {
			return CreateReleaseResult{}, err
		}
		if c.integrations.Jenkins == nil {
			return CreateReleaseResult{}, NewJenkinsTriggerError("Jenkins 集成未初始化")
		}
		jenkins, pipeline, err := c.resolveJenkinsPipeline(environment, jobName, jobURL)
		if err != nil {
			return CreateReleaseResult{}, NewJenkinsTriggerError(err.Error())
		}
		parameters, parameterized, err := c.resolveJenkinsBuildParameters(ctx, jenkins, pipeline, request.Jenkins.Parameters, request.Jenkins.Branch)
		if err != nil {
			return CreateReleaseResult{}, NewJenkinsTriggerError(err.Error())
		}
		executionMode := "JENKINS_ONLY"
		agentTaskID := ""
		if environment.NetworkMode == "AGENT" {
			executionMode = "JENKINS_AGENT"
			agentTaskID = id
		}
		order, err := c.createReleaseOrder(domain.CreateReleaseOrderInput{
			ID:                   id,
			Type:                 request.Type,
			ReleaseSource:        releaseSource,
			ExecutionMode:        executionMode,
			BuildStatus:          "TRIGGERING",
			JenkinsID:            jenkins.ID,
			JenkinsJobName:       jobName,
			JenkinsJobURL:        pipeline.URL,
			TargetEnvironmentID:  request.TargetEnvironmentID,
			AgentID:              request.AgentID,
			Status:               "JENKINS_TRIGGERING",
			Progress:             0,
			SelectedServiceCount: len(serviceIDs),
			ServiceIDs:           serviceIDs,
			ServiceNames:         serviceNames,
		})
		if err != nil {
			return CreateReleaseResult{}, err
		}
		c.triggerJenkinsReleaseAsync(id, environment, jenkins, pipeline, parameterized, parameters, request.AgentID, request.TargetEnvironmentID)
		return CreateReleaseResult{
			ID:            order.ID,
			Status:        order.Status,
			ExecutionMode: order.ExecutionMode,
			AgentTaskID:   agentTaskID,
			ReleaseSource: order.ReleaseSource,
			BuildStatus:   "TRIGGERING",
			ServiceIDs:    order.ServiceIDs,
			ServiceNames:  order.ServiceNames,
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

	id := c.nextID("DEP")
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
	if strings.TrimSpace(agentID) == "" {
		if c.environments != nil {
			environment, ok := c.environments.GetEnvironment(environmentID)
			if ok && environment.NetworkMode == "DIRECT" {
				return nil
			}
		}
		return ErrAgentNotFound
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

func (c *ReleaseCreator) resolveJenkinsPipeline(environment domain.Environment, jobName string, jobURL string) (domain.JenkinsInstance, domain.JenkinsPipeline, error) {
	if c.jenkins == nil {
		return domain.JenkinsInstance{}, domain.JenkinsPipeline{}, fmt.Errorf("Jenkins 数据源未初始化")
	}
	jenkinsID := strings.TrimSpace(environment.JenkinsID)
	if jenkinsID == "" {
		for _, binding := range environment.Bindings {
			if binding.ResourceType == "JENKINS" && (binding.BindingRole == "" || binding.BindingRole == "BUILD_SOURCE") {
				jenkinsID = strings.TrimSpace(binding.ResourceID)
				break
			}
		}
	}
	if jenkinsID == "" {
		return domain.JenkinsInstance{}, domain.JenkinsPipeline{}, fmt.Errorf("当前产品未绑定 Jenkins")
	}
	jenkins, ok := c.jenkins.GetJenkinsInstance(jenkinsID)
	if !ok || strings.TrimSpace(jenkins.URL) == "" {
		return domain.JenkinsInstance{}, domain.JenkinsPipeline{}, fmt.Errorf("当前产品绑定的 Jenkins 不存在或地址为空")
	}
	pipeline, ok := selectJenkinsPipeline(environment, jenkins, jobName, jobURL)
	if !ok {
		return domain.JenkinsInstance{}, domain.JenkinsPipeline{}, fmt.Errorf("未在当前产品绑定的 Jenkins view 中找到 Pipeline：%s", jobName)
	}
	return jenkins, pipeline, nil
}

func (c *ReleaseCreator) resolveJenkinsBuildParameters(ctx context.Context, jenkins domain.JenkinsInstance, pipeline domain.JenkinsPipeline, requestParameters map[string]string, branch string) (map[string]string, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	_ = jenkins
	parameters := pipeline.Parameters
	parameterized := len(parameters) > 0 || len(requestParameters) > 0
	definedParameters := map[string]bool{}
	result := map[string]string{}
	for _, parameter := range parameters {
		name := strings.TrimSpace(parameter.Name)
		if name == "" {
			continue
		}
		definedParameters[name] = true
		result[name] = parameter.DefaultValue
	}
	for key, value := range requestParameters {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if len(definedParameters) > 0 && !definedParameters[trimmedKey] {
			continue
		}
		result[trimmedKey] = value
	}
	applyCompatibleBranchParameter(result, parameters, branch)
	for _, parameter := range parameters {
		name := strings.TrimSpace(parameter.Name)
		if name == "" || !parameter.Required {
			continue
		}
		if strings.TrimSpace(result[name]) == "" {
			return nil, parameterized, fmt.Errorf("请填写 Jenkins 参数：%s", name)
		}
	}
	return result, parameterized, nil
}

func selectJenkinsPipeline(environment domain.Environment, jenkins domain.JenkinsInstance, jobName string, jobURL string) (domain.JenkinsPipeline, bool) {
	trimmedJob := strings.TrimSpace(jobName)
	normalizedJobURL := normalizeJenkinsJobURL(jobURL)
	if trimmedJob == "" {
		return domain.JenkinsPipeline{}, false
	}
	viewSet := map[string]bool{}
	if view := strings.TrimSpace(environment.JenkinsView); view != "" {
		for _, key := range jenkinsViewKeyCandidates(view) {
			viewSet[key] = true
		}
	}
	for _, binding := range environment.Bindings {
		if binding.ResourceType != "JENKINS" || strings.TrimSpace(binding.ResourceID) != strings.TrimSpace(jenkins.ID) {
			continue
		}
		if binding.BindingRole != "" && binding.BindingRole != "BUILD_SOURCE" {
			continue
		}
		if binding.ScopeType == "VIEW" {
			for _, key := range jenkinsViewKeyCandidates(binding.ScopeValue) {
				viewSet[key] = true
			}
		}
	}
	hasViewFilter := len(viewSet) > 0
	for _, pipeline := range jenkins.Pipelines {
		if strings.TrimSpace(pipeline.Name) != trimmedJob {
			continue
		}
		if normalizedJobURL != "" && !sameJenkinsJobURL(pipeline.URL, normalizedJobURL) {
			continue
		}
		if len(viewSet) == 0 || jenkinsPipelineMatchesView(pipeline, viewSet) {
			return pipeline, true
		}
	}
	for _, job := range jenkins.Jobs {
		if hasViewFilter {
			break
		}
		if strings.TrimSpace(job) != trimmedJob {
			continue
		}
		if normalizedJobURL != "" {
			continue
		}
		return domain.JenkinsPipeline{Name: trimmedJob}, true
	}
	return domain.JenkinsPipeline{}, false
}

func sameJenkinsJobURL(left string, right string) bool {
	return normalizeJenkinsJobURL(left) == normalizeJenkinsJobURL(right)
}

func normalizeJenkinsJobURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if decoded, err := url.PathUnescape(trimmed); err == nil && decoded != "" {
		return strings.TrimRight(decoded, "/")
	}
	return trimmed
}

func applyCompatibleBranchParameter(result map[string]string, parameters []domain.JenkinsPipelineParameter, branch string) {
	trimmedBranch := strings.TrimSpace(branch)
	if trimmedBranch == "" {
		return
	}
	for _, parameter := range parameters {
		name := strings.TrimSpace(parameter.Name)
		if name == "" || !isJenkinsBranchParameter(name) {
			continue
		}
		if strings.TrimSpace(result[name]) == "" {
			result[name] = trimmedBranch
		}
		return
	}
}

func isJenkinsBranchParameter(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "branch", "branch_name", "branchname", "git_branch", "gitbranch", "git_branch_name", "gitbranchname", "ref", "git_ref":
		return true
	default:
		return false
	}
}

func jenkinsPipelineMatchesView(pipeline domain.JenkinsPipeline, viewSet map[string]bool) bool {
	for _, value := range []string{pipeline.View, pipeline.ViewURL} {
		for _, key := range jenkinsViewKeyCandidates(value) {
			if viewSet[key] {
				return true
			}
		}
	}
	return false
}

func jenkinsViewKeyCandidates(value string) []string {
	normalized := strings.Trim(strings.ToLower(strings.TrimSpace(value)), "/")
	if normalized == "" {
		return nil
	}
	keys := []string{normalized}
	if parsed, err := url.Parse(normalized); err == nil && parsed.Path != "" {
		pathValue := strings.Trim(strings.ToLower(parsed.Path), "/")
		keys = append(keys, pathValue)
		keys = append(keys, extractJenkinsViewPathKeys(pathValue)...)
	}
	keys = append(keys, extractJenkinsViewPathKeys(normalized)...)
	if decoded, err := url.PathUnescape(normalized); err == nil && decoded != normalized {
		decoded = strings.Trim(strings.ToLower(strings.TrimSpace(decoded)), "/")
		keys = append(keys, decoded)
		keys = append(keys, extractJenkinsViewPathKeys(decoded)...)
	}
	return uniqueStrings(keys)
}

func extractJenkinsViewPathKeys(value string) []string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	keys := []string{}
	for index := 0; index < len(parts); index++ {
		if parts[index] == "view" && index+1 < len(parts) {
			keys = append(keys, parts[index+1])
			keys = append(keys, "view/"+parts[index+1])
		}
	}
	return keys
}

func (c *ReleaseCreator) resolveReleaseServices(productID string, requestedServiceIDs []string) ([]string, []string) {
	serviceIDs := normalizeStringSlice(requestedServiceIDs)
	if len(serviceIDs) == 0 {
		return []string{}, []string{}
	}
	managedByID := make(map[string]domain.ManagedService)
	if c.managed != nil {
		for _, item := range c.managed.ListManagedServices(productID) {
			managedByID[item.ID] = item
		}
	}
	serviceNames := make([]string, 0, len(serviceIDs))
	for _, serviceID := range serviceIDs {
		if item, ok := managedByID[serviceID]; ok && strings.TrimSpace(item.Name) != "" {
			serviceNames = append(serviceNames, strings.TrimSpace(item.Name))
			continue
		}
		serviceNames = append(serviceNames, serviceID)
	}
	return serviceIDs, serviceNames
}

func (c *ReleaseCreator) triggerJenkinsReleaseAsync(releaseID string, environment domain.Environment, jenkins domain.JenkinsInstance, pipeline domain.JenkinsPipeline, parameterized bool, parameters map[string]string, agentID string, targetEnvironmentID string) {
	if c.integrations.Jenkins == nil {
		c.updateReleaseTriggerFailure(releaseID, "Jenkins 集成未初始化")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		c.updateReleaseBuildStatus(releaseID, "", "TRIGGERING", pipeline.URL, "JENKINS_TRIGGERING", 5)
		build, err := c.integrations.Jenkins.TriggerBuild(ctx, integration.BuildRequest{
			JenkinsURL:            jenkins.URL,
			Username:              jenkins.Username,
			Token:                 jenkins.Token,
			InsecureSkipTLSVerify: jenkins.InsecureSkipTLSVerify,
			JobName:               pipeline.Name,
			JobURL:                pipeline.URL,
			Parameterized:         parameterized,
			Parameters:            parameters,
		})
		if err != nil {
			log.Printf("jenkins trigger failed: release=%s environment=%s jenkins=%s job=%s jobURL=%s reason=%s", releaseID, environment.ID, jenkins.ID, pipeline.Name, pipeline.URL, err.Error())
			c.updateReleaseTriggerFailure(releaseID, err.Error())
			return
		}
		status := releaseStatusFromBuildStatus(build.Status)
		progress := 10
		if status == "RUNNING" {
			progress = 60
		}
		if status == "SUCCESS" || status == "FAILED" {
			progress = 100
		}
		c.updateReleaseBuildStatus(releaseID, build.BuildID, build.Status, build.URL, status, progress)
		if environment.NetworkMode == "AGENT" {
			c.enqueueIfNeeded(context.Background(), releaseID, "release", "project-agent-sync", agentID, targetEnvironmentID)
		}
	}()
}

func (c *ReleaseCreator) updateReleaseTriggerFailure(releaseID string, reason string) {
	log.Printf("jenkins release trigger failed: release=%s reason=%s", releaseID, compactLogMessage(reason))
	c.updateReleaseBuildStatus(releaseID, "", "TRIGGER_FAILED", "", "FAILED", 100)
}

func (c *ReleaseCreator) updateReleaseBuildStatus(releaseID string, buildID string, buildStatus string, buildURL string, status string, progress int) {
	updater, ok := c.orders.(ReleaseOrderBuildUpdater)
	if !ok || updater == nil {
		return
	}
	if _, found, err := updater.UpdateReleaseBuildStatus(releaseID, buildID, buildStatus, buildURL, status, progress); err != nil || !found {
		if err != nil {
			log.Printf("release build status update failed: release=%s reason=%s", releaseID, err.Error())
			return
		}
		log.Printf("release build status update skipped: release=%s reason=release not found", releaseID)
	}
}

func releaseStatusFromBuildStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS":
		return "SUCCESS"
	case "FAILURE", "ABORTED", "UNSTABLE", "NOT_BUILT", "TRIGGER_FAILED":
		return "FAILED"
	case "BUILDING", "RUNNING":
		return "RUNNING"
	case "QUEUED":
		return "JENKINS_QUEUED"
	case "TRIGGERING":
		return "JENKINS_TRIGGERING"
	default:
		return "JENKINS_QUEUED"
	}
}

func compactLogMessage(message string) string {
	compact := strings.Join(strings.Fields(message), " ")
	const maxLength = 200
	if len([]rune(compact)) <= maxLength {
		return compact
	}
	runes := []rune(compact)
	return string(runes[:maxLength]) + "..."
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
			JenkinsID:             input.JenkinsID,
			JenkinsJobName:        input.JenkinsJobName,
			JenkinsJobURL:         input.JenkinsJobURL,
			ImageRepository:       input.ImageRepository,
			ImageTag:              input.ImageTag,
			ImageDigest:           input.ImageDigest,
			TargetEnvironmentID:   input.TargetEnvironmentID,
			TargetEnvironmentName: input.TargetEnvironmentID,
			Status:                input.Status,
			Progress:              input.Progress,
			AgentName:             input.AgentID,
			ServiceIDs:            input.ServiceIDs,
			ServiceNames:          input.ServiceNames,
		}, nil
	}
	order, err := c.orders.CreateReleaseOrder(input)
	if err != nil {
		return domain.ReleaseOrder{}, ErrReleaseOrderCreate
	}
	return order, nil
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func (c *ReleaseCreator) nextID(prefix string) string {
	now := c.now()
	return fmt.Sprintf("%s-%s-%06d", prefix, now.Format("20060102-150405"), now.Nanosecond()/1000)
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
