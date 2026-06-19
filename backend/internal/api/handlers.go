package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/repository"
	"ops-release-platform/backend/internal/service"
)

type Handler struct {
	repo         repository.Store
	queue        *agent.Queue
	protocol     agent.Protocol
	integrations integration.Suite
	service      *service.ReleaseCreator
}

type kubernetesClusterRequest struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	APIServer  string `json:"apiServer"`
	Context    string `json:"context"`
	Kubeconfig string `json:"kubeconfig"`
}

type harborRegistryRequest struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	Scheme                string `json:"scheme"`
	Username              string `json:"username"`
	Password              string `json:"password"`
	InsecureSkipTLSVerify bool   `json:"insecureSkipTLSVerify"`
}

type jenkinsInstanceRequest struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	URL                   string `json:"url"`
	Username              string `json:"username"`
	Token                 string `json:"token"`
	InsecureSkipTLSVerify bool   `json:"insecureSkipTLSVerify"`
}

type environmentUpdateRequest struct {
	ID               *string                              `json:"id"`
	Name             *string                              `json:"name"`
	Code             *string                              `json:"code"`
	Type             *string                              `json:"type"`
	DeployTargetType *string                              `json:"deployTargetType"`
	NetworkMode      *string                              `json:"networkMode"`
	ClusterID        *string                              `json:"clusterId"`
	Namespace        *string                              `json:"namespace"`
	RegistryID       *string                              `json:"registryId"`
	RegistryProject  *string                              `json:"registryProject"`
	JenkinsID        *string                              `json:"jenkinsId"`
	JenkinsView      *string                              `json:"jenkinsView"`
	Bindings         *[]domain.EnvironmentResourceBinding `json:"bindings"`
	Status           *string                              `json:"status"`
}

func NewHandler(repo repository.Store, queue *agent.Queue, protocol agent.Protocol, integrations integration.Suite) *Handler {
	if protocol == nil {
		protocol = agent.NewProtocolStore()
	}
	handler := &Handler{repo: repo, queue: queue, protocol: protocol, integrations: integrations}
	handler.service = service.NewReleaseCreator(integrations, repo, repo, handler.enqueueTask, repo)
	return handler
}

func (h *Handler) Login(c *gin.Context) {
	user := h.repo.GetCurrentUser()
	OK(c, gin.H{
		"token": "mock-token-admin",
		"user":  user,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	OK(c, gin.H{"success": true})
}

func (h *Handler) Me(c *gin.Context) {
	OK(c, h.repo.GetCurrentUser())
}

func (h *Handler) ListUsers(c *gin.Context) {
	OK(c, paginate(h.repo.ListUsers(c.Query("keyword")), c))
}

func (h *Handler) ListRoles(c *gin.Context) {
	OK(c, paginate(h.repo.ListRoles(c.Query("keyword")), c))
}

func (h *Handler) ListPermissions(c *gin.Context) {
	OK(c, paginate(h.repo.ListPermissions(c.Query("keyword")), c))
}

func (h *Handler) ListChangelog(c *gin.Context) {
	OK(c, paginate(h.repo.ListChangelog(c.Query("keyword")), c))
}

func (h *Handler) ListEnvironments(c *gin.Context) {
	OK(c, paginate(h.repo.ListEnvironments(c.Query("keyword")), c))
}

func (h *Handler) GetEnvironment(c *gin.Context) {
	environment, ok := h.repo.GetEnvironment(c.Param("id"))
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	OK(c, environment)
}

func (h *Handler) CreateEnvironment(c *gin.Context) {
	var request domain.Environment
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid environment request")
		return
	}
	environment, err := h.repo.CreateEnvironment(request)
	if err != nil {
		BadRequest(c, "invalid environment request")
		return
	}
	Created(c, environment)
}

func (h *Handler) UpdateEnvironment(c *gin.Context) {
	existing, ok := h.repo.GetEnvironment(c.Param("id"))
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		BadRequest(c, "invalid environment request")
		return
	}
	var request environmentUpdateRequest
	if err := json.Unmarshal(body, &request); err != nil {
		BadRequest(c, "invalid environment request")
		return
	}
	merged := mergeEnvironmentUpdate(existing, request)
	environment, ok, err := h.repo.UpdateEnvironment(c.Param("id"), merged)
	if err != nil {
		BadRequest(c, "invalid environment request")
		return
	}
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	OK(c, environment)
}

