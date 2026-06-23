package repository

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ops-release-platform/backend/internal/domain"
)

//go:embed mockdata/*.json
var mockFiles embed.FS

type MockRepository struct {
	projects        []domain.Project
	environments    []domain.Environment
	kubernetes      []domain.KubernetesCluster
	harbor          []domain.HarborRegistry
	jenkins         []domain.JenkinsInstance
	agents          []domain.Agent
	agentTokens     map[string]string
	registerTokens  map[string]mockAgentRegisterToken
	baselines       []domain.Baseline
	baselineDetails map[string]domain.BaselineDetail
	releases        []domain.ReleaseOrder
	releaseDetail   domain.ReleaseDetail
	deployTasks     []domain.DeployTask
	deployDetail    domain.DeployDetail
	currentUser     domain.CurrentUser
	users           []domain.User
	roles           []domain.Role
	permissions     []domain.EnvironmentPermission
	changelog       []domain.ChangelogEntry
}

type mockAgentRegisterToken struct {
	AgentID       string
	EnvironmentID string
	ExpiresAt     time.Time
	Used          bool
}

func NewMockRepository() (*MockRepository, error) {
	repo := &MockRepository{
		agentTokens:    map[string]string{},
		registerTokens: map[string]mockAgentRegisterToken{},
	}
	var baselineDetail domain.BaselineDetail
	loaders := []func() error{
		func() error { return loadJSON("mockdata/environments.json", &repo.environments) },
		func() error { return loadJSON("mockdata/agents.json", &repo.agents) },
		func() error { return loadJSON("mockdata/baselines.json", &repo.baselines) },
		func() error { return loadJSON("mockdata/baseline-detail.json", &baselineDetail) },
		func() error { return loadJSON("mockdata/releases.json", &repo.releases) },
		func() error { return loadJSON("mockdata/release-detail.json", &repo.releaseDetail) },
		func() error { return loadJSON("mockdata/deploy-tasks.json", &repo.deployTasks) },
		func() error { return loadJSON("mockdata/deploy-detail.json", &repo.deployDetail) },
		func() error { return loadJSON("mockdata/auth-me.json", &repo.currentUser) },
		func() error { return loadJSON("mockdata/users.json", &repo.users) },
		func() error { return loadJSON("mockdata/roles.json", &repo.roles) },
		func() error { return loadJSON("mockdata/permissions.json", &repo.permissions) },
		func() error { return loadJSON("mockdata/changelog.json", &repo.changelog) },
	}
	for _, load := range loaders {
		if err := load(); err != nil {
			return nil, err
		}
	}
	repo.bootstrapBaselineDetails(baselineDetail)
	for _, item := range repo.agents {
		repo.agentTokens[item.ID] = mockTokenHash(item.ID + "-test-token")
	}
	return repo, nil
}

func (r *MockRepository) ListProjects(query string) []domain.Project {
	items := filter(r.projects, query, func(item domain.Project) string {
		return item.ID + " " + item.Name + " " + item.Code + " " + item.Status
	})
	if len(items) == 0 && strings.TrimSpace(query) == "" {
		return []domain.Project{}
	}
	counts := r.productCountByProjectID()
	for index := range items {
		items[index].Status = normalizeProjectStatus(items[index].Status)
		items[index].ProductCount = counts[items[index].ID]
	}
	return items
}

func (r *MockRepository) GetProject(id string) (domain.Project, bool) {
	for _, item := range r.projects {
		if item.ID == id {
			item.Status = normalizeProjectStatus(item.Status)
			item.ProductCount = r.productCountByProjectID()[item.ID]
			return item, true
		}
	}
	return domain.Project{}, false
}

