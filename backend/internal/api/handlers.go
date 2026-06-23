package api

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"

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

type projectRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type environmentUpdateRequest struct {
	ID               *string                              `json:"id"`
	Name             *string                              `json:"name"`
	Code             *string                              `json:"code"`
	ProjectID        *string                              `json:"projectId"`
	ProductStatus    *string                              `json:"productStatus"`
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

func (h *Handler) ListProjects(c *gin.Context) {
	OK(c, paginate(h.repo.ListProjects(c.Query("keyword")), c))
}

func (h *Handler) GetProject(c *gin.Context) {
	project, ok := h.repo.GetProject(c.Param("id"))
	if !ok {
		NotFound(c, "project not found")
		return
	}
	OK(c, project)
}

func (h *Handler) CreateProject(c *gin.Context) {
	var request projectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid project request")
		return
	}
	project, err := h.repo.CreateProject(domain.Project{
		ID:          request.ID,
		Name:        request.Name,
		Code:        request.Code,
		Description: request.Description,
		Status:      request.Status,
	})
	if err != nil {
		BadRequest(c, "invalid project request")
		return
	}
	Created(c, project)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	var request projectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid project request")
		return
	}
	project, ok, err := h.repo.UpdateProject(c.Param("id"), domain.Project{
		ID:          request.ID,
		Name:        request.Name,
		Code:        request.Code,
		Description: request.Description,
		Status:      request.Status,
	})
	if err != nil {
		BadRequest(c, "invalid project request")
		return
	}
	if !ok {
		NotFound(c, "project not found")
		return
	}
	OK(c, project)
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
	if request.ProjectID != nil {
		merged.ProjectID = *request.ProjectID
	}
	if request.ProductStatus != nil {
		merged.ProductStatus = *request.ProductStatus
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
	if h.integrations.IsMock() {
		h.checkEnvironmentByCachedScopes(c, environment)
		return
	}
	if environment.Type != "LOCAL" {
		h.checkEnvironmentByCachedScopes(c, environment)
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

func (h *Handler) ProbeEnvironment(c *gin.Context) {
	environmentID := c.Param("id")
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	agentItem, ok := h.findProbeAgent(environmentID)
	if !ok {
		BadRequest(c, "未找到已认领且在线的 Agent，请先在 Agent 管理中完成注册、启动和认领")
		return
	}
	taskID := "PROBE-" + environmentID + "-" + time.Now().Format("20060102150405")
	task := agent.Task{
		ID:            taskID,
		Type:          "probe",
		Action:        "remote_resource_probe",
		AgentID:       agentItem.ID,
		EnvironmentID: environmentID,
		Payload:       probePayloadFromEnvironment(environment),
		CreatedAt:     time.Now(),
	}
	if h.protocol != nil {
		h.protocol.Enqueue(task)
	}
	if h.queue != nil {
		_ = h.queue.Enqueue(c.Request.Context(), task)
	}
	_, _, _ = h.repo.UpdateEnvironmentCheck(environmentID, "VERIFYING", time.Now())
	OK(c, gin.H{
		"taskId":        taskID,
		"agentId":       agentItem.ID,
		"environmentId": environmentID,
		"status":        "PENDING",
		"message":       "远程探测任务已下发，等待 Agent 回传结果",
	})
}

func (h *Handler) ListEnvironmentServices(c *gin.Context) {
	environmentID := c.Param("id")
	if _, ok := h.repo.GetEnvironment(environmentID); !ok {
		NotFound(c, "environment not found")
		return
	}
	OK(c, paginate(h.repo.ListManagedServices(environmentID), c))
}

func (h *Handler) ListDiscoveredEnvironmentServices(c *gin.Context) {
	environmentID := c.Param("id")
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	services, err := h.discoverEnvironmentServices(c.Request.Context(), environment)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, paginate(services, c))
}

func (h *Handler) AdoptEnvironmentServices(c *gin.Context) {
	environmentID := c.Param("id")
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	var input domain.AdoptServiceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "invalid service adopt request")
		return
	}
	if len(input.Services) == 0 {
		BadRequest(c, "请选择需要纳管的服务")
		return
	}
	discovered, err := h.discoverEnvironmentServices(c.Request.Context(), environment)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	byID := make(map[string]domain.DiscoveredService, len(discovered))
	for _, item := range discovered {
		byID[item.ID] = item
	}
	selected := make([]domain.DiscoveredService, 0, len(input.Services))
	seen := map[string]bool{}
	for _, requested := range input.Services {
		id := strings.TrimSpace(requested.ID)
		if id == "" || seen[id] {
			continue
		}
		service, exists := byID[id]
		if !exists {
			BadRequest(c, "只能纳管当前产品已发现的服务")
			return
		}
		seen[id] = true
		selected = append(selected, service)
	}
	if len(selected) == 0 {
		BadRequest(c, "请选择需要纳管的服务")
		return
	}
	services, err := h.repo.UpsertManagedServices(environmentID, selected)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, services)
}

func (h *Handler) RemoveEnvironmentServices(c *gin.Context) {
	environmentID := c.Param("id")
	if _, ok := h.repo.GetEnvironment(environmentID); !ok {
		NotFound(c, "environment not found")
		return
	}
	var input domain.RemoveManagedServiceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "invalid service remove request")
		return
	}
	if len(input.ServiceIDs) == 0 {
		BadRequest(c, "请选择需要移除纳管的服务")
		return
	}
	services, err := h.repo.RemoveManagedServices(environmentID, input.ServiceIDs)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, services)
}