func mergeEnvironmentUpdate(existing domain.Environment, request environmentUpdateRequest) domain.Environment {
	merged := existing
	if request.ID != nil {
		merged.ID = *request.ID
	}
	if request.Name != nil {
		merged.Name = *request.Name
	}
	if request.Code != nil {
		merged.Code = *request.Code
	}
	if request.Type != nil {
		merged.Type = *request.Type
	}
	if request.DeployTargetType != nil {
		merged.DeployTargetType = *request.DeployTargetType
	}
	if request.NetworkMode != nil {
		merged.NetworkMode = *request.NetworkMode
	}
	if request.ClusterID != nil {
		merged.ClusterID = *request.ClusterID
	}
	if request.Namespace != nil {
		merged.Namespace = *request.Namespace
	}
	if request.RegistryID != nil {
		merged.RegistryID = *request.RegistryID
	}
	if request.RegistryProject != nil {
		merged.RegistryProject = *request.RegistryProject
	}
	if request.JenkinsID != nil {
		merged.JenkinsID = *request.JenkinsID
	}
	if request.JenkinsView != nil {
		merged.JenkinsView = *request.JenkinsView
	}
	if request.Bindings != nil {
		merged.Bindings = *request.Bindings
	}
	if request.Status != nil {
		merged.Status = *request.Status
	}
	return merged
}

func (h *Handler) CheckEnvironment(c *gin.Context) {
	environmentID := c.Param("id")
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	if environment.Type != "LOCAL" {
		BadRequest(c, "remote environment checks are handled by agent")
		return
	}
	if h.integrations.IsMock() {
		BadRequest(c, "real environment integrations are not configured")
		return
	}
	environment, err := h.environmentWithIntegrationResources(environment)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	checks := make([]integration.IntegrationCheck, 0, 2)
	if h.integrations.Kubernetes != nil {
		check, err := h.integrations.Kubernetes.CheckConnection(c.Request.Context(), environment)
		if err != nil {
			log.Printf("environment %s kubernetes check failed: %v", environmentID, err)
			_, _, _ = h.repo.UpdateEnvironmentCheck(environmentID, "UNHEALTHY", time.Now())
			BadRequest(c, "kubernetes check failed")
			return
		}
		checks = append(checks, check)
	}
	if h.integrations.Registry != nil {
		check, err := h.integrations.Registry.CheckConnection(c.Request.Context(), environment)
		if err != nil {
			log.Printf("environment %s registry check failed: %v", environmentID, err)
			_, _, _ = h.repo.UpdateEnvironmentCheck(environmentID, "UNHEALTHY", time.Now())
			BadRequest(c, "registry check failed")
			return
		}
		checks = append(checks, check)
	}
	checkedAt := time.Now()
	_, _, _ = h.repo.UpdateEnvironmentCheck(environmentID, "HEALTHY", checkedAt)
	OK(c, gin.H{
		"environmentId": environmentID,
		"status":        "HEALTHY",
		"checkedAt":     checkedAt.Format(time.RFC3339),
		"checks":        checks,
	})
}