func (r *MockRepository) CreateProject(input domain.Project) (domain.Project, error) {
	item := domain.Project{
		ID:          strings.TrimSpace(input.ID),
		Name:        strings.TrimSpace(input.Name),
		Code:        normalizeEnvironmentCode(input.Code),
		Description: strings.TrimSpace(input.Description),
		Status:      normalizeProjectStatus(input.Status),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	if item.Code == "" {
		item.Code = generateEnvironmentCode(item.Name, "PROJECT")
	}
	if item.ID == "" && item.Code != "" {
		item.ID = "proj-" + item.Code
	}
	if item.ID == "" || item.Name == "" || item.Code == "" {
		return domain.Project{}, fmt.Errorf("missing required fields")
	}
	if _, exists := r.GetProject(item.ID); exists {
		return domain.Project{}, fmt.Errorf("project already exists")
	}
	r.projects = append(r.projects, item)
	return item, nil
}

func (r *MockRepository) UpdateProject(id string, input domain.Project) (domain.Project, bool, error) {
	for index := range r.projects {
		if r.projects[index].ID != id {
			continue
		}
		if value := strings.TrimSpace(input.Name); value != "" {
			r.projects[index].Name = value
		}
		if value := normalizeEnvironmentCode(input.Code); value != "" {
			r.projects[index].Code = value
		}
		r.projects[index].Description = strings.TrimSpace(input.Description)
		r.projects[index].Status = normalizeProjectStatus(input.Status)
		r.projects[index].ProductCount = r.productCountByProjectID()[r.projects[index].ID]
		return r.projects[index], true, nil
	}
	return domain.Project{}, false, nil
}

func mockTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func loadJSON(path string, dest any) error {
	content, err := mockFiles.ReadFile(path)
	if err != nil {
		return err
	}
	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(content, dest)
}

func (r *MockRepository) ListEnvironments(query string) []domain.Environment {
	items := filter(r.environments, query, func(item domain.Environment) string {
		return item.ID + " " + item.Name + " " + item.Code + " " + item.ProjectID + " " + item.ProjectName + " " + item.Type + " " + item.NetworkMode + " " + item.Status
	})
	return r.refreshEnvironmentStatuses(r.attachProjectInfo(items))
}

func (r *MockRepository) GetEnvironment(id string) (domain.Environment, bool) {
	item, ok := r.getEnvironment(id)
	if !ok {
		return domain.Environment{}, false
	}
	return r.refreshEnvironmentStatuses(r.attachProjectInfo([]domain.Environment{item}))[0], true
}

func (r *MockRepository) CreateEnvironment(input domain.Environment) (domain.Environment, error) {
	normalized, err := normalizeEnvironmentInput(input)
	if err != nil {
		return domain.Environment{}, err
	}
	id := strings.TrimSpace(normalized.ID)
	if id == "" && normalized.Code != "" {
		id = "env-" + normalized.Code
	}
	item := domain.Environment{
		ID:                  id,
		Name:                normalized.Name,
		Code:                normalized.Code,
		ProjectID:           normalized.ProjectID,
		ProjectName:         r.projectInfo(normalized.ProjectID).Name,
		ProductStatus:       normalizeProductStatusWithProject(normalized.ProductStatus, normalized.ProjectID, r.projectInfo(normalized.ProjectID)),
		Type:                normalized.Type,
		DeployTargetType:    normalized.DeployTargetType,
		NetworkMode:         normalized.NetworkMode,
		ClusterID:           normalized.ClusterID,
		Namespace:           normalized.Namespace,
		RegistryID:          normalized.RegistryID,
		RegistryProject:     normalized.RegistryProject,
		PrivateRegistryHost: normalized.PrivateRegistryHost,
		JenkinsID:           normalized.JenkinsID,
		JenkinsView:         normalized.JenkinsView,
		Status:              r.environmentStatusAfterDefinitionSave(normalized, firstNonEmpty(normalized.Status, "UNKNOWN")),
		AgentStatus:         "UNBOUND",
		LastCheckAt:         "",
		Bindings:            withEnvironmentID(normalized.Bindings, id),
	}
	if item.ID == "" || item.Name == "" || item.Code == "" || item.Type == "" || item.NetworkMode == "" {
		return domain.Environment{}, fmt.Errorf("missing required fields")
	}
	if _, exists := r.getEnvironment(item.ID); exists {
		return domain.Environment{}, fmt.Errorf("environment already exists")
	}
	r.environments = append(r.environments, item)
	return item, nil
}

func (r *MockRepository) UpdateEnvironment(id string, input domain.Environment) (domain.Environment, bool, error) {
	for index := range r.environments {
		if r.environments[index].ID != id {
			continue
		}
		if value := strings.TrimSpace(input.Name); value != "" {
			r.environments[index].Name = value
		}
		if value := strings.TrimSpace(input.Code); value != "" {
			r.environments[index].Code = value
		}
		r.environments[index].ProjectID = strings.TrimSpace(input.ProjectID)
		r.environments[index].ProductStatus = normalizeProductStatus(input.ProductStatus, r.environments[index].ProjectID)
		if value := strings.TrimSpace(input.Type); value != "" {
			r.environments[index].Type = value
		}
		if value := strings.TrimSpace(input.DeployTargetType); value != "" {
			r.environments[index].DeployTargetType = value
		}
		if value := strings.TrimSpace(input.NetworkMode); value != "" {
			r.environments[index].NetworkMode = value
		}
		r.environments[index].ClusterID = strings.TrimSpace(input.ClusterID)
		r.environments[index].Namespace = strings.TrimSpace(input.Namespace)
		r.environments[index].RegistryID = strings.TrimSpace(input.RegistryID)
		r.environments[index].RegistryProject = strings.TrimSpace(input.RegistryProject)
		r.environments[index].PrivateRegistryHost = strings.TrimSpace(input.PrivateRegistryHost)
		r.environments[index].JenkinsID = strings.TrimSpace(input.JenkinsID)
		r.environments[index].JenkinsView = strings.TrimSpace(input.JenkinsView)
		r.environments[index].Bindings = input.Bindings
		if value := strings.TrimSpace(input.Status); value != "" {
			r.environments[index].Status = value
		}
		normalized, err := normalizeEnvironmentInput(r.environments[index])
		if err != nil {
			return domain.Environment{}, true, err
		}
		r.environments[index].Type = normalized.Type
		r.environments[index].ProjectID = normalized.ProjectID
		project := r.projectInfo(normalized.ProjectID)
		r.environments[index].ProjectName = project.Name
		r.environments[index].ProductStatus = normalizeProductStatusWithProject(normalized.ProductStatus, normalized.ProjectID, project)
		r.environments[index].DeployTargetType = normalized.DeployTargetType
		r.environments[index].NetworkMode = normalized.NetworkMode
		r.environments[index].ClusterID = normalized.ClusterID
		r.environments[index].Namespace = normalized.Namespace
		r.environments[index].RegistryID = normalized.RegistryID
		r.environments[index].RegistryProject = normalized.RegistryProject
		r.environments[index].PrivateRegistryHost = normalized.PrivateRegistryHost
		r.environments[index].JenkinsID = normalized.JenkinsID
		r.environments[index].JenkinsView = normalized.JenkinsView
		r.environments[index].Status = r.environmentStatusAfterDefinitionSave(normalized, r.environments[index].Status)
		r.environments[index].Bindings = withEnvironmentID(normalized.Bindings, r.environments[index].ID)
		return r.environments[index], true, nil
	}
	return domain.Environment{}, false, nil
}

func (r *MockRepository) UpdateEnvironmentCheck(id string, status string, checkedAt time.Time) (domain.Environment, bool, error) {
	for index := range r.environments {
		if r.environments[index].ID != id {
			continue
		}
		r.environments[index].Status = firstNonEmpty(strings.TrimSpace(status), "UNKNOWN")
		r.environments[index].LastCheckAt = checkedAt.Format(time.RFC3339)
		return r.environments[index], true, nil
	}
	return domain.Environment{}, false, nil
}

func (r *MockRepository) environmentStatusByScopeCache(item domain.Environment, currentStatus string) string {
	if r.environmentHasUnverifiedScopes(item) {
		return "DEGRADED"
	}
	return firstNonEmpty(strings.TrimSpace(currentStatus), "UNKNOWN")
}

func (r *MockRepository) environmentStatusAfterDefinitionSave(item domain.Environment, currentStatus string) string {
	if r.environmentHasUnverifiedScopes(item) {
		return "DEGRADED"
	}
	return verifiedEnvironmentStatus(currentStatus)
}

func (r *MockRepository) environmentHasUnverifiedScopes(item domain.Environment) bool {
	bindings := item.Bindings
	if len(bindings) == 0 {
		bindings = defaultEnvironmentBindings(item)
	}
	for _, binding := range bindings {
		switch binding.ResourceType {
		case "K8S":
			if item.Type != "LOCAL" || binding.BindingRole == "RUNTIME_TARGET" {
				continue
			}
			cluster, exists := r.GetKubernetesCluster(binding.ResourceID)
			if !exists || !stringListContains(cluster.Namespaces, binding.ScopeValue) {
				return true
			}
		case "HARBOR":
			if binding.BindingRole == "RUNTIME_TARGET" {
				continue
			}
			registry, exists := r.GetHarborRegistry(binding.ResourceID)
			if !exists || !stringListContains(registry.Projects, binding.ScopeValue) {
				return true
			}
		case "JENKINS":
			if binding.BindingRole == "RUNTIME_TARGET" {
				continue
			}
			instance, exists := r.GetJenkinsInstance(binding.ResourceID)
			if !exists || !stringListContains(instance.Views, binding.ScopeValue) {
				return true
			}
		}
	}
	return false
}

func (r *MockRepository) refreshEnvironmentStatuses(items []domain.Environment) []domain.Environment {
	for index := range items {
		items[index].Status = r.environmentStatusByScopeCache(items[index], items[index].Status)
	}
	return items
}

func (r *MockRepository) attachProjectInfo(items []domain.Environment) []domain.Environment {
	for index := range items {
		project := r.projectInfo(items[index].ProjectID)
		items[index].ProjectName = project.Name
		items[index].ProductStatus = normalizeProductStatusWithProject(items[index].ProductStatus, items[index].ProjectID, project)
	}
	return items
}

func (r *MockRepository) projectInfo(projectID string) projectLookupInfo {
	if strings.TrimSpace(projectID) == "" {
		return projectLookupInfo{}
	}
	for _, project := range r.projects {
		if project.ID == projectID {
			return projectLookupInfo{Name: project.Name, Status: normalizeProjectStatus(project.Status), Found: true}
		}
	}
	return projectLookupInfo{}
}

func (r *MockRepository) productCountByProjectID() map[string]int {
	counts := map[string]int{}
	for _, environment := range r.environments {
		if environment.ProjectID != "" {
			counts[environment.ProjectID]++
		}
	}
	return counts
}

func (r *MockRepository) ListKubernetesClusters(query string) []domain.KubernetesCluster {
	return filter(r.kubernetes, query, func(item domain.KubernetesCluster) string {
		return item.ID + " " + item.Name + " " + item.APIServer + " " + item.Status
	})
}

func (r *MockRepository) GetKubernetesCluster(id string) (domain.KubernetesCluster, bool) {
	for _, item := range r.kubernetes {
		if item.ID == strings.TrimSpace(id) {
			return item, true
		}
	}
	return domain.KubernetesCluster{}, false
}

func (r *MockRepository) CreateKubernetesCluster(input domain.KubernetesCluster) (domain.KubernetesCluster, error) {
	item := domain.KubernetesCluster{
		ID:            strings.TrimSpace(input.ID),
		Name:          strings.TrimSpace(input.Name),
		APIServer:     strings.TrimSpace(input.APIServer),
		CredentialRef: strings.TrimSpace(input.CredentialRef),
		Kubeconfig:    strings.TrimSpace(input.Kubeconfig),
		Context:       strings.TrimSpace(input.Context),
		Status:        "UNKNOWN",
	}
	if item.ID == "" || item.Name == "" || (item.APIServer == "" && item.Kubeconfig == "") {
		return domain.KubernetesCluster{}, fmt.Errorf("missing required fields")
	}
	r.kubernetes = append(r.kubernetes, item)
	return item, nil
}

func (r *MockRepository) UpdateKubernetesCluster(id string, input domain.KubernetesCluster) (domain.KubernetesCluster, bool, error) {
	for index := range r.kubernetes {
		if r.kubernetes[index].ID != id {
			continue
		}
		if value := strings.TrimSpace(input.Name); value != "" {
			r.kubernetes[index].Name = value
		}
		if value := strings.TrimSpace(input.APIServer); value != "" {
			r.kubernetes[index].APIServer = value
		}
		if value := strings.TrimSpace(input.Kubeconfig); value != "" {
			r.kubernetes[index].Kubeconfig = value
		}
		r.kubernetes[index].Context = strings.TrimSpace(input.Context)
		if value := strings.TrimSpace(input.CredentialRef); value != "" {
			r.kubernetes[index].CredentialRef = value
		}
		return r.kubernetes[index], true, nil
	}
	return domain.KubernetesCluster{}, false, nil
}

func (r *MockRepository) UpdateKubernetesClusterProbe(id string, status string, message string, namespaces []string, checkedAt time.Time) (domain.KubernetesCluster, bool, error) {
	for index := range r.kubernetes {
		if r.kubernetes[index].ID != id {
			continue
		}
		r.kubernetes[index].Status = firstNonEmpty(strings.TrimSpace(status), "UNKNOWN")
		r.kubernetes[index].ProbeMessage = strings.TrimSpace(message)
		if namespaces != nil {
			r.kubernetes[index].Namespaces = append([]string(nil), namespaces...)
		}
		r.kubernetes[index].LastCheckAt = checkedAt.Format(time.RFC3339)
		return r.kubernetes[index], true, nil
	}
	return domain.KubernetesCluster{}, false, nil
}

func (r *MockRepository) ListHarborRegistries(query string) []domain.HarborRegistry {
	return filter(r.harbor, query, func(item domain.HarborRegistry) string {
		return item.ID + " " + item.Name + " " + item.URL + " " + item.Status
	})
}

func (r *MockRepository) GetHarborRegistry(id string) (domain.HarborRegistry, bool) {
	for _, item := range r.harbor {
		if item.ID == strings.TrimSpace(id) {
			return item, true
		}
	}
	return domain.HarborRegistry{}, false
}

func (r *MockRepository) CreateHarborRegistry(input domain.HarborRegistry) (domain.HarborRegistry, error) {
	item := domain.HarborRegistry{
		ID:                    strings.TrimSpace(input.ID),
		Name:                  strings.TrimSpace(input.Name),
		URL:                   strings.TrimSpace(input.URL),
		Scheme:                strings.TrimSpace(input.Scheme),
		Username:              strings.TrimSpace(input.Username),
		Password:              strings.TrimSpace(input.Password),
		CredentialRef:         strings.TrimSpace(input.CredentialRef),
		InsecureSkipTLSVerify: input.InsecureSkipTLSVerify,
		Status:                "UNKNOWN",
	}
	if item.ID == "" || item.Name == "" || item.URL == "" {
		return domain.HarborRegistry{}, fmt.Errorf("missing required fields")
	}
	r.harbor = append(r.harbor, item)
	return item, nil
}

func (r *MockRepository) UpdateHarborRegistry(id string, input domain.HarborRegistry) (domain.HarborRegistry, bool, error) {
	for index := range r.harbor {
		if r.harbor[index].ID != id {
			continue
		}
		if value := strings.TrimSpace(input.Name); value != "" {
			r.harbor[index].Name = value
		}
		if value := strings.TrimSpace(input.URL); value != "" {
			r.harbor[index].URL = value
		}
		if value := strings.TrimSpace(input.Username); value != "" {
			r.harbor[index].Username = value
		}
		if value := strings.TrimSpace(input.Password); value != "" {
			r.harbor[index].Password = value
		}
		if value := strings.TrimSpace(input.CredentialRef); value != "" {
			r.harbor[index].CredentialRef = value
		}
		r.harbor[index].Scheme = strings.TrimSpace(input.Scheme)
		r.harbor[index].InsecureSkipTLSVerify = input.InsecureSkipTLSVerify
		return r.harbor[index], true, nil
	}
	return domain.HarborRegistry{}, false, nil
}

func (r *MockRepository) UpdateHarborRegistryProbe(id string, status string, message string, projects []string, registryHost string, checkedAt time.Time) (domain.HarborRegistry, bool, error) {
	for index := range r.harbor {
		if r.harbor[index].ID != id {
			continue
		}
		r.harbor[index].Status = firstNonEmpty(strings.TrimSpace(status), "UNKNOWN")
		r.harbor[index].ProbeMessage = strings.TrimSpace(message)
		if projects != nil {
			r.harbor[index].Projects = append([]string(nil), projects...)
		}
		if value := strings.TrimSpace(registryHost); value != "" {
			r.harbor[index].RegistryHost = value
		}
		r.harbor[index].LastCheckAt = checkedAt.Format(time.RFC3339)
		return r.harbor[index], true, nil
	}
	return domain.HarborRegistry{}, false, nil
}

func (r *MockRepository) ListJenkinsInstances(query string) []domain.JenkinsInstance {
	return filter(r.jenkins, query, func(item domain.JenkinsInstance) string {
		return item.ID + " " + item.Name + " " + item.URL + " " + item.Status
	})
}

func (r *MockRepository) GetJenkinsInstance(id string) (domain.JenkinsInstance, bool) {
	for _, item := range r.jenkins {
		if item.ID == strings.TrimSpace(id) {
			return item, true
		}
	}
	return domain.JenkinsInstance{}, false
}

func (r *MockRepository) CreateJenkinsInstance(input domain.JenkinsInstance) (domain.JenkinsInstance, error) {
	item := domain.JenkinsInstance{
		ID:                    strings.TrimSpace(input.ID),
		Name:                  strings.TrimSpace(input.Name),
		URL:                   strings.TrimSpace(input.URL),
		Username:              strings.TrimSpace(input.Username),
		Token:                 strings.TrimSpace(input.Token),
		CredentialRef:         strings.TrimSpace(input.CredentialRef),
		InsecureSkipTLSVerify: input.InsecureSkipTLSVerify,
		Status:                "UNKNOWN",
	}
	if item.ID == "" || item.Name == "" || item.URL == "" {
		return domain.JenkinsInstance{}, fmt.Errorf("missing required fields")
	}
	r.jenkins = append(r.jenkins, item)
	return item, nil
}

func (r *MockRepository) UpdateJenkinsInstance(id string, input domain.JenkinsInstance) (domain.JenkinsInstance, bool, error) {
	for index := range r.jenkins {
		if r.jenkins[index].ID != id {
			continue
		}
		if value := strings.TrimSpace(input.Name); value != "" {
			r.jenkins[index].Name = value
		}
		if value := strings.TrimSpace(input.URL); value != "" {
			r.jenkins[index].URL = value
		}
		if value := strings.TrimSpace(input.Username); value != "" {
			r.jenkins[index].Username = value
		}
		if value := strings.TrimSpace(input.Token); value != "" {
			r.jenkins[index].Token = value
		}
		if value := strings.TrimSpace(input.CredentialRef); value != "" {
			r.jenkins[index].CredentialRef = value
		}
		r.jenkins[index].InsecureSkipTLSVerify = input.InsecureSkipTLSVerify
		return r.jenkins[index], true, nil
	}
	return domain.JenkinsInstance{}, false, nil
}

func (r *MockRepository) UpdateJenkinsInstanceProbe(id string, status string, message string, views []string, jobs []string, checkedAt time.Time) (domain.JenkinsInstance, bool, error) {
	for index := range r.jenkins {
		if r.jenkins[index].ID != id {
			continue
		}
		r.jenkins[index].Status = firstNonEmpty(strings.TrimSpace(status), "UNKNOWN")
		r.jenkins[index].ProbeMessage = strings.TrimSpace(message)
		if views != nil {
			r.jenkins[index].Views = append([]string(nil), views...)
		}
		if jobs != nil {
			r.jenkins[index].Jobs = append([]string(nil), jobs...)
		}
		r.jenkins[index].LastCheckAt = checkedAt.Format(time.RFC3339)
		return r.jenkins[index], true, nil
	}
	return domain.JenkinsInstance{}, false, nil
}

func (r *MockRepository) ListAgents(query string) []domain.Agent {
	items := filter(r.agents, query, func(item domain.Agent) string {
		return item.ID + " " + item.Name + " " + item.EnvironmentName + " " + strings.Join(item.Capabilities, " ") + " " + item.Status
	})
	for index := range items {
		if items[index].ClaimStatus == "" {
			items[index].ClaimStatus = "CLAIMED"
		}
	}
	return items
}

func (r *MockRepository) GetAgent(id string) (domain.Agent, bool) {
	for _, agent := range r.agents {
		if agent.ID == id {
			if agent.ClaimStatus == "" {
				agent.ClaimStatus = "CLAIMED"
			}
			return agent, true
		}
	}
	return domain.Agent{}, false
}

func (r *MockRepository) CreateAgentRegisterToken(tokenHash string, agentID string, environmentID string, expiresAt time.Time) bool {
	tokenHash = strings.TrimSpace(tokenHash)
	if tokenHash == "" {
		return false
	}
	if r.registerTokens == nil {
		r.registerTokens = map[string]mockAgentRegisterToken{}
	}
	r.registerTokens[tokenHash] = mockAgentRegisterToken{
		AgentID:       strings.TrimSpace(agentID),
		EnvironmentID: strings.TrimSpace(environmentID),
		ExpiresAt:     expiresAt,
	}
	return tokenHash != ""
}

func (r *MockRepository) ConsumeAgentRegisterToken(tokenHash string, now time.Time) (string, string, bool) {
	tokenHash = strings.TrimSpace(tokenHash)
	item, ok := r.registerTokens[tokenHash]
	if !ok || item.Used || item.ExpiresAt.Before(now) {
		return "", "", false
	}
	item.Used = true
	r.registerTokens[tokenHash] = item
	return item.AgentID, item.EnvironmentID, true
}

func (r *MockRepository) RegisterAgent(id string, environmentID string, version string, capabilities []string, tokenHash string) (domain.Agent, bool) {
	id = strings.TrimSpace(id)
	tokenHash = strings.TrimSpace(tokenHash)
	if id == "" || tokenHash == "" {
		return domain.Agent{}, false
	}
	if r.agentTokens == nil {
		r.agentTokens = map[string]string{}
	}
	r.agentTokens[id] = tokenHash
	for index := range r.agents {
		if r.agents[index].ID != id {
			continue
		}
		r.agents[index].Name = id
		r.agents[index].Status = "ONLINE"
		r.agents[index].LastHeartbeatAt = time.Now().Format(time.RFC3339)
		if version != "" {
			r.agents[index].Version = version
		}
		if len(capabilities) > 0 {
			r.agents[index].Capabilities = capabilities
		}
		if r.agents[index].ClaimStatus == "" {
			if r.agents[index].EnvironmentID != "" {
				r.agents[index].ClaimStatus = "CLAIMED"
			} else {
				r.agents[index].ClaimStatus = "PENDING_CLAIM"
			}
		}
		return r.agents[index], true
	}
	agent := domain.Agent{
		ID:              id,
		Name:            id,
		Version:         firstNonEmpty(version, "dev"),
		Status:          "ONLINE",
		ClaimStatus:     "PENDING_CLAIM",
		Capabilities:    append([]string(nil), capabilities...),
		LastHeartbeatAt: time.Now().Format(time.RFC3339),
	}
	r.agents = append(r.agents, agent)
	return agent, true
}

func (r *MockRepository) ValidateAgentToken(id string, tokenHash string) bool {
	id = strings.TrimSpace(id)
	tokenHash = strings.TrimSpace(tokenHash)
	if id == "" || tokenHash == "" {
		return false
	}
	stored, ok := r.agentTokens[id]
	return ok && stored == tokenHash
}

func (r *MockRepository) ClaimAgent(id string, environmentID string) (domain.Agent, bool) {
	agent, ok := r.UpsertAgent(id, environmentID, "", nil, "")
	if !ok {
		return domain.Agent{}, false
	}
	agent.ClaimStatus = "CLAIMED"
	for index := range r.agents {
		if r.agents[index].ID == id {
			r.agents[index].ClaimStatus = "CLAIMED"
			break
		}
	}
	return agent, true
}

func (r *MockRepository) UpsertAgent(id string, environmentID string, version string, capabilities []string, status string) (domain.Agent, bool) {
	for index := range r.agents {
		if r.agents[index].ID != id {
			continue
		}
		r.agents[index].EnvironmentID = environmentID
		r.agents[index].EnvironmentName = r.resolveEnvironmentName(environmentID)
		if version != "" {
			r.agents[index].Version = version
		}
		if len(capabilities) > 0 {
			r.agents[index].Capabilities = capabilities
		}
		if status != "" {
			r.agents[index].Status = status
		}
		if environmentID != "" {
			r.agents[index].ClaimStatus = "CLAIMED"
		} else if r.agents[index].ClaimStatus == "" {
			if r.agents[index].EnvironmentID != "" {
				r.agents[index].ClaimStatus = "CLAIMED"
			} else {
				r.agents[index].ClaimStatus = "PENDING_CLAIM"
			}
		}
		now := time.Now().Format(time.RFC3339)
		r.agents[index].LastHeartbeatAt = now
		return r.agents[index], true
	}
	agent := domain.Agent{
		ID:              id,
		Name:            id,
		EnvironmentID:   environmentID,
		EnvironmentName: r.resolveEnvironmentName(environmentID),
		Version:         firstNonEmpty(version, "dev"),
		Status:          firstNonEmpty(status, "ONLINE"),
		ClaimStatus:     firstNonEmpty(map[bool]string{true: "CLAIMED", false: "PENDING_CLAIM"}[environmentID != ""], "PENDING_CLAIM"),
		Capabilities:    append([]string(nil), capabilities...),
		LastHeartbeatAt: time.Now().Format(time.RFC3339),
	}
	r.agents = append(r.agents, agent)
	return agent, true
}

func (r *MockRepository) UpdateAgentHeartbeat(id string, environmentID string, version string, capabilities []string, runtimeStatus domain.RuntimeStatus) (domain.Agent, bool) {
	for index := range r.agents {
		if r.agents[index].ID != id {
			continue
		}
		r.agents[index].Status = "ONLINE"
		r.agents[index].LastHeartbeatAt = time.Now().Format(time.RFC3339)
		if environmentID != "" {
			if _, ok := r.getEnvironment(environmentID); !ok {
				return domain.Agent{}, false
			}
		}
		if version != "" {
			r.agents[index].Version = version
		}
		if len(capabilities) > 0 {
			r.agents[index].Capabilities = capabilities
		}
		if runtimeStatusHasData(runtimeStatus) {
			r.agents[index].RuntimeStatus = runtimeStatus
		}
		if r.agents[index].ClaimStatus == "" {
			if r.agents[index].EnvironmentID != "" {
				r.agents[index].ClaimStatus = "CLAIMED"
			} else {
				r.agents[index].ClaimStatus = "PENDING_CLAIM"
			}
		}
		return r.agents[index], true
	}
	return domain.Agent{}, false
}

func (r *MockRepository) AssignAgentTask(id string, taskID string) (domain.Agent, bool) {
	for index := range r.agents {
		if r.agents[index].ID != id {
			continue
		}
		if taskID == "" {
			r.agents[index].CurrentTaskID = nil
		} else {
			r.agents[index].CurrentTaskID = &taskID
		}
		return r.agents[index], true
	}
	return domain.Agent{}, false
}

func (r *MockRepository) ListBaselines(query string) []domain.Baseline {
	return filter(r.baselines, query, func(item domain.Baseline) string {
		return item.ID + " " + item.Name + " " + item.SourceEnvironmentName + " " + item.Purpose + " " + item.Status
	})
}

func (r *MockRepository) GetBaselineDetail(id string) (domain.BaselineDetail, bool) {
	if id == "" {
		for _, detail := range r.baselineDetails {
			return detail, true
		}
		return domain.BaselineDetail{}, false
	}
	detail, ok := r.baselineDetails[id]
	return detail, ok
}

func (r *MockRepository) GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool) {
	baseline, ok := r.GetBaselineDetail(id)
	if !ok {
		return domain.DiffResult{}, false
	}
	if targetEnvironmentID == "" {
		targetEnvironmentID = baseline.SourceEnvironmentID
	}
	if _, ok := r.getEnvironment(targetEnvironmentID); !ok {
		return domain.DiffResult{}, false
	}
	return buildDiffResult(baseline, targetEnvironmentID, buildTargetRuntimeSnapshot(targetEnvironmentID, baseline.Items)), true
}