func (h *Handler) ConfirmEnvironmentServiceRegistry(c *gin.Context) {
	environmentID := c.Param("id")
	environment, ok := h.repo.GetEnvironment(environmentID)
	if !ok {
		NotFound(c, "environment not found")
		return
	}
	var input domain.ConfirmServiceRegistryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "invalid private registry confirm request")
		return
	}
	confirmedRegistryHost := normalizedImageRegistry(input.PrivateRegistryHost)
	if confirmedRegistryHost == "" {
		BadRequest(c, "请选择需要确认的私有镜像 registry")
		return
	}
	managed := h.repo.ListManagedServices(environment.ID)
	if len(managed) == 0 {
		BadRequest(c, "请先纳管服务，再确认私有镜像 registry")
		return
	}
	candidates := h.privateRegistryCandidatesFromManagedServices(c.Request.Context(), environment, workloadBindingRole(environment), managed)
	if !registryCandidateAllowed(confirmedRegistryHost, candidates) {
		BadRequest(c, "确认的私有镜像 registry 不在已纳管服务可识别的候选范围内")
		return
	}
	updatedEnvironment, _, err := h.repo.UpdateEnvironment(environment.ID, domain.Environment{
		ID:                  environment.ID,
		Name:                environment.Name,
		Code:                environment.Code,
		ProjectID:           environment.ProjectID,
		ProductStatus:       environment.ProductStatus,
		Type:                environment.Type,
		DeployTargetType:    environment.DeployTargetType,
		NetworkMode:         environment.NetworkMode,
		ClusterID:           environment.ClusterID,
		Namespace:           environment.Namespace,
		RegistryID:          environment.RegistryID,
		RegistryProject:     environment.RegistryProject,
		PrivateRegistryHost: confirmedRegistryHost,
		JenkinsID:           environment.JenkinsID,
		JenkinsView:         environment.JenkinsView,
		Bindings:            environment.Bindings,
		Status:              environment.Status,
	})
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	projects := mapKeys(h.harborScope(c.Request.Context(), updatedEnvironment, workloadBindingRole(updatedEnvironment)).Projects)
	services, err := h.repo.ConfirmManagedServiceRegistry(updatedEnvironment.ID, confirmedRegistryHost, projects)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, services)
}

func (h *Handler) discoverEnvironmentServices(ctx context.Context, environment domain.Environment) ([]domain.DiscoveredService, error) {
	workloads, err := h.environmentWorkloads(ctx, environment)
	if err != nil {
		return nil, err
	}
	namespaceSet := h.boundScopeSet(environment, "K8S", workloadBindingRole(environment))
	harbor := h.harborScopeForDiscovery(ctx, environment, workloadBindingRole(environment))
	harbor = inferHarborScopeRegistryHost(harbor, workloads)
	privateRegistryHost := normalizedImageRegistry(harbor.RegistryHost)
	managed := h.repo.ListManagedServices(environment.ID)
	managedIDs := make(map[string]bool, len(managed))
	for _, item := range managed {
		managedIDs[item.ID] = true
	}
	services := make([]domain.DiscoveredService, 0)
	for _, workload := range workloads {
		namespace := strings.TrimSpace(workload.Namespace)
		if len(namespaceSet) > 0 && !namespaceSet[namespace] {
			continue
		}
		for _, container := range workload.Containers {
			image := strings.TrimSpace(container.Image)
			if image == "" {
				continue
			}
			imageParts := parseContainerImage(image)
			imageSource := classifyImageSource(imageParts, harbor)
			containerType := firstNonEmpty(strings.TrimSpace(container.Type), "APP")
			id := stableDiscoveredServiceID(environment.ID, namespace, workload.Type, workload.Name, containerType, container.Name)
			services = append(services, domain.DiscoveredService{
				ID:                       id,
				ProductID:                environment.ID,
				Name:                     firstNonEmpty(workload.Name, container.Name),
				Namespace:                namespace,
				WorkloadName:             workload.Name,
				WorkloadType:             workload.Type,
				ContainerName:            container.Name,
				ContainerType:            containerType,
				Image:                    image,
				ImageRegistry:            imageParts.Registry,
				ImageProject:             imageParts.Project,
				ImageRepository:          imageParts.Repository,
				ImageTag:                 imageParts.Tag,
				ImageSource:              imageSource,
				PrivateRegistryHost:      privateRegistryHost,
				PrivateRegistryConfirmed: privateRegistryHost != "" && sameRegistryHost(environment.PrivateRegistryHost, privateRegistryHost),
				Replicas:                 workload.Replicas,
				ReadyReplicas:            workload.ReadyReplicas,
				Managed:                  managedIDs[id],
			})
		}
	}
	log.Printf("environment %s service discovery: type=%s namespaces=%v workloads=%d services=%d harborRegistry=%s harborProjects=%v", environment.ID, environment.Type, mapKeys(namespaceSet), len(workloads), len(services), harbor.RegistryHost, mapKeys(harbor.Projects))
	return services, nil
}