func (h *Handler) environmentWithIntegrationResources(environment domain.Environment) (domain.Environment, error) {
	if strings.TrimSpace(environment.ClusterID) == "" {
		return domain.Environment{}, fmt.Errorf("kubernetes cluster is required")
	}
	if strings.TrimSpace(environment.Namespace) == "" {
		return domain.Environment{}, fmt.Errorf("kubernetes namespace is required")
	}
	cluster, ok := h.repo.GetKubernetesCluster(environment.ClusterID)
	if !ok {
		return domain.Environment{}, fmt.Errorf("kubernetes cluster not found")
	}
	if strings.TrimSpace(environment.RegistryID) == "" {
		return domain.Environment{}, fmt.Errorf("harbor registry is required")
	}
	if strings.TrimSpace(environment.RegistryProject) == "" {
		return domain.Environment{}, fmt.Errorf("harbor project is required")
	}
	registry, ok := h.repo.GetHarborRegistry(environment.RegistryID)
	if !ok {
		return domain.Environment{}, fmt.Errorf("harbor registry not found")
	}
	environment.ClusterAPIServer = cluster.APIServer
	environment.ClusterCredentialRef = cluster.CredentialRef
	environment.RegistryURL = registry.URL
	environment.RegistryCredentialRef = registry.CredentialRef
	if strings.TrimSpace(environment.JenkinsID) != "" {
		jenkins, ok := h.repo.GetJenkinsInstance(environment.JenkinsID)
		if !ok {
			return domain.Environment{}, fmt.Errorf("jenkins instance not found")
		}
		environment.JenkinsURL = jenkins.URL
		environment.JenkinsCredentialRef = jenkins.CredentialRef
	}
	return environment, nil
}

func (h *Handler) ListKubernetesClusters(c *gin.Context) {
	OK(c, paginate(h.repo.ListKubernetesClusters(c.Query("keyword")), c))
}

func (h *Handler) CreateKubernetesCluster(c *gin.Context) {
	var request kubernetesClusterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	item, err := h.repo.CreateKubernetesCluster(domain.KubernetesCluster{
		ID:         request.ID,
		Name:       request.Name,
		APIServer:  request.APIServer,
		Context:    request.Context,
		Kubeconfig: request.Kubeconfig,
	})
	if err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	Created(c, item)
}

func (h *Handler) UpdateKubernetesCluster(c *gin.Context) {
	var request kubernetesClusterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	item, ok, err := h.repo.UpdateKubernetesCluster(c.Param("id"), domain.KubernetesCluster{
		ID:         request.ID,
		Name:       request.Name,
		APIServer:  request.APIServer,
		Context:    request.Context,
		Kubeconfig: request.Kubeconfig,
	})
	if err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	if !ok {
		NotFound(c, "kubernetes cluster not found")
		return
	}
	OK(c, item)
}

func (h *Handler) ListHarborRegistries(c *gin.Context) {
	OK(c, paginate(h.repo.ListHarborRegistries(c.Query("keyword")), c))
}

func (h *Handler) CreateHarborRegistry(c *gin.Context) {
	var request harborRegistryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid harbor registry request")
		return
	}
	item, err := h.repo.CreateHarborRegistry(domain.HarborRegistry{
		ID:                    request.ID,
		Name:                  request.Name,
		URL:                   request.URL,
		Scheme:                request.Scheme,
		Username:              request.Username,
		Password:              request.Password,
		InsecureSkipTLSVerify: request.InsecureSkipTLSVerify,
	})
	if err != nil {
		BadRequest(c, "invalid harbor registry request")
		return
	}
	Created(c, item)
}

func (h *Handler) UpdateHarborRegistry(c *gin.Context) {
	var request harborRegistryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid harbor registry request")
		return
	}
	item, ok, err := h.repo.UpdateHarborRegistry(c.Param("id"), domain.HarborRegistry{
		ID:                    request.ID,
		Name:                  request.Name,
		URL:                   request.URL,
		Scheme:                request.Scheme,
		Username:              request.Username,
		Password:              request.Password,
		InsecureSkipTLSVerify: request.InsecureSkipTLSVerify,
	})
	if err != nil {
		BadRequest(c, "invalid harbor registry request")
		return
	}
	if !ok {
		NotFound(c, "harbor registry not found")
		return
	}
	OK(c, item)
}