func (r *MockRepository) ListReleases(query string) []domain.ReleaseOrder {
	return filter(r.releases, query, func(item domain.ReleaseOrder) string {
		return item.ID + " " + item.Type + " " + item.SourceBaselineID + " " + item.ReleaseSource + " " + item.BuildID + " " +
			item.ImageRepository + " " + item.ImageTag + " " + item.TargetEnvironmentName + " " + item.Status + " " + item.AgentName
	})
}

func (r *MockRepository) ListReleaseSourceServices(productID string, query string) []domain.ReleaseSourceService {
	services := make([]domain.ReleaseSourceService, 0)
	for _, detail := range r.baselineDetails {
		for _, item := range detail.Items {
			repository := strings.TrimSpace(item.ServiceName)
			if repository == "" {
				repository = item.ServiceID
			}
			services = append(services, domain.ReleaseSourceService{
				ServiceID:       item.ServiceID,
				ServiceName:     item.ServiceName,
				Namespace:       item.Namespace,
				WorkloadName:    item.WorkloadName,
				WorkloadType:    item.WorkloadType,
				ImageProject:    "library",
				ImageRepository: "library/" + repository,
				ImageSource:     "PRIVATE",
				Publishable:     false,
			})
		}
		break
	}
	return filter(services, query, func(item domain.ReleaseSourceService) string {
		return item.ServiceID + " " + item.ServiceName + " " + item.Namespace + " " + item.WorkloadName + " " + item.ImageRepository
	})
}