func mapKeys(input map[string]bool) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (h *Handler) environmentWorkloads(ctx context.Context, environment domain.Environment) ([]integration.Workload, error) {
	if environment.Type == "PROJECT" {
		agentItem, ok := h.findProbeAgent(environment.ID)
		if !ok {
			return nil, fmt.Errorf("未找到已绑定且在线的 Agent，无法获取远程服务清单")
		}
		if agentItem.RuntimeStatus.Kubernetes.Status != "HEALTHY" {
			return nil, fmt.Errorf("远程 K8s 未就绪：%s", firstNonEmpty(agentItem.RuntimeStatus.Kubernetes.Message, "等待 Agent 上报"))
		}
		return workloadsFromRuntime(agentItem.RuntimeStatus.Kubernetes.Workloads), nil
	}
	clusterID, namespaces := h.kubernetesScopeForDiscovery(environment)
	if clusterID == "" {
		return nil, fmt.Errorf("本地产品未绑定 Kubernetes 集群")
	}
	if len(namespaces) == 0 {
		return nil, fmt.Errorf("本地产品未绑定 Kubernetes namespace")
	}
	cluster, ok := h.repo.GetKubernetesCluster(clusterID)
	if !ok {
		return nil, fmt.Errorf("kubernetes cluster not found")
	}
	if strings.TrimSpace(cluster.Kubeconfig) == "" {
		return nil, fmt.Errorf("kubernetes cluster kubeconfig is required")
	}
	return integration.ListWorkloadsWithKubeconfig(ctx, cluster.Kubeconfig, cluster.APIServer, namespaces, 10*time.Second)
}

func workloadsFromRuntime(items []domain.RuntimeWorkload) []integration.Workload {
	workloads := make([]integration.Workload, 0, len(items))
	for _, item := range items {
		containers := make([]integration.WorkloadContainer, 0, len(item.Containers))
		for _, container := range item.Containers {
			containers = append(containers, integration.WorkloadContainer{
				Name:  container.Name,
				Type:  container.Type,
				Image: container.Image,
			})
		}
		workloads = append(workloads, integration.Workload{
			Namespace:     item.Namespace,
			Name:          item.Name,
			Type:          item.Type,
			Replicas:      item.Replicas,
			ReadyReplicas: item.ReadyReplicas,
			Containers:    containers,
		})
	}
	return workloads
}

func workloadBindingRole(environment domain.Environment) string {
	if environment.Type == "PROJECT" {
		return "RUNTIME_TARGET"
	}
	return "BUILD_SOURCE"
}

func (h *Handler) boundScopeSet(environment domain.Environment, resourceType string, bindingRole string) map[string]bool {
	scopes := map[string]bool{}
	for _, binding := range environment.Bindings {
		if binding.ResourceType == resourceType && binding.BindingRole == bindingRole {
			if scope := strings.TrimSpace(binding.ScopeValue); scope != "" {
				scopes[scope] = true
			}
		}
	}
	if len(scopes) == 0 && resourceType == "K8S" && bindingRole == "BUILD_SOURCE" {
		if scope := strings.TrimSpace(environment.Namespace); scope != "" {
			scopes[scope] = true
		}
	}
	return scopes
}