func (h *Handler) ListJenkinsInstances(c *gin.Context) {
	OK(c, paginate(h.repo.ListJenkinsInstances(c.Query("keyword")), c))
}

func (h *Handler) CreateJenkinsInstance(c *gin.Context) {
	var request jenkinsInstanceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid jenkins instance request")
		return
	}
	item, err := h.repo.CreateJenkinsInstance(domain.JenkinsInstance{
		ID:                    request.ID,
		Name:                  request.Name,
		URL:                   request.URL,
		Username:              request.Username,
		Token:                 request.Token,
		InsecureSkipTLSVerify: request.InsecureSkipTLSVerify,
	})
	if err != nil {
		BadRequest(c, "invalid jenkins instance request")
		return
	}
	Created(c, item)
}

func (h *Handler) UpdateJenkinsInstance(c *gin.Context) {
	var request jenkinsInstanceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid jenkins instance request")
		return
	}
	item, ok, err := h.repo.UpdateJenkinsInstance(c.Param("id"), domain.JenkinsInstance{
		ID:                    request.ID,
		Name:                  request.Name,
		URL:                   request.URL,
		Username:              request.Username,
		Token:                 request.Token,
		InsecureSkipTLSVerify: request.InsecureSkipTLSVerify,
	})
	if err != nil {
		BadRequest(c, "invalid jenkins instance request")
		return
	}
	if !ok {
		NotFound(c, "jenkins instance not found")
		return
	}
	OK(c, item)
}

func (h *Handler) ListAgents(c *gin.Context) {
	OK(c, paginate(h.repo.ListAgents(c.Query("keyword")), c))
}

func (h *Handler) CreateAgentRegisterToken(c *gin.Context) {
	var request struct {
		AgentID       string `json:"agentId"`
		EnvironmentID string `json:"environmentId"`
		TTLMinutes    int    `json:"ttlMinutes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid register token request")
		return
	}
	if request.EnvironmentID == "" {
		BadRequest(c, "environmentId is required")
		return
	}
	if request.TTLMinutes <= 0 {
		request.TTLMinutes = 10
	}
	if _, exists := h.repo.GetEnvironment(request.EnvironmentID); !exists {
		BadRequest(c, "environment not found")
		return
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	if request.AgentID == "" {
		request.AgentID = "agent-" + strings.TrimPrefix(request.EnvironmentID, "env-")
	}
	token := "agt_" + request.EnvironmentID + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	baseURL := requestBaseURL(c)
	Created(c, gin.H{
		"token":     token,
		"expiresAt": time.Now().Add(time.Duration(request.TTLMinutes) * time.Minute).Format(time.RFC3339),
		"installCommand": strings.Join([]string{
			"cat > agent.env <<'EOF'",
			"AGENT_ID=" + request.AgentID,
			"AGENT_ENVIRONMENT_ID=" + request.EnvironmentID,
			"PLATFORM_URL=" + baseURL,
			"AGENT_TOKEN=" + token,
			"AGENT_MODE=mock",
			"AGENT_HEALTH_PORT=18080",
			"AGENT_POLL_INTERVAL_SECONDS=5",
			"AGENT_HEARTBEAT_INTERVAL_SECONDS=15",
			"AGENT_HTTP_TIMEOUT_SECONDS=10",
			"AGENT_MAX_TASKS=1",
			"AGENT_CAPABILITIES=mock-executor,image-sync,kubectl,http-check",
			"EOF",
			"./ops-release-agent -f ./agent.env",
		}, "\n"),
	})
}

func (h *Handler) AgentHeartbeat(c *gin.Context) {
	var request struct {
		EnvironmentID string   `json:"environmentId"`
		Version       string   `json:"version"`
		Capabilities  []string `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid heartbeat request")
		return
	}
	agentID := c.Param("id")
	request.EnvironmentID = strings.TrimSpace(request.EnvironmentID)
	if request.EnvironmentID != "" {
		if _, exists := h.repo.GetEnvironment(request.EnvironmentID); !exists {
			BadRequest(c, "environment not found")
			return
		}
	}
	agentItem, ok := h.repo.UpdateAgentHeartbeat(agentID, request.EnvironmentID, request.Version, request.Capabilities)
	if !ok && request.EnvironmentID != "" {
		agentItem, ok = h.repo.UpsertAgent(agentID, request.EnvironmentID, request.Version, request.Capabilities, "ONLINE")
	}
	if !ok {
		NotFound(c, "agent not found")
		return
	}
	OK(c, gin.H{
		"agent":       agentItem,
		"serverTime":  time.Now().Format(time.RFC3339),
		"nextPollSec": 5,
	})
}