func (r *MockRepository) ListManagedServices(productID string) []domain.ManagedService {
	return []domain.ManagedService{}
}

func (r *MockRepository) UpsertManagedServices(productID string, services []domain.DiscoveredService) ([]domain.ManagedService, error) {
	return []domain.ManagedService{}, nil
}

func (r *MockRepository) RemoveManagedServices(productID string, serviceIDs []string) ([]domain.ManagedService, error) {
	return []domain.ManagedService{}, nil
}

func (r *MockRepository) ConfirmManagedServiceRegistry(productID string, registryHost string, harborProjects []string) ([]domain.ManagedService, error) {
	return []domain.ManagedService{}, nil
}

func (r *MockRepository) CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error) {
	order := domain.ReleaseOrder{
		ID:                    strings.TrimSpace(input.ID),
		Type:                  strings.TrimSpace(input.Type),
		SourceBaselineID:      strings.TrimSpace(input.SourceBaselineID),
		ReleaseSource:         strings.TrimSpace(input.ReleaseSource),
		ExecutionMode:         strings.TrimSpace(input.ExecutionMode),
		BuildID:               strings.TrimSpace(input.BuildID),
		BuildStatus:           strings.TrimSpace(input.BuildStatus),
		BuildURL:              strings.TrimSpace(input.BuildURL),
		ImageRepository:       strings.TrimSpace(input.ImageRepository),
		ImageTag:              strings.TrimSpace(input.ImageTag),
		ImageDigest:           strings.TrimSpace(input.ImageDigest),
		TargetEnvironmentName: r.resolveEnvironmentName(input.TargetEnvironmentID),
		Status:                firstNonEmpty(strings.TrimSpace(input.Status), "PENDING"),
		Progress:              input.Progress,
		AgentName:             input.AgentID,
	}
	if order.ID == "" || order.Type == "" || input.TargetEnvironmentID == "" {
		return domain.ReleaseOrder{}, fmt.Errorf("missing required fields")
	}
	if agent, ok := r.GetAgent(input.AgentID); ok {
		order.AgentName = firstNonEmpty(agent.Name, agent.ID)
	}
	r.releases = append([]domain.ReleaseOrder{order}, r.releases...)
	r.releaseDetail = domain.ReleaseDetail{
		ID:                    order.ID,
		Type:                  order.Type,
		SourceBaselineID:      order.SourceBaselineID,
		ReleaseSource:         order.ReleaseSource,
		ExecutionMode:         order.ExecutionMode,
		BuildID:               order.BuildID,
		BuildStatus:           order.BuildStatus,
		BuildURL:              order.BuildURL,
		ImageRepository:       order.ImageRepository,
		ImageTag:              order.ImageTag,
		ImageDigest:           order.ImageDigest,
		TargetEnvironmentName: order.TargetEnvironmentName,
		Status:                order.Status,
		Progress:              order.Progress,
		AgentName:             order.AgentName,
		AgentTaskID:           order.ID,
		Steps:                 []domain.ReleaseStep{},
		Failures:              []domain.ReleaseFailure{},
		ActionRecords:         []domain.ActionRecord{},
		Logs:                  []string{},
	}
	return order, nil
}