func (h *Handler) kubernetesScopeForDiscovery(environment domain.Environment) (string, []string) {
	clusterID := strings.TrimSpace(environment.ClusterID)
	namespaces := make([]string, 0)
	for _, binding := range environment.Bindings {
		if binding.ResourceType != "K8S" || binding.BindingRole != workloadBindingRole(environment) {
			continue
		}
		if clusterID == "" {
			clusterID = strings.TrimSpace(binding.ResourceID)
		}
		namespaces = appendUniqueString(namespaces, binding.ScopeValue)
	}
	namespaces = appendUniqueString(namespaces, environment.Namespace)
	return clusterID, namespaces
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

type harborScopeInfo struct {
	RegistryHost string
	Projects     map[string]bool
	Confirmed    bool
}

func (h *Handler) harborScope(ctx context.Context, environment domain.Environment, bindingRole string) harborScopeInfo {
	info := harborScopeInfo{Projects: map[string]bool{}}
	if registryHost := normalizedImageRegistry(environment.PrivateRegistryHost); registryHost != "" {
		info.RegistryHost = registryHost
		info.Confirmed = true
	}
	for _, binding := range environment.Bindings {
		if binding.ResourceType != "HARBOR" || binding.BindingRole != bindingRole {
			continue
		}
		if scope := strings.TrimSpace(binding.ScopeValue); scope != "" {
			info.Projects[scope] = true
		}
		if info.RegistryHost == "" {
			info.RegistryHost = h.harborRegistryHost(ctx, binding.ResourceID)
		}
	}
	if info.RegistryHost == "" {
		info.RegistryHost = h.harborRegistryHost(ctx, environment.RegistryID)
	}
	if len(info.Projects) == 0 {
		if project := strings.TrimSpace(environment.RegistryProject); project != "" {
			info.Projects[project] = true
		}
	}
	return info
}

func (h *Handler) harborScopeForDiscovery(ctx context.Context, environment domain.Environment, bindingRole string) harborScopeInfo {
	if environment.Type != "PROJECT" {
		return h.harborScope(ctx, environment, bindingRole)
	}
	info := harborScopeInfo{Projects: map[string]bool{}}
	if registryHost := normalizedImageRegistry(environment.PrivateRegistryHost); registryHost != "" {
		info.RegistryHost = registryHost
		info.Confirmed = true
	}
	for _, binding := range environment.Bindings {
		if binding.ResourceType != "HARBOR" || binding.BindingRole != bindingRole {
			continue
		}
		if scope := strings.TrimSpace(binding.ScopeValue); scope != "" {
			info.Projects[scope] = true
		}
	}
	agentItem, ok := h.findProbeAgent(environment.ID)
	if !ok {
		return info
	}
	if info.RegistryHost == "" {
		info.RegistryHost = normalizedImageRegistry(agentItem.RuntimeStatus.Harbor.RegistryHost)
	}
	return info
}

func (h *Handler) privateRegistryCandidates(ctx context.Context, environment domain.Environment, bindingRole string, workloads []integration.Workload) []string {
	candidates := []string{}
	if value := normalizedImageRegistry(environment.PrivateRegistryHost); value != "" {
		candidates = appendUniqueRegistryCandidate(candidates, value)
	}
	if environment.Type == "PROJECT" {
		if agentItem, ok := h.findProbeAgent(environment.ID); ok {
			candidates = appendUniqueRegistryCandidate(candidates, agentItem.RuntimeStatus.Harbor.RegistryHost)
		}
	} else {
		for _, binding := range environment.Bindings {
			if binding.ResourceType != "HARBOR" || binding.BindingRole != bindingRole {
				continue
			}
			candidates = appendUniqueRegistryCandidate(candidates, h.harborRegistryHost(ctx, binding.ResourceID))
		}
		candidates = appendUniqueRegistryCandidate(candidates, h.harborRegistryHost(ctx, environment.RegistryID))
	}
	harbor := h.harborScopeForDiscovery(ctx, environment, bindingRole)
	for _, candidate := range inferRegistryCandidatesFromWorkloads(harbor, workloads) {
		candidates = appendUniqueRegistryCandidate(candidates, candidate)
	}
	return candidates
}

func (h *Handler) privateRegistryCandidatesFromManagedServices(ctx context.Context, environment domain.Environment, bindingRole string, services []domain.ManagedService) []string {
	candidates := []string{}
	if value := normalizedImageRegistry(environment.PrivateRegistryHost); value != "" {
		candidates = appendUniqueRegistryCandidate(candidates, value)
	}
	if environment.Type == "PROJECT" {
		if agentItem, ok := h.findProbeAgent(environment.ID); ok {
			candidates = appendUniqueRegistryCandidate(candidates, agentItem.RuntimeStatus.Harbor.RegistryHost)
		}
	} else {
		for _, binding := range environment.Bindings {
			if binding.ResourceType != "HARBOR" || binding.BindingRole != bindingRole {
				continue
			}
			candidates = appendUniqueRegistryCandidate(candidates, h.harborRegistryHost(ctx, binding.ResourceID))
		}
		candidates = appendUniqueRegistryCandidate(candidates, h.harborRegistryHost(ctx, environment.RegistryID))
	}
	harbor := h.harborScope(ctx, environment, bindingRole)
	for _, candidate := range inferRegistryCandidatesFromManagedServices(harbor, services) {
		candidates = appendUniqueRegistryCandidate(candidates, candidate)
	}
	return candidates
}

func appendUniqueRegistryCandidate(values []string, candidate string) []string {
	candidate = normalizedImageRegistry(candidate)
	if candidate == "" {
		return values
	}
	for _, value := range values {
		if sameRegistryHost(value, candidate) {
			return values
		}
	}
	return append(values, candidate)
}

func registryCandidateAllowed(registryHost string, candidates []string) bool {
	for _, candidate := range candidates {
		if sameRegistryHost(registryHost, candidate) {
			return true
		}
	}
	return false
}

func (h *Handler) harborRegistryHost(ctx context.Context, registryID string) string {
	registryID = strings.TrimSpace(registryID)
	if registryID == "" || registryID == runtimeHarborResourceID {
		return ""
	}
	registry, ok := h.repo.GetHarborRegistry(registryID)
	if !ok {
		return ""
	}
	if registryHost := normalizedImageRegistry(registry.RegistryHost); registryHost != "" {
		return registryHost
	}
	_, registryHost, err := checkHarborRegistry(ctx, registry, false)
	if err != nil || registryHost == "" {
		return ""
	}
	if item, ok, updateErr := h.repo.UpdateHarborRegistryProbe(registryID, registry.Status, registry.ProbeMessage, registry.Projects, registryHost, time.Now()); updateErr == nil && ok {
		return normalizedImageRegistry(item.RegistryHost)
	}
	return normalizedImageRegistry(registryHost)
}

const runtimeHarborResourceID = "agent-runtime-harbor"

type parsedContainerImage struct {
	Registry   string
	Project    string
	Repository string
	Tag        string
}

func parseContainerImage(image string) parsedContainerImage {
	image = strings.TrimSpace(image)
	name := image
	if at := strings.Index(name, "@"); at >= 0 {
		name = name[:at]
	}
	tag := ""
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		if colon := strings.LastIndex(name, ":"); colon > slash {
			tag = name[colon+1:]
			name = name[:colon]
		}
	} else if colon := strings.LastIndex(name, ":"); colon >= 0 {
		tag = name[colon+1:]
		name = name[:colon]
	}
	parts := strings.Split(name, "/")
	registry := "docker.io"
	repositoryParts := parts
	if len(parts) > 1 && looksLikeRegistry(parts[0]) {
		registry = normalizedImageRegistry(parts[0])
		repositoryParts = parts[1:]
	}
	project := "library"
	if len(repositoryParts) > 1 {
		project = repositoryParts[0]
	}
	return parsedContainerImage{
		Registry:   registry,
		Project:    project,
		Repository: strings.Join(repositoryParts, "/"),
		Tag:        tag,
	}
}

func classifyImageSource(image parsedContainerImage, harbor harborScopeInfo) string {
	if strings.EqualFold(image.Registry, "docker.io") || strings.EqualFold(image.Registry, "registry-1.docker.io") {
		return "EXTERNAL"
	}
	if harbor.RegistryHost != "" && sameRegistryHost(image.Registry, harbor.RegistryHost) {
		if harbor.Projects[image.Project] {
			return "PRIVATE"
		}
		return "UNMATCHED_PRIVATE"
	}
	return "EXTERNAL"
}

func inferHarborScopeRegistryHost(harbor harborScopeInfo, workloads []integration.Workload) harborScopeInfo {
	if harbor.RegistryHost != "" || len(harbor.Projects) == 0 {
		return harbor
	}
	candidates := inferRegistryCandidatesFromWorkloads(harbor, workloads)
	if len(candidates) != 1 {
		return harbor
	}
	harbor.RegistryHost = candidates[0]
	return harbor
}