func (h *Handler) PullAgentTask(c *gin.Context) {
	agentItem, ok := h.repo.GetAgent(c.Param("id"))
	if !ok {
		NotFound(c, "agent not found")
		return
	}
	if agentItem.Status != "ONLINE" {
		BadRequest(c, "agent must be ONLINE")
		return
	}
	task, ok := h.protocol.Pull(agentItem.ID)
	if !ok {
		OK(c, gin.H{
			"task": nil,
		})
		return
	}
	h.repo.AssignAgentTask(agentItem.ID, task.ID)
	OK(c, gin.H{
		"task": task,
	})
}

func (h *Handler) LeaseAgentTask(c *gin.Context) {
	var request struct {
		AgentID       string `json:"agentId"`
		EnvironmentID string `json:"environmentId"`
		MaxTasks      int    `json:"maxTasks"`
		LeaseSeconds  int    `json:"leaseSeconds"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid lease request")
		return
	}
	if request.AgentID == "" {
		BadRequest(c, "agentId is required")
		return
	}
	request.EnvironmentID = strings.TrimSpace(request.EnvironmentID)
	agentItem, ok := h.repo.GetAgent(request.AgentID)
	if !ok {
		NotFound(c, "agent not found")
		return
	}
	if agentItem.Status != "ONLINE" {
		BadRequest(c, "agent must be ONLINE")
		return
	}
	if request.EnvironmentID != "" && agentItem.EnvironmentID != request.EnvironmentID {
		BadRequest(c, fmt.Sprintf(
			"agent does not belong to target environment: agentId=%s requestedEnvironmentId=%s boundEnvironmentId=%s",
			agentItem.ID,
			request.EnvironmentID,
			agentItem.EnvironmentID,
		))
		return
	}
	result := h.protocol.Lease(agent.LeaseRequest{
		AgentID:       agentItem.ID,
		EnvironmentID: request.EnvironmentID,
		MaxTasks:      request.MaxTasks,
		LeaseSeconds:  request.LeaseSeconds,
		CallbackBase:  requestBaseURL(c),
	})
	if result.Leased && result.Task != nil {
		h.repo.AssignAgentTask(agentItem.ID, result.Task.ID)
	}
	OK(c, result)
}

func (h *Handler) ReportAgentTaskStep(c *gin.Context) {
	var request struct {
		Step   string `json:"step"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid step report")
		return
	}
	if request.Step == "" || request.Status == "" {
		BadRequest(c, "step and status are required")
		return
	}
	task, ok := h.protocol.ReportStep(c.Param("id"), request.Step, request.Status)
	if !ok {
		NotFound(c, "agent task not found")
		return
	}
	OK(c, task)
}