func (r *MockRepository) GetReleaseDetail(id string) (domain.ReleaseDetail, bool) {
	if id != "" && id != r.releaseDetail.ID {
		return domain.ReleaseDetail{}, false
	}
	return r.releaseDetail, true
}

func (r *MockRepository) ListDeployTasks(query string) []domain.DeployTask {
	return filter(r.deployTasks, query, func(item domain.DeployTask) string {
		return item.ID + " " + item.Type + " " + item.ProductName + " " + item.TargetEnvironmentName + " " +
			item.SourceBaselineID + " " + item.Source + " " + strings.Join(item.ServiceNames, " ") + " " +
			item.CurrentStep + " " + item.Status + " " + item.AgentName + " " + item.AgentTaskID + " " + item.NextAction
	})
}

func (r *MockRepository) GetDeployDetail(id string) (domain.DeployDetail, bool) {
	if id != "" && id != r.deployDetail.ID {
		return domain.DeployDetail{}, false
	}
	return r.deployDetail, true
}

func (r *MockRepository) GetCurrentUser() domain.CurrentUser {
	return r.currentUser
}

func (r *MockRepository) SetCurrentUserForTest(user domain.CurrentUser) {
	r.currentUser = user
}

func (r *MockRepository) ListUsers(query string) []domain.User {
	return filter(r.users, query, func(item domain.User) string {
		return item.ID + " " + item.Username + " " + item.DisplayName + " " + strings.Join(item.Roles, " ") + " " + item.Status
	})
}