func inferRegistryCandidatesFromWorkloads(harbor harborScopeInfo, workloads []integration.Workload) []string {
	if len(harbor.Projects) == 0 {
		return []string{}
	}
	candidateSet := map[string]bool{}
	for _, workload := range workloads {
		for _, container := range workload.Containers {
			image := parseContainerImage(container.Image)
			if image.Registry == "" || strings.EqualFold(image.Registry, "docker.io") || strings.EqualFold(image.Registry, "registry-1.docker.io") {
				continue
			}
			if harbor.Projects[image.Project] {
				candidateSet[image.Registry] = true
			}
		}
	}
	candidates := make([]string, 0, len(candidateSet))
	for candidate := range candidateSet {
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return candidates
}

func inferRegistryCandidatesFromManagedServices(harbor harborScopeInfo, services []domain.ManagedService) []string {
	if len(harbor.Projects) == 0 {
		return []string{}
	}
	candidateSet := map[string]bool{}
	for _, service := range services {
		image := parseContainerImage(service.Image)
		if image.Registry == "" || strings.EqualFold(image.Registry, "docker.io") || strings.EqualFold(image.Registry, "registry-1.docker.io") {
			continue
		}
		if harbor.Projects[image.Project] {
			candidateSet[image.Registry] = true
		}
	}
	candidates := make([]string, 0, len(candidateSet))
	for candidate := range candidateSet {
		candidates = append(candidates, candidate)
	}
	sort.Strings(candidates)
	return candidates
}

func looksLikeRegistry(value string) bool {
	return strings.Contains(value, ".") || strings.Contains(value, ":") || value == "localhost"
}

func normalizedImageRegistry(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.TrimPrefix(strings.TrimPrefix(value, "http://"), "https://")
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	return strings.TrimRight(strings.ToLower(host), "/")
}

func sameRegistryHost(left string, right string) bool {
	left = normalizedImageRegistry(left)
	right = normalizedImageRegistry(right)
	if left == right {
		return true
	}
	leftHost, leftPort, leftErr := net.SplitHostPort(left)
	rightHost, rightPort, rightErr := net.SplitHostPort(right)
	if leftErr == nil && rightErr == nil {
		return leftHost == rightHost && leftPort == rightPort
	}
	return false
}

func stableDiscoveredServiceID(parts ...string) string {
	key := strings.Join(parts, "\x00")
	sum := sha1.Sum([]byte(key))
	return "svc-" + hex.EncodeToString(sum[:8])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) findProbeAgent(environmentID string) (domain.Agent, bool) {
	for _, item := range h.repo.ListAgents("") {
		if item.EnvironmentID == environmentID && item.Status == "ONLINE" && item.ClaimStatus == "CLAIMED" {
			return item, true
		}
	}
	return domain.Agent{}, false
}

func probePayloadFromEnvironment(environment domain.Environment) map[string]string {
	payload := map[string]string{
		"source":        "platform",
		"environmentId": environment.ID,
		"environment":   environment.Name,
	}
	for _, binding := range bindingsForProbe(environment) {
		switch binding.ResourceType {
		case "K8S":
			appendPayloadCSV(payload, "k8sNamespaces", binding.ScopeValue)
		case "HARBOR":
			appendPayloadCSV(payload, "harborProjects", binding.ScopeValue)
		}
	}
	return payload
}

func bindingsForProbe(environment domain.Environment) []domain.EnvironmentResourceBinding {
	runtimeBindings := []domain.EnvironmentResourceBinding{}
	if len(environment.Bindings) > 0 {
		for _, binding := range environment.Bindings {
			if binding.BindingRole == "RUNTIME_TARGET" {
				runtimeBindings = append(runtimeBindings, binding)
			}
		}
	}
	return runtimeBindings
}

func appendPayloadCSV(payload map[string]string, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	existing := splitCSVValues(payload[key])
	for _, item := range existing {
		if item == value {
			return
		}
	}
	existing = append(existing, value)
	payload[key] = strings.Join(existing, ",")
}

func splitCSVValues(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func (h *Handler) checkEnvironmentByCachedScopes(c *gin.Context, environment domain.Environment) {
	checkedAt := time.Now()
	status := "UNKNOWN"
	checks := h.cachedScopeChecks(environment)
	for _, check := range checks {
		if check.Status != "HEALTHY" {
			status = "DEGRADED"
			break
		}
	}
	if status == "UNKNOWN" && len(checks) > 0 {
		status = "HEALTHY"
	}
	updated, ok, _ := h.repo.UpdateEnvironmentCheck(environment.ID, status, checkedAt)
	if ok {
		status = updated.Status
	}
	OK(c, gin.H{
		"environmentId": environment.ID,
		"status":        status,
		"checkedAt":     checkedAt.Format(time.RFC3339),
		"checks":        checks,
	})
}

func (h *Handler) cachedScopeChecks(environment domain.Environment) []integration.IntegrationCheck {
	checks := []integration.IntegrationCheck{}
	bindings := environment.Bindings
	if len(bindings) == 0 {
		bindings = defaultEnvironmentBindingsForCheck(environment)
	}
	for _, binding := range bindings {
		if binding.BindingRole == "RUNTIME_TARGET" {
			continue
		}
		switch binding.ResourceType {
		case "K8S":
			if environment.Type != "LOCAL" {
				continue
			}
			cluster, exists := h.repo.GetKubernetesCluster(binding.ResourceID)
			checks = append(checks, cachedScopeCheck("K8s 命名空间", exists && containsTrimmedString(cluster.Namespaces, binding.ScopeValue), binding.ScopeValue))
		case "HARBOR":
			registry, exists := h.repo.GetHarborRegistry(binding.ResourceID)
			checks = append(checks, cachedScopeCheck("Harbor 镜像项目", exists && containsTrimmedString(registry.Projects, binding.ScopeValue), binding.ScopeValue))
		case "JENKINS":
			instance, exists := h.repo.GetJenkinsInstance(binding.ResourceID)
			checks = append(checks, cachedScopeCheck("Jenkins 流水线视图", exists && containsTrimmedString(instance.Views, binding.ScopeValue), binding.ScopeValue))
		}
	}
	return checks
}

func defaultEnvironmentBindingsForCheck(environment domain.Environment) []domain.EnvironmentResourceBinding {
	bindings := []domain.EnvironmentResourceBinding{}
	if strings.TrimSpace(environment.ClusterID) != "" || strings.TrimSpace(environment.Namespace) != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "K8S",
			ResourceID:   strings.TrimSpace(environment.ClusterID),
			ScopeType:    "NAMESPACE",
			ScopeValue:   strings.TrimSpace(environment.Namespace),
			BindingRole:  "BUILD_SOURCE",
			IsDefault:    true,
		})
	}
	if strings.TrimSpace(environment.RegistryID) != "" || strings.TrimSpace(environment.RegistryProject) != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "HARBOR",
			ResourceID:   strings.TrimSpace(environment.RegistryID),
			ScopeType:    "PROJECT",
			ScopeValue:   strings.TrimSpace(environment.RegistryProject),
			BindingRole:  "BUILD_SOURCE",
			IsDefault:    true,
		})
	}
	if strings.TrimSpace(environment.JenkinsID) != "" || strings.TrimSpace(environment.JenkinsView) != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "JENKINS",
			ResourceID:   strings.TrimSpace(environment.JenkinsID),
			ScopeType:    "VIEW",
			ScopeValue:   strings.TrimSpace(environment.JenkinsView),
			BindingRole:  "BUILD_SOURCE",
			IsDefault:    true,
		})
	}
	return bindings
}