func (h *Handler) AppendAgentTaskLog(c *gin.Context) {
	var request struct {
		Line string `json:"line"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid log report")
		return
	}
	if request.Line == "" {
		BadRequest(c, "line is required")
		return
	}
	task, ok := h.protocol.AppendLog(c.Param("id"), request.Line)
	if !ok {
		NotFound(c, "agent task not found")
		return
	}
	OK(c, task)
}

func (h *Handler) ReportAgentTaskResult(c *gin.Context) {
	var request struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid result report")
		return
	}
	if request.Status == "" {
		BadRequest(c, "status is required")
		return
	}
	task, ok := h.protocol.ReportResult(c.Param("id"), request.Status, request.Message)
	if !ok {
		NotFound(c, "agent task not found")
		return
	}
	if task.AgentID != "" && (request.Status == "SUCCESS" || request.Status == "FAILED") {
		h.repo.AssignAgentTask(task.AgentID, "")
	}
	OK(c, task)
}

func (h *Handler) CreateBaseline(c *gin.Context) {
	var request struct {
		SourceEnvironmentID string `json:"sourceEnvironmentId"`
		Name                string `json:"name"`
		Purpose             string `json:"purpose"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid baseline request")
		return
	}
	if request.SourceEnvironmentID == "" || request.Name == "" {
		BadRequest(c, "sourceEnvironmentId and name are required")
		return
	}
	detail, err := h.repo.CreateBaseline(request.SourceEnvironmentID, request.Name, request.Purpose)
	if err != nil {
		BadRequest(c, "source environment not found")
		return
	}
	Created(c, detail)
}

func (h *Handler) ListBaselines(c *gin.Context) {
	OK(c, paginate(h.repo.ListBaselines(c.Query("keyword")), c))
}

func (h *Handler) GetBaseline(c *gin.Context) {
	detail, ok := h.repo.GetBaselineDetail(c.Param("id"))
	if !ok {
		NotFound(c, "baseline not found")
		return
	}
	OK(c, detail)
}

func (h *Handler) LockBaseline(c *gin.Context) {
	detail, ok := h.repo.LockBaseline(c.Param("id"))
	if !ok {
		NotFound(c, "baseline not found")
		return
	}
	OK(c, detail)
}