func (r *MockRepository) ListRoles(query string) []domain.Role {
	return filter(r.roles, query, func(item domain.Role) string {
		return item.Code + " " + item.Name + " " + item.Description + " " + strings.Join(item.Permissions, " ")
	})
}

func (r *MockRepository) ListPermissions(query string) []domain.EnvironmentPermission {
	return filter(r.permissions, query, func(item domain.EnvironmentPermission) string {
		return item.EnvironmentID + " " + item.EnvironmentName + " " + item.RoleCode + " " + item.Scope + " " + strings.Join(item.Actions, " ")
	})
}

func (r *MockRepository) HasEnvironmentAction(environmentID string, action string) bool {
	userRoles := make(map[string]struct{}, len(r.currentUser.Roles))
	for _, role := range r.currentUser.Roles {
		userRoles[role] = struct{}{}
	}
	for _, permission := range r.permissions {
		if permission.EnvironmentID != environmentID && permission.Scope != "ALL" {
			continue
		}
		if _, ok := userRoles[permission.RoleCode]; !ok {
			continue
		}
		for _, allowedAction := range permission.Actions {
			if allowedAction == action || allowedAction == "write" {
				return true
			}
		}
	}
	return false
}

func (r *MockRepository) ListChangelog(query string) []domain.ChangelogEntry {
	return filter(r.changelog, query, func(item domain.ChangelogEntry) string {
		return item.ID + " " + item.Version + " " + item.Title + " " + item.Type + " " + item.Operator + " " +
			strings.Join(item.Features, " ") + " " + strings.Join(item.Fixes, " ") + " " + strings.Join(item.KnownIssues, " ")
	})
}