func cachedScopeCheck(name string, exists bool, scope string) integration.IntegrationCheck {
	if exists {
		return integration.IntegrationCheck{Component: name, Status: "HEALTHY", Message: scope + " 已在最近探测结果中发现", CheckedAt: time.Now().Format(time.RFC3339)}
	}
	return integration.IntegrationCheck{Component: name, Status: "DEGRADED", Message: scope + " 未在最近探测结果中发现", CheckedAt: time.Now().Format(time.RFC3339)}
}

func containsTrimmedString(values []string, target string) bool {
	trimmedTarget := strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == trimmedTarget {
			return true
		}
	}
	return false
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
	environment.ClusterKubeconfig = cluster.Kubeconfig
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
	items := h.repo.ListKubernetesClusters(c.Query("keyword"))
	for index := range items {
		items[index] = resolveKubernetesClusterAPIServer(items[index])
	}
	OK(c, paginate(items, c))
}

func (h *Handler) CreateKubernetesCluster(c *gin.Context) {
	var request kubernetesClusterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	input := resolveKubernetesClusterAPIServer(domain.KubernetesCluster{
		ID:         request.ID,
		Name:       request.Name,
		APIServer:  request.APIServer,
		Context:    request.Context,
		Kubeconfig: request.Kubeconfig,
	})
	item, err := h.repo.CreateKubernetesCluster(input)
	if err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	Created(c, resolveKubernetesClusterAPIServer(item))
}

func (h *Handler) UpdateKubernetesCluster(c *gin.Context) {
	var request kubernetesClusterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	input := resolveKubernetesClusterAPIServer(domain.KubernetesCluster{
		ID:         request.ID,
		Name:       request.Name,
		APIServer:  request.APIServer,
		Context:    request.Context,
		Kubeconfig: request.Kubeconfig,
	})
	item, ok, err := h.repo.UpdateKubernetesCluster(c.Param("id"), input)
	if err != nil {
		BadRequest(c, "invalid kubernetes cluster request")
		return
	}
	if !ok {
		NotFound(c, "kubernetes cluster not found")
		return
	}
	OK(c, resolveKubernetesClusterAPIServer(item))
}