func (h *Handler) CompareBaseline(c *gin.Context) {
	var request struct {
		TargetEnvironmentID  string `json:"targetEnvironmentId"`
		RefreshTargetRuntime bool   `json:"refreshTargetRuntime"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid compare request")
		return
	}
	result, ok := h.repo.GetDiffResult(c.Param("id"), request.TargetEnvironmentID)
	if !ok {
		NotFound(c, "baseline not found")
		return
	}
	OK(c, result)
}

func (h *Handler) ListReleases(c *gin.Context) {
	OK(c, paginate(h.repo.ListReleases(c.Query("keyword")), c))
}

func (h *Handler) ListReleaseSources(c *gin.Context) {
	environmentID := strings.TrimSpace(c.Query("environmentId"))
	if environmentID == "" {
		BadRequest(c, "environmentId is required")
		return
	}
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	if h.integrations.Registry == nil {
		BadRequest(c, "registry integration is not configured")
		return
	}

	services := h.repo.ListReleaseSourceServices(c.Query("keyword"))
	for index := range services {
		if strings.TrimSpace(services[index].ImageRepository) == "" {
			services[index].Tags = []domain.ReleaseImageTag{}
			services[index].Publishable = false
			services[index].Message = "image repository is not configured"
			continue
		}
		tags, err := h.integrations.Registry.ListImageTags(c.Request.Context(), environment, services[index].ImageRepository)
		if err != nil {
			services[index].Tags = []domain.ReleaseImageTag{}
			services[index].Publishable = false
			services[index].Message = "registry image tags unavailable"
			continue
		}
		services[index].Tags = toReleaseImageTags(tags)
		services[index].Publishable = len(services[index].Tags) > 0
		if !services[index].Publishable {
			services[index].Message = "no image tags found"
		}
	}

	OK(c, domain.ReleaseSource{
		EnvironmentID: environmentID,
		Services:      services,
		JenkinsJobs:   []string{},
	})
}

func (h *Handler) CreateRelease(c *gin.Context) {
	var request service.CreateReleaseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid release request")
		return
	}
	result, err := h.service.CreateRelease(c.Request.Context(), request)
	if err != nil {
		switch err {
		case service.ErrInvalidReleaseType:
			BadRequest(c, "release type must be SERVICE_RELEASE")
		case service.ErrAgentNotFound:
			BadRequest(c, "agent not found")
		case service.ErrAgentOffline:
			BadRequest(c, "agent must be ONLINE")
		case service.ErrAgentEnvironment:
			BadRequest(c, "agent does not belong to target environment")
		case service.ErrEnvironmentPermission:
			Forbidden(c, "environment permission denied")
		case service.ErrInvalidReleaseSource:
			BadRequest(c, "release source must be LOCAL_HARBOR_IMAGE or JENKINS_JOB")
		case service.ErrEnvironmentNotFound:
			BadRequest(c, "environment not found")
		case service.ErrReleaseOrderCreate:
			BadRequest(c, "release order create failed")
		case service.ErrJenkinsTrigger:
			BadRequest(c, "jenkins trigger failed")
		case service.ErrRegistryImageCheck:
			BadRequest(c, "registry image check failed")
		case service.ErrRegistryImageSync:
			BadRequest(c, "registry image sync failed")
		case service.ErrImageNotFound:
			BadRequest(c, "release image not found")
		case service.ErrReleaseBaselineUnsupported:
			BadRequest(c, "service release must not include source baseline")
		case service.ErrBaselineNotFound:
			BadRequest(c, "baseline not found")
		case service.ErrInvalidServiceSelection:
			BadRequest(c, "release services must come from NEED_UPDATE diff items")
		default:
			BadRequest(c, "invalid release request")
		}
		return
	}
	Created(c, result)
}

func (h *Handler) GetRelease(c *gin.Context) {
	detail, ok := h.repo.GetReleaseDetail(c.Param("id"))
	if !ok {
		NotFound(c, "release not found")
		return
	}
	OK(c, detail)
}

func (h *Handler) RetryRelease(c *gin.Context) {
	if h.protocol != nil {
		h.protocol.ReportStep(c.Param("id"), "retry", "RUNNING")
		h.protocol.AppendLog(c.Param("id"), "release retry requested")
	}
	OK(c, gin.H{
		"id":     c.Param("id"),
		"status": "RUNNING",
		"action": "retry",
	})
}

func (h *Handler) RollbackRelease(c *gin.Context) {
	if h.protocol != nil {
		h.protocol.ReportStep(c.Param("id"), "rollback", "ROLLED_BACK")
		h.protocol.AppendLog(c.Param("id"), "release rollback requested")
	}
	OK(c, gin.H{
		"id":     c.Param("id"),
		"status": "ROLLED_BACK",
		"action": "rollback",
	})
}

func (h *Handler) ListDeployTasks(c *gin.Context) {
	OK(c, paginate(h.repo.ListDeployTasks(c.Query("keyword")), c))
}

func (h *Handler) CreateDeployTask(c *gin.Context) {
	var request service.CreateDeployTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid deploy request")
		return
	}
	result, err := h.service.CreateDeployTask(c.Request.Context(), request)
	if err != nil {
		switch err {
		case service.ErrInvalidDeployType:
			BadRequest(c, "deploy type must be SERVICE_DEPLOYMENT")
		case service.ErrAgentNotFound:
			BadRequest(c, "agent not found")
		case service.ErrAgentOffline:
			BadRequest(c, "agent must be ONLINE")
		case service.ErrAgentEnvironment:
			BadRequest(c, "agent does not belong to target environment")
		case service.ErrEnvironmentPermission:
			Forbidden(c, "environment permission denied")
		case service.ErrWorkloadProbe:
			BadRequest(c, "kubernetes workload probe failed")
		case service.ErrDeployBaselineRequired:
			BadRequest(c, "source baseline is required for service deployment")
		case service.ErrBaselineNotFound:
			BadRequest(c, "baseline not found")
		case service.ErrInvalidServiceSelection:
			BadRequest(c, "deploy services must come from MISSING_IN_TARGET diff items")
		default:
			BadRequest(c, "invalid deploy request")
		}
		return
	}
	Created(c, result)
}

func (h *Handler) GetDeployTask(c *gin.Context) {
	detail, ok := h.repo.GetDeployDetail(c.Param("id"))
	if !ok {
		detail, ok = h.repo.GetDeployDetail("")
		if !ok {
			NotFound(c, "deploy task not found")
			return
		}
		detail.ID = c.Param("id")
	}
	OK(c, detail)
}

func (h *Handler) RetryDeployStep(c *gin.Context) {
	h.stepAction(c, "retry", "RUNNING")
}

func (h *Handler) SkipDeployStep(c *gin.Context) {
	h.stepAction(c, "skip", "SKIPPED")
}

func (h *Handler) ConfirmDeployStep(c *gin.Context) {
	h.stepAction(c, "confirm", "SUCCESS")
}

func (h *Handler) stepAction(c *gin.Context, action string, status string) {
	if h.protocol != nil {
		h.protocol.ReportStep(c.Param("id"), c.Param("stepId"), status)
		h.protocol.AppendLog(c.Param("id"), "deploy step "+action+" requested: "+c.Param("stepId"))
	}
	OK(c, gin.H{
		"taskId": c.Param("id"),
		"stepId": c.Param("stepId"),
		"action": action,
		"status": status,
	})
}

func (h *Handler) GetAgentTaskStatus(c *gin.Context) {
	if h.protocol != nil {
		status, logs, ok := h.protocol.Status(c.Param("id"))
		if ok {
			OK(c, gin.H{
				"enabled": true,
				"status":  status,
				"logs":    logs,
			})
			return
		}
	}
	if h.queue == nil {
		OK(c, gin.H{
			"enabled": false,
			"message": "redis queue is not configured",
		})
		return
	}
	status, err := h.queue.Status(c.Request.Context(), c.Param("id"))
	if err != nil || len(status) == 0 {
		NotFound(c, "agent task status not found")
		return
	}
	logs, err := h.queue.Logs(c.Request.Context(), c.Param("id"))
	if err != nil {
		logs = []string{}
	}
	OK(c, gin.H{
		"enabled": true,
		"status":  status,
		"logs":    logs,
	})
}

func (h *Handler) enqueue(ctx interface{ Request() *http.Request }, id string, taskType string, action string) {
	requestContext := ctx.Request().Context()
	if h.queue == nil {
		return
	}
	_ = h.queue.Enqueue(requestContext, agent.Task{
		ID:        id,
		Type:      taskType,
		Action:    action,
		CreatedAt: time.Now(),
	})
}

func (h *Handler) enqueueTask(ctx context.Context, id string, taskType string, action string, agentID string, environmentID string) {
	task := agent.Task{
		ID:            id,
		Type:          taskType,
		Action:        action,
		AgentID:       agentID,
		EnvironmentID: environmentID,
		Payload: map[string]string{
			"source": "platform",
		},
		CreatedAt: time.Now(),
	}
	if h.protocol != nil {
		h.protocol.Enqueue(task)
	}
	if h.queue == nil {
		return
	}
	_ = h.queue.Enqueue(ctx, task)
}

func requestBaseURL(c *gin.Context) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}

func toReleaseImageTags(items []integration.ImageInfo) []domain.ReleaseImageTag {
	tags := make([]domain.ReleaseImageTag, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Tag) == "" {
			continue
		}
		tags = append(tags, domain.ReleaseImageTag{
			Tag:       item.Tag,
			Digest:    item.Digest,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return tags
}

func paginate[T any](items []T, c *gin.Context) domain.PageResult[T] {
	page := positiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := positiveInt(c.DefaultQuery("pageSize", "20"), 20)
	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return domain.PageResult[T]{
		Items:    items[start:end],
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func NoRoute(c *gin.Context) {
	c.JSON(http.StatusNotFound, Response{
		Code:      "NOT_FOUND",
		Message:   "route not found",
		RequestID: requestID(),
	})
}