func filter[T any](items []T, query string, text func(T) string) []T {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return items
	}
	result := make([]T, 0)
	for _, item := range items {
		if strings.Contains(strings.ToLower(text(item)), q) {
			result = append(result, item)
		}
	}
	return result
}

func (r *MockRepository) CreateBaseline(sourceEnvironmentID, name, purpose string) (domain.BaselineDetail, error) {
	environment, ok := r.getEnvironment(sourceEnvironmentID)
	if !ok {
		return domain.BaselineDetail{}, fmt.Errorf("environment not found")
	}
	now := time.Now()
	baselineID := fmt.Sprintf("BL-%s-%04d", now.Format("20060102"), len(r.baselines)+1)
	items := buildRuntimeSnapshotItems(environment.Code)
	detail := domain.BaselineDetail{
		ID:                    baselineID,
		Name:                  name,
		SourceEnvironmentID:   environment.ID,
		SourceEnvironmentName: environment.Name,
		ServiceCount:          len(items),
		Status:                "DRAFT",
		CreatedBy:             r.currentUser.DisplayName,
		CreatedAt:             now.Format(time.RFC3339),
		Purpose:               purpose,
		SnapshotSource:        fmt.Sprintf("%s/%s", environment.Name, environment.Code),
		SnapshotCollectedAt:   now.Format(time.RFC3339),
		SnapshotMode:          "MOCK_RUNTIME",
		SnapshotTaskID:        fmt.Sprintf("snapshot-%s", strings.ToLower(baselineID)),
		Items:                 items,
	}
	baseline := domain.Baseline{
		ID:                    detail.ID,
		Name:                  detail.Name,
		SourceEnvironmentID:   detail.SourceEnvironmentID,
		SourceEnvironmentName: detail.SourceEnvironmentName,
		ServiceCount:          detail.ServiceCount,
		CreatedBy:             detail.CreatedBy,
		CreatedAt:             detail.CreatedAt,
		Status:                detail.Status,
		Purpose:               detail.Purpose,
		SnapshotSource:        detail.SnapshotSource,
		SnapshotCollectedAt:   detail.SnapshotCollectedAt,
		SnapshotMode:          detail.SnapshotMode,
	}
	r.baselines = append([]domain.Baseline{baseline}, r.baselines...)
	r.baselineDetails[detail.ID] = detail
	return detail, nil
}

func (r *MockRepository) LockBaseline(id string) (domain.BaselineDetail, bool) {
	detail, ok := r.baselineDetails[id]
	if !ok {
		return domain.BaselineDetail{}, false
	}
	if detail.Status != "LOCKED" {
		detail.Status = "LOCKED"
		detail.LockedAt = time.Now().Format(time.RFC3339)
		r.baselineDetails[id] = detail
	}
	for index := range r.baselines {
		if r.baselines[index].ID == id {
			r.baselines[index].Status = detail.Status
			r.baselines[index].LockedAt = detail.LockedAt
			break
		}
	}
	return detail, true
}

func (r *MockRepository) bootstrapBaselineDetails(seedDetail domain.BaselineDetail) {
	r.baselineDetails = make(map[string]domain.BaselineDetail, len(r.baselines))
	for index, baseline := range r.baselines {
		if baseline.SourceEnvironmentID == "" || baseline.SourceEnvironmentName == "" {
			if environment, ok := r.resolveBaselineEnvironment(baseline); ok {
				if baseline.SourceEnvironmentID == "" {
					baseline.SourceEnvironmentID = environment.ID
				}
				if baseline.SourceEnvironmentName == "" {
					baseline.SourceEnvironmentName = environment.Name
				}
			}
			r.baselines[index] = baseline
		}
		items := buildRuntimeSnapshotItems(baseline.ID)
		if seedDetail.ID == baseline.ID && len(seedDetail.Items) > 0 {
			items = seedDetail.Items
		}
		snapshotCollectedAt := baseline.SnapshotCollectedAt
		if snapshotCollectedAt == "" {
			snapshotCollectedAt = baseline.CreatedAt
		}
		snapshotSource := baseline.SnapshotSource
		if snapshotSource == "" {
			snapshotSource = baseline.SourceEnvironmentName
		}
		snapshotMode := baseline.SnapshotMode
		if snapshotMode == "" {
			snapshotMode = "MOCK_RUNTIME"
		}
		snapshotTaskID := fmt.Sprintf("snapshot-%s", strings.ToLower(baseline.ID))
		r.baselineDetails[baseline.ID] = domain.BaselineDetail{
			ID:                    baseline.ID,
			Name:                  baseline.Name,
			SourceEnvironmentID:   baseline.SourceEnvironmentID,
			SourceEnvironmentName: baseline.SourceEnvironmentName,
			ServiceCount:          baseline.ServiceCount,
			Status:                baseline.Status,
			CreatedBy:             baseline.CreatedBy,
			CreatedAt:             baseline.CreatedAt,
			Purpose:               baseline.Purpose,
			LockedAt:              baseline.LockedAt,
			SnapshotSource:        snapshotSource,
			SnapshotCollectedAt:   snapshotCollectedAt,
			SnapshotMode:          snapshotMode,
			SnapshotTaskID:        snapshotTaskID,
			Items:                 items,
		}
	}
}