func resolveKubernetesClusterAPIServer(item domain.KubernetesCluster) domain.KubernetesCluster {
	if strings.TrimSpace(item.APIServer) != "" || strings.TrimSpace(item.Kubeconfig) == "" {
		return item
	}
	var parsed resourceKubeconfig
	if err := yaml.Unmarshal([]byte(item.Kubeconfig), &parsed); err != nil {
		return item
	}
	cluster, _, err := selectResourceKubeEntries(parsed, item.Context)
	if err != nil {
		return item
	}
	item.APIServer = strings.TrimSpace(cluster.Server)
	return item
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
		AgentID    string `json:"agentId"`
		TTLMinutes int    `json:"ttlMinutes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid register token request")
		return
	}
	if request.TTLMinutes <= 0 {
		request.TTLMinutes = 10
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	if request.AgentID == "" {
		request.AgentID = "agent-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	token, err := randomToken("agtr")
	if err != nil {
		BadRequest(c, "generate register token failed")
		return
	}
	expiresAt := time.Now().Add(time.Duration(request.TTLMinutes) * time.Minute)
	if !h.repo.CreateAgentRegisterToken(hashToken(token), request.AgentID, "", expiresAt) {
		BadRequest(c, "create register token failed")
		return
	}
	baseURL := requestBaseURL(c)
	lines := []string{
		"# 平台侧登记的 Agent 唯一标识。首次注册建议直接使用平台页面生成的值。",
		"AGENT_ID=" + request.AgentID,
		"",
		"# 首次注册时建议留空；在平台页面认领 Agent 后，Agent 会通过心跳同步绑定关系。",
		"AGENT_ENVIRONMENT_ID=",
		"",
		"# Agent 可出站访问的平台后端 API 地址。部署到项目环境前，请改成该机器可访问的平台地址。",
		"PLATFORM_URL=" + baseURL,
		"",
		"# 首次注册可留空；使用 -f 配置文件启动时，注册成功后 Agent 会自动写回平台签发的运行令牌。",
		"AGENT_TOKEN=",
		"",
		"# 一次性注册密钥，使用一次后失效。",
		"AGENT_REGISTER_TOKEN=" + token,
	}
	lines = append(lines,
		"",
		"AGENT_MODE=remote-probe",
		"AGENT_HEALTH_PORT=18080",
		"AGENT_POLL_INTERVAL_SECONDS=5",
		"AGENT_HEARTBEAT_INTERVAL_SECONDS=15",
		"AGENT_HTTP_TIMEOUT_SECONDS=10",
		"AGENT_MAX_TASKS=1",
		"AGENT_CAPABILITIES=remote-probe,k8s-api,http-check",
		"",
		"# 远程 Kubernetes 连接配置。Agent 通过 Kubernetes API 上报资源，namespace 与产品的对应关系在平台维护。",
		"AGENT_KUBECONFIG=",
		"",
		"# 远程 Harbor 连接配置。Agent 只负责上报 project、镜像和 tag，project 与产品的对应关系在平台维护。",
		"AGENT_HARBOR_URL=",
		"AGENT_HARBOR_USERNAME=",
		"AGENT_HARBOR_PASSWORD=",
		"AGENT_HARBOR_INSECURE_SKIP_TLS_VERIFY=false",
	)
	configText := strings.Join(lines, "\n")
	Created(c, gin.H{
		"agentId":        request.AgentID,
		"platformUrl":    baseURL,
		"token":          token,
		"expiresAt":      expiresAt.Format(time.RFC3339),
		"configText":     configText,
		"installCommand": configText,
	})
}

func (h *Handler) RegisterAgent(c *gin.Context) {
	var request struct {
		AgentID       string   `json:"agentId"`
		EnvironmentID string   `json:"environmentId"`
		RegisterToken string   `json:"registerToken"`
		Version       string   `json:"version"`
		Capabilities  []string `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid agent register request")
		return
	}
	request.AgentID = strings.TrimSpace(request.AgentID)
	request.EnvironmentID = strings.TrimSpace(request.EnvironmentID)
	request.RegisterToken = strings.TrimSpace(request.RegisterToken)
	if request.RegisterToken == "" {
		BadRequest(c, "register token is required")
		return
	}
	tokenAgentID, tokenEnvironmentID, ok := h.repo.ConsumeAgentRegisterToken(hashToken(request.RegisterToken), time.Now())
	if !ok {
		BadRequest(c, "register token is invalid, expired, or already used")
		return
	}
	if request.AgentID == "" {
		request.AgentID = tokenAgentID
	}
	if request.AgentID == "" {
		BadRequest(c, "agentId is required")
		return
	}
	if tokenAgentID != "" && request.AgentID != tokenAgentID {
		BadRequest(c, "agentId does not match register token")
		return
	}
	if tokenEnvironmentID != "" && request.EnvironmentID != "" && request.EnvironmentID != tokenEnvironmentID {
		BadRequest(c, "environmentId does not match register token")
		return
	}
	runtimeToken, err := randomToken("agt")
	if err != nil {
		BadRequest(c, "generate agent token failed")
		return
	}
	agentItem, ok := h.repo.RegisterAgent(request.AgentID, "", request.Version, request.Capabilities, hashToken(runtimeToken))
	if !ok {
		BadRequest(c, "register agent failed")
		return
	}
	Created(c, gin.H{
		"agent":      agentItem,
		"agentToken": runtimeToken,
	})
}

func (h *Handler) ClaimAgent(c *gin.Context) {
	var request struct {
		EnvironmentID string `json:"environmentId"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid claim request")
		return
	}
	request.EnvironmentID = strings.TrimSpace(request.EnvironmentID)
	if request.EnvironmentID == "" {
		BadRequest(c, "environmentId is required")
		return
	}
	environment, ok := h.repo.GetEnvironment(request.EnvironmentID)
	if !ok {
		BadRequest(c, "agent or environment not found")
		return
	}
	if environment.Type != "PROJECT" {
		BadRequest(c, "Agent 只能绑定远程产品，本地产品不需要绑定 Agent")
		return
	}
	agentItem, ok := h.repo.ClaimAgent(c.Param("id"), request.EnvironmentID)
	if !ok {
		BadRequest(c, "agent or environment not found")
		return
	}
	OK(c, agentItem)
}

func (h *Handler) AgentHeartbeat(c *gin.Context) {
	var request struct {
		EnvironmentID string               `json:"environmentId"`
		Version       string               `json:"version"`
		Capabilities  []string             `json:"capabilities"`
		RuntimeStatus domain.RuntimeStatus `json:"runtimeStatus"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid heartbeat request")
		return
	}
	agentID := c.Param("id")
	if !h.authorizeAgent(c, agentID) {
		return
	}
	request.EnvironmentID = strings.TrimSpace(request.EnvironmentID)
	if request.EnvironmentID != "" {
		if _, exists := h.repo.GetEnvironment(request.EnvironmentID); !exists {
			BadRequest(c, "environment not found")
			return
		}
	}
	currentAgent, ok := h.repo.GetAgent(agentID)
	if !ok {
		NotFound(c, "agent not found")
		return
	}
	if request.EnvironmentID != "" && currentAgent.EnvironmentID != "" && request.EnvironmentID != currentAgent.EnvironmentID {
		BadRequest(c, "agent environment does not match claimed environment")
		return
	}
	agentItem, ok := h.repo.UpdateAgentHeartbeat(agentID, request.EnvironmentID, request.Version, request.Capabilities, request.RuntimeStatus)
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
	if !h.authorizeAgent(c, c.Param("id")) {
		return
	}
	agentItem, ok := h.repo.GetAgent(c.Param("id"))
	if !ok {
		NotFound(c, "agent not found")
		return
	}
	if agentItem.Status != "ONLINE" {
		BadRequest(c, "agent must be ONLINE")
		return
	}
	if agentItem.ClaimStatus != "CLAIMED" || agentItem.EnvironmentID == "" {
		BadRequest(c, "agent is online but not claimed by an environment")
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
	if !h.authorizeAgent(c, request.AgentID) {
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
	if agentItem.ClaimStatus != "CLAIMED" || agentItem.EnvironmentID == "" {
		BadRequest(c, "agent is online but not claimed by an environment")
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
	leaseEnvironmentID := request.EnvironmentID
	if leaseEnvironmentID == "" {
		leaseEnvironmentID = agentItem.EnvironmentID
	}
	result := h.protocol.Lease(agent.LeaseRequest{
		AgentID:       agentItem.ID,
		EnvironmentID: leaseEnvironmentID,
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
	if !h.authorizeTaskAgent(c, c.Param("id")) {
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
	if !h.authorizeTaskAgent(c, c.Param("id")) {
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
	if !h.authorizeTaskAgent(c, c.Param("id")) {
		return
	}
	task, ok := h.protocol.ReportResult(c.Param("id"), request.Status, request.Message)
	if !ok {
		NotFound(c, "agent task not found")
		return
	}
	h.handleRemoteProbeResult(task, request.Status, request.Message)
	if task.AgentID != "" && (request.Status == "SUCCESS" || request.Status == "FAILED") {
		h.repo.AssignAgentTask(task.AgentID, "")
	}
	OK(c, task)
}

type remoteProbeResult struct {
	Status string                         `json:"status"`
	Checks []integration.IntegrationCheck `json:"checks"`
}

func (h *Handler) handleRemoteProbeResult(task agent.ProtocolTask, taskStatus string, message string) {
	if task.Type != "probe" || task.Action != "remote_resource_probe" || task.EnvironmentID == "" {
		return
	}
	checkedAt := time.Now()
	status := "UNHEALTHY"
	if taskStatus == "SUCCESS" {
		status = "HEALTHY"
	}
	var result remoteProbeResult
	if err := json.Unmarshal([]byte(message), &result); err == nil {
		if result.Status == "" {
			status = statusFromProbeChecks(result.Checks)
		} else {
			status = result.Status
		}
	}
	if taskStatus == "FAILED" && status == "HEALTHY" {
		status = "UNHEALTHY"
	}
	_, _, _ = h.repo.UpdateEnvironmentCheck(task.EnvironmentID, status, checkedAt)
}

func statusFromProbeChecks(checks []integration.IntegrationCheck) string {
	if len(checks) == 0 {
		return "UNKNOWN"
	}
	for _, check := range checks {
		if check.Status == "UNHEALTHY" {
			return "UNHEALTHY"
		}
	}
	for _, check := range checks {
		if check.Status != "HEALTHY" {
			return "DEGRADED"
		}
	}
	return "HEALTHY"
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
			payload := gin.H{
				"enabled": true,
				"status":  status,
				"logs":    logs,
			}
			if probe, ok := remoteProbeResultFromStatus(status, logs); ok {
				payload["probe"] = probe
			}
			OK(c, payload)
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
	payload := gin.H{
		"enabled": true,
		"status":  status,
		"logs":    logs,
	}
	if probe, ok := remoteProbeResultFromStatus(status, logs); ok {
		payload["probe"] = probe
	}
	OK(c, payload)
}

func remoteProbeResultFromStatus(status map[string]string, logs []string) (remoteProbeResult, bool) {
	if status["type"] != "probe" || status["action"] != "remote_resource_probe" {
		return remoteProbeResult{}, false
	}
	for i := len(logs) - 1; i >= 0; i-- {
		var result remoteProbeResult
		if err := json.Unmarshal([]byte(logs[i]), &result); err == nil && (result.Status != "" || len(result.Checks) > 0) {
			return result, true
		}
	}
	return remoteProbeResult{}, false
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

func (h *Handler) authorizeAgent(c *gin.Context, agentID string) bool {
	token := bearerToken(c)
	if token == "" || !h.repo.ValidateAgentToken(agentID, hashToken(token)) {
		c.JSON(http.StatusUnauthorized, Response{
			Code:      "UNAUTHORIZED",
			Message:   "invalid agent token",
			RequestID: requestID(),
		})
		return false
	}
	return true
}

func (h *Handler) authorizeTaskAgent(c *gin.Context, taskID string) bool {
	status, _, ok := h.protocol.Status(taskID)
	agentID := strings.TrimSpace(status["agentId"])
	if !ok || agentID == "" {
		NotFound(c, "agent task not found")
		return false
	}
	return h.authorizeAgent(c, agentID)
}

func bearerToken(c *gin.Context) string {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw == "" {
		return ""
	}
	prefix := "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(raw, prefix))
}

func randomToken(prefix string) (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(data[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