func (r *MockRepository) getEnvironment(id string) (domain.Environment, bool) {
	for _, environment := range r.environments {
		if environment.ID == id {
			return environment, true
		}
	}
	return domain.Environment{}, false
}

func (r *MockRepository) resolveBaselineEnvironment(baseline domain.Baseline) (domain.Environment, bool) {
	if baseline.SourceEnvironmentID != "" {
		return r.getEnvironment(baseline.SourceEnvironmentID)
	}
	for _, environment := range r.environments {
		if baseline.SourceEnvironmentName != "" && environment.Name == baseline.SourceEnvironmentName {
			return environment, true
		}
	}
	name := strings.ToLower(baseline.Name)
	switch {
	case strings.Contains(name, "local-prod"):
		return r.getEnvironment("env-local-prod")
	case strings.Contains(name, "project-x"):
		return r.getEnvironment("env-project-x-prod")
	case strings.Contains(name, "project-z"):
		return r.getEnvironment("env-project-z-prod")
	}
	return domain.Environment{}, false
}

func (r *MockRepository) resolveEnvironmentName(id string) string {
	if environment, ok := r.getEnvironment(id); ok {
		return environment.Name
	}
	return id
}

func buildRuntimeSnapshotItems(seed string) []domain.BaselineItem {
	prefix := sanitizeSeed(seed)
	return []domain.BaselineItem{
		{
			ServiceID:     prefix + "-gateway",
			ServiceName:   prefix + "-gateway",
			Namespace:     "core-system",
			WorkloadName:  prefix + "-gateway",
			WorkloadType:  "DEPLOYMENT",
			Tag:           "20260608-a1b2c3",
			Digest:        "sha256:8f21aa09",
			Replicas:      3,
			ReadyReplicas: 3,
			HealthStatus:  "HEALTHY",
		},
		{
			ServiceID:     prefix + "-order",
			ServiceName:   prefix + "-order",
			Namespace:     "biz-service",
			WorkloadName:  prefix + "-order",
			WorkloadType:  "DEPLOYMENT",
			Tag:           "20260608-d4e5f6",
			Digest:        "sha256:901b1220",
			Replicas:      2,
			ReadyReplicas: 2,
			HealthStatus:  "HEALTHY",
		},
		{
			ServiceID:     prefix + "-web",
			ServiceName:   prefix + "-web",
			Namespace:     "frontend",
			WorkloadName:  prefix + "-web",
			WorkloadType:  "DEPLOYMENT",
			Tag:           "20260608-77aa11",
			Digest:        "sha256:b0fd91ef",
			Replicas:      2,
			ReadyReplicas: 1,
			HealthStatus:  "DEGRADED",
		},
	}
}

func buildTargetRuntimeSnapshot(targetEnvironmentID string, baselineItems []domain.BaselineItem) []domain.BaselineItem {
	targetItems := make([]domain.BaselineItem, 0, len(baselineItems))
	for index, item := range baselineItems {
		switch index % 4 {
		case 0:
			item.Tag = item.Tag + "-hotfix"
			item.Digest = item.Digest + "99"
		case 1:
			// keep target consistent with baseline
		case 2:
			continue
		case 3:
			item.ReadyReplicas = max(0, item.ReadyReplicas-1)
			item.HealthStatus = "DEGRADED"
		}
		targetItems = append(targetItems, item)
	}
	if len(targetItems) == 0 {
		targetItems = append(targetItems, buildRuntimeSnapshotItems(targetEnvironmentID)[0])
	}
	return targetItems
}

func buildDiffResult(baseline domain.BaselineDetail, targetEnvironmentID string, targetItems []domain.BaselineItem) domain.DiffResult {
	targetByServiceID := make(map[string]domain.BaselineItem, len(targetItems))
	for _, item := range targetItems {
		targetByServiceID[item.ServiceID] = item
	}

	result := domain.DiffResult{
		SourceBaselineID:    baseline.ID,
		TargetEnvironmentID: targetEnvironmentID,
		Items:               make([]domain.DiffItem, 0, len(baseline.Items)),
	}

	for _, sourceItem := range baseline.Items {
		diffItem := domain.DiffItem{
			ServiceID:   sourceItem.ServiceID,
			ServiceName: sourceItem.ServiceName,
			Namespace:   sourceItem.Namespace,
			SourceTag:   sourceItem.Tag,
		}
		targetItem, ok := targetByServiceID[sourceItem.ServiceID]
		switch {
		case !ok:
			diffItem.DiffStatus = "MISSING_IN_TARGET"
			diffItem.Publishable = true
			diffItem.Strategy = "确认后新增部署"
			result.Summary.MissingInTarget++
			result.Summary.Publishable++
		case targetItem.HealthStatus != "HEALTHY" || targetItem.ReadyReplicas < targetItem.Replicas:
			targetTag := targetItem.Tag
			diffItem.TargetTag = &targetTag
			diffItem.DiffStatus = "WORKLOAD_ERROR"
			diffItem.Publishable = false
			diffItem.Strategy = "先修复 workload"
			result.Summary.WorkloadError++
		case targetItem.Tag != sourceItem.Tag:
			targetTag := targetItem.Tag
			diffItem.TargetTag = &targetTag
			diffItem.DiffStatus = "NEED_UPDATE"
			diffItem.Publishable = true
			diffItem.Strategy = "同步镜像并更新 tag"
			result.Summary.NeedUpdate++
			result.Summary.Publishable++
		default:
			targetTag := targetItem.Tag
			diffItem.TargetTag = &targetTag
			diffItem.DiffStatus = "CONSISTENT"
			diffItem.Publishable = false
			diffItem.Strategy = "无需处理"
			result.Summary.Consistent++
		}
		result.Items = append(result.Items, diffItem)
	}

	return result
}

func sanitizeSeed(seed string) string {
	replacer := strings.NewReplacer("env-", "", "BL-", "baseline-", "_", "-", "/", "-", " ", "-")
	value := strings.Trim(replacer.Replace(strings.ToLower(seed)), "-")
	if value == "" {
		return "runtime"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
