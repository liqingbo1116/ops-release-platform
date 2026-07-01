package repository

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"ops-release-platform/backend/internal/domain"
)

type DatabaseStore struct {
	db *gorm.DB
}

const agentHeartbeatTimeout = 60 * time.Second

type projectLookupInfo struct {
	Name   string
	Status string
	Found  bool
}

func NewDatabaseStore(db *gorm.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

func (s *DatabaseStore) ListProjects(query string) []domain.Project {
	var models []ProjectModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR code ILIKE ? OR status ILIKE ?", like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.Project{}
	}
	counts := s.productCountByProjectID()
	items := make([]domain.Project, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainProject(model, counts[model.ID]))
	}
	return items
}

func (s *DatabaseStore) GetProject(id string) (domain.Project, bool) {
	var model ProjectModel
	if err := s.db.Where("id = ?", strings.TrimSpace(id)).Take(&model).Error; err != nil {
		return domain.Project{}, false
	}
	counts := s.productCountByProjectID()
	return toDomainProject(model, counts[model.ID]), true
}

func (s *DatabaseStore) CreateProject(input domain.Project) (domain.Project, error) {
	model := ProjectModel{
		ID:          strings.TrimSpace(input.ID),
		Name:        strings.TrimSpace(input.Name),
		Code:        normalizeEnvironmentCode(input.Code),
		Description: strings.TrimSpace(input.Description),
		Status:      normalizeProjectStatus(input.Status),
	}
	if model.Code == "" {
		model.Code = generateEnvironmentCode(model.Name, "PROJECT")
	}
	if model.ID == "" && model.Code != "" {
		model.ID = "proj-" + model.Code
	}
	if model.ID == "" || model.Name == "" || model.Code == "" {
		return domain.Project{}, errors.New("missing required fields")
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.Project{}, err
	}
	return toDomainProject(model, 0), nil
}

func (s *DatabaseStore) UpdateProject(id string, input domain.Project) (domain.Project, bool, error) {
	var model ProjectModel
	if err := s.db.Where("id = ?", strings.TrimSpace(id)).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		model.Name = name
	}
	if code := normalizeEnvironmentCode(input.Code); code != "" {
		model.Code = code
	}
	model.Description = strings.TrimSpace(input.Description)
	model.Status = normalizeProjectStatus(input.Status)
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Project{}, false, err
	}
	counts := s.productCountByProjectID()
	return toDomainProject(model, counts[model.ID]), true, nil
}

func (s *DatabaseStore) ListEnvironments(query string) []domain.Environment {
	var models []EnvironmentModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR code ILIKE ? OR type ILIKE ? OR status ILIKE ? OR project_id ILIKE ?", like, like, like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.Environment{}
	}
	agentStatuses := s.agentStatusByIDMap()
	projects := s.projectInfoByIDMap()
	items := make([]domain.Environment, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainEnvironment(model, agentStatuses[model.AgentID], projects[model.ProjectID]))
	}
	return s.refreshEnvironmentStatuses(s.attachEnvironmentBindings(items))
}

func (s *DatabaseStore) GetEnvironment(id string) (domain.Environment, bool) {
	var model EnvironmentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Environment{}, false
	}
	item := toDomainEnvironment(model, s.getAgentStatus(model.AgentID), s.getProjectInfo(model.ProjectID))
	items := s.attachEnvironmentBindings([]domain.Environment{item})
	items = s.refreshEnvironmentStatuses(items)
	return items[0], true
}

func (s *DatabaseStore) CreateEnvironment(input domain.Environment) (domain.Environment, error) {
	normalized, err := normalizeEnvironmentInput(input)
	if err != nil {
		return domain.Environment{}, err
	}
	id := strings.TrimSpace(input.ID)
	if id == "" && normalized.Code != "" {
		id = "env-" + normalized.Code
	}
	model := EnvironmentModel{
		ID:                  id,
		Name:                strings.TrimSpace(normalized.Name),
		Code:                normalized.Code,
		ProjectID:           normalized.ProjectID,
		ProductStatus:       normalizeProductStatus(normalized.ProductStatus, normalized.ProjectID),
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
		Status:              fallbackString(strings.TrimSpace(normalized.Status), "UNKNOWN"),
	}
	model.Status = s.environmentStatusByScopeCache(normalized, model.Status)
	if model.ID == "" || model.Name == "" || model.Code == "" || model.Type == "" || model.NetworkMode == "" {
		return domain.Environment{}, errors.New("missing required fields")
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return replaceEnvironmentBindings(tx, model.ID, normalized.Bindings)
	}); err != nil {
		return domain.Environment{}, err
	}
	item := toDomainEnvironment(model, "", s.getProjectInfo(model.ProjectID))
	item.Bindings = withEnvironmentID(normalized.Bindings, model.ID)
	return item, nil
}

func (s *DatabaseStore) UpdateEnvironment(id string, input domain.Environment) (domain.Environment, bool, error) {
	var model EnvironmentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.Environment{}, false, nil
		}
		return domain.Environment{}, false, err
	}
	if name := strings.TrimSpace(input.Name); name != "" {
		model.Name = name
	}
	if code := normalizeEnvironmentCode(input.Code); code != "" {
		model.Code = code
	}
	model.ProjectID = strings.TrimSpace(input.ProjectID)
	model.ProductStatus = normalizeProductStatus(input.ProductStatus, model.ProjectID)
	if envType := strings.TrimSpace(input.Type); envType != "" {
		model.Type = envType
	}
	if deployTargetType := strings.TrimSpace(input.DeployTargetType); deployTargetType != "" {
		model.DeployTargetType = deployTargetType
	}
	if networkMode := strings.TrimSpace(input.NetworkMode); networkMode != "" {
		model.NetworkMode = networkMode
	}
	model.ClusterID = strings.TrimSpace(input.ClusterID)
	model.Namespace = strings.TrimSpace(input.Namespace)
	model.RegistryID = strings.TrimSpace(input.RegistryID)
	model.RegistryProject = strings.TrimSpace(input.RegistryProject)
	model.PrivateRegistryHost = strings.TrimSpace(input.PrivateRegistryHost)
	model.JenkinsID = strings.TrimSpace(input.JenkinsID)
	model.JenkinsView = strings.TrimSpace(input.JenkinsView)
	if status := strings.TrimSpace(input.Status); status != "" {
		model.Status = status
	}
	normalized, err := normalizeEnvironmentInput(domain.Environment{
		ID:                  model.ID,
		Name:                model.Name,
		Code:                model.Code,
		ProjectID:           model.ProjectID,
		ProductStatus:       model.ProductStatus,
		Type:                model.Type,
		DeployTargetType:    model.DeployTargetType,
		NetworkMode:         model.NetworkMode,
		ClusterID:           model.ClusterID,
		Namespace:           model.Namespace,
		RegistryID:          model.RegistryID,
		RegistryProject:     model.RegistryProject,
		PrivateRegistryHost: model.PrivateRegistryHost,
		JenkinsID:           model.JenkinsID,
		JenkinsView:         model.JenkinsView,
		Bindings:            input.Bindings,
		Status:              model.Status,
	})
	if err != nil {
		return domain.Environment{}, false, err
	}
	model.Type = normalized.Type
	model.ProjectID = normalized.ProjectID
	model.ProductStatus = normalizeProductStatus(normalized.ProductStatus, normalized.ProjectID)
	model.DeployTargetType = normalized.DeployTargetType
	model.NetworkMode = normalized.NetworkMode
	model.ClusterID = normalized.ClusterID
	model.Namespace = normalized.Namespace
	model.RegistryID = normalized.RegistryID
	model.RegistryProject = normalized.RegistryProject
	model.PrivateRegistryHost = normalized.PrivateRegistryHost
	model.JenkinsID = normalized.JenkinsID
	model.JenkinsView = normalized.JenkinsView
	model.Status = s.environmentStatusByScopeCache(normalized, model.Status)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		return replaceEnvironmentBindings(tx, model.ID, normalized.Bindings)
	}); err != nil {
		return domain.Environment{}, false, err
	}
	item := toDomainEnvironment(model, s.getAgentStatus(model.AgentID), s.getProjectInfo(model.ProjectID))
	item.Bindings = withEnvironmentID(normalized.Bindings, model.ID)
	return item, true, nil
}

func (s *DatabaseStore) UpdateEnvironmentCheck(id string, status string, checkedAt time.Time) (domain.Environment, bool, error) {
	var model EnvironmentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.Environment{}, false, nil
		}
		return domain.Environment{}, false, err
	}
	model.Status = fallbackString(strings.TrimSpace(status), "UNKNOWN")
	model.LastCheckAt = &checkedAt
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Environment{}, false, err
	}
	item := toDomainEnvironment(model, s.getAgentStatus(model.AgentID), s.getProjectInfo(model.ProjectID))
	items := s.attachEnvironmentBindings([]domain.Environment{item})
	items = s.refreshEnvironmentStatuses(items)
	return items[0], true, nil
}

func (s *DatabaseStore) ListKubernetesClusters(query string) []domain.KubernetesCluster {
	var models []KubernetesClusterModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR api_server ILIKE ? OR status ILIKE ?", like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.KubernetesCluster{}
	}
	items := make([]domain.KubernetesCluster, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainKubernetesCluster(model))
	}
	return items
}

func (s *DatabaseStore) GetKubernetesCluster(id string) (domain.KubernetesCluster, bool) {
	var model KubernetesClusterModel
	if err := s.db.Where("id = ?", strings.TrimSpace(id)).Take(&model).Error; err != nil {
		return domain.KubernetesCluster{}, false
	}
	return toDomainKubernetesCluster(model), true
}

func (s *DatabaseStore) CreateKubernetesCluster(input domain.KubernetesCluster) (domain.KubernetesCluster, error) {
	model := KubernetesClusterModel{
		ID:            strings.TrimSpace(input.ID),
		Name:          strings.TrimSpace(input.Name),
		APIServer:     strings.TrimSpace(input.APIServer),
		CredentialRef: fallbackString(strings.TrimSpace(input.CredentialRef), "resource:"+strings.TrimSpace(input.ID)),
		Kubeconfig:    strings.TrimSpace(input.Kubeconfig),
		Context:       strings.TrimSpace(input.Context),
		Status:        "UNKNOWN",
	}
	if model.ID == "" || model.Name == "" || (model.APIServer == "" && model.Kubeconfig == "") {
		return domain.KubernetesCluster{}, errors.New("missing required fields")
	}
	if model.CredentialRef == "resource:" {
		model.CredentialRef = "resource:" + model.ID
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.KubernetesCluster{}, err
	}
	return toDomainKubernetesCluster(model), nil
}

func (s *DatabaseStore) UpdateKubernetesCluster(id string, input domain.KubernetesCluster) (domain.KubernetesCluster, bool, error) {
	var model KubernetesClusterModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.KubernetesCluster{}, false, nil
		}
		return domain.KubernetesCluster{}, false, err
	}
	if value := strings.TrimSpace(input.Name); value != "" {
		model.Name = value
	}
	if value := strings.TrimSpace(input.APIServer); value != "" {
		model.APIServer = value
	}
	if value := strings.TrimSpace(input.Kubeconfig); value != "" {
		model.Kubeconfig = value
	}
	if value := strings.TrimSpace(input.Context); value != "" || strings.TrimSpace(input.Kubeconfig) != "" {
		model.Context = value
	}
	if value := strings.TrimSpace(input.CredentialRef); value != "" {
		model.CredentialRef = value
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.KubernetesCluster{}, false, err
	}
	return toDomainKubernetesCluster(model), true, nil
}

func (s *DatabaseStore) UpdateKubernetesClusterProbe(id string, status string, message string, namespaces []string, checkedAt time.Time) (domain.KubernetesCluster, bool, error) {
	var model KubernetesClusterModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.KubernetesCluster{}, false, nil
		}
		return domain.KubernetesCluster{}, false, err
	}
	model.Status = fallbackString(strings.TrimSpace(status), "UNKNOWN")
	model.ProbeMessage = strings.TrimSpace(message)
	if namespaces != nil {
		model.Namespaces = compactStringList(namespaces)
	}
	model.LastCheckAt = &checkedAt
	if err := s.db.Save(&model).Error; err != nil {
		return domain.KubernetesCluster{}, false, err
	}
	return toDomainKubernetesCluster(model), true, nil
}

func (s *DatabaseStore) ListHarborRegistries(query string) []domain.HarborRegistry {
	var models []HarborRegistryModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR url ILIKE ? OR status ILIKE ?", like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.HarborRegistry{}
	}
	items := make([]domain.HarborRegistry, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainHarborRegistry(model))
	}
	return items
}

func (s *DatabaseStore) GetHarborRegistry(id string) (domain.HarborRegistry, bool) {
	var model HarborRegistryModel
	if err := s.db.Where("id = ?", strings.TrimSpace(id)).Take(&model).Error; err != nil {
		return domain.HarborRegistry{}, false
	}
	return toDomainHarborRegistry(model), true
}

func (s *DatabaseStore) CreateHarborRegistry(input domain.HarborRegistry) (domain.HarborRegistry, error) {
	model := HarborRegistryModel{
		ID:                    strings.TrimSpace(input.ID),
		Name:                  strings.TrimSpace(input.Name),
		URL:                   strings.TrimSpace(input.URL),
		Scheme:                normalizeScheme(input.Scheme, input.URL),
		Username:              strings.TrimSpace(input.Username),
		Password:              strings.TrimSpace(input.Password),
		CredentialRef:         fallbackString(strings.TrimSpace(input.CredentialRef), "resource:"+strings.TrimSpace(input.ID)),
		InsecureSkipTLSVerify: input.InsecureSkipTLSVerify,
		Status:                "UNKNOWN",
	}
	if model.ID == "" || model.Name == "" || model.URL == "" {
		return domain.HarborRegistry{}, errors.New("missing required fields")
	}
	if model.CredentialRef == "resource:" {
		model.CredentialRef = "resource:" + model.ID
	}
	model.URL = normalizeResourceURL(model.URL, model.Scheme)
	if err := s.db.Create(&model).Error; err != nil {
		return domain.HarborRegistry{}, err
	}
	return toDomainHarborRegistry(model), nil
}

func (s *DatabaseStore) UpdateHarborRegistry(id string, input domain.HarborRegistry) (domain.HarborRegistry, bool, error) {
	var model HarborRegistryModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.HarborRegistry{}, false, nil
		}
		return domain.HarborRegistry{}, false, err
	}
	if value := strings.TrimSpace(input.Name); value != "" {
		model.Name = value
	}
	if value := strings.TrimSpace(input.URL); value != "" {
		model.URL = value
	}
	if value := normalizeScheme(input.Scheme, model.URL); value != "" {
		model.Scheme = value
	}
	model.URL = normalizeResourceURL(model.URL, model.Scheme)
	if value := strings.TrimSpace(input.Username); value != "" {
		model.Username = value
	}
	if value := strings.TrimSpace(input.Password); value != "" {
		model.Password = value
	}
	if value := strings.TrimSpace(input.CredentialRef); value != "" {
		model.CredentialRef = value
	}
	model.InsecureSkipTLSVerify = input.InsecureSkipTLSVerify
	if err := s.db.Save(&model).Error; err != nil {
		return domain.HarborRegistry{}, false, err
	}
	return toDomainHarborRegistry(model), true, nil
}

func (s *DatabaseStore) UpdateHarborRegistryProbe(id string, status string, message string, projects []string, registryHost string, checkedAt time.Time) (domain.HarborRegistry, bool, error) {
	var model HarborRegistryModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.HarborRegistry{}, false, nil
		}
		return domain.HarborRegistry{}, false, err
	}
	model.Status = fallbackString(strings.TrimSpace(status), "UNKNOWN")
	model.ProbeMessage = strings.TrimSpace(message)
	if projects != nil {
		model.Projects = compactStringList(projects)
	}
	if value := strings.TrimSpace(registryHost); value != "" {
		model.RegistryHost = value
	}
	model.LastCheckAt = &checkedAt
	if err := s.db.Save(&model).Error; err != nil {
		return domain.HarborRegistry{}, false, err
	}
	return toDomainHarborRegistry(model), true, nil
}

func (s *DatabaseStore) ListJenkinsInstances(query string) []domain.JenkinsInstance {
	var models []JenkinsInstanceModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR url ILIKE ? OR status ILIKE ?", like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.JenkinsInstance{}
	}
	items := make([]domain.JenkinsInstance, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainJenkinsInstance(model))
	}
	return items
}

func (s *DatabaseStore) GetJenkinsInstance(id string) (domain.JenkinsInstance, bool) {
	var model JenkinsInstanceModel
	if err := s.db.Where("id = ?", strings.TrimSpace(id)).Take(&model).Error; err != nil {
		return domain.JenkinsInstance{}, false
	}
	return toDomainJenkinsInstance(model), true
}

func (s *DatabaseStore) CreateJenkinsInstance(input domain.JenkinsInstance) (domain.JenkinsInstance, error) {
	model := JenkinsInstanceModel{
		ID:                    strings.TrimSpace(input.ID),
		Name:                  strings.TrimSpace(input.Name),
		URL:                   normalizeResourceURL(strings.TrimSpace(input.URL), "https"),
		Username:              strings.TrimSpace(input.Username),
		Token:                 strings.TrimSpace(input.Token),
		CredentialRef:         fallbackString(strings.TrimSpace(input.CredentialRef), "resource:"+strings.TrimSpace(input.ID)),
		InsecureSkipTLSVerify: input.InsecureSkipTLSVerify,
		Status:                "UNKNOWN",
	}
	if model.ID == "" || model.Name == "" || model.URL == "" {
		return domain.JenkinsInstance{}, errors.New("missing required fields")
	}
	if model.CredentialRef == "resource:" {
		model.CredentialRef = "resource:" + model.ID
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.JenkinsInstance{}, err
	}
	return toDomainJenkinsInstance(model), nil
}

func (s *DatabaseStore) UpdateJenkinsInstance(id string, input domain.JenkinsInstance) (domain.JenkinsInstance, bool, error) {
	var model JenkinsInstanceModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.JenkinsInstance{}, false, nil
		}
		return domain.JenkinsInstance{}, false, err
	}
	if value := strings.TrimSpace(input.Name); value != "" {
		model.Name = value
	}
	if value := strings.TrimSpace(input.URL); value != "" {
		model.URL = normalizeResourceURL(value, "https")
	}
	if value := strings.TrimSpace(input.Username); value != "" {
		model.Username = value
	}
	if value := strings.TrimSpace(input.Token); value != "" {
		model.Token = value
	}
	if value := strings.TrimSpace(input.CredentialRef); value != "" {
		model.CredentialRef = value
	}
	model.InsecureSkipTLSVerify = input.InsecureSkipTLSVerify
	if err := s.db.Save(&model).Error; err != nil {
		return domain.JenkinsInstance{}, false, err
	}
	return toDomainJenkinsInstance(model), true, nil
}

func (s *DatabaseStore) UpdateJenkinsInstanceProbe(id string, status string, message string, views []string, jobs []string, pipelines []domain.JenkinsPipeline, checkedAt time.Time) (domain.JenkinsInstance, bool, error) {
	var model JenkinsInstanceModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.JenkinsInstance{}, false, nil
		}
		return domain.JenkinsInstance{}, false, err
	}
	model.Status = fallbackString(strings.TrimSpace(status), "UNKNOWN")
	model.ProbeMessage = strings.TrimSpace(message)
	if views != nil {
		model.Views = compactStringList(views)
	}
	if jobs != nil {
		model.Jobs = compactStringList(jobs)
	}
	if pipelines != nil {
		model.Pipelines = compactJenkinsPipelines(pipelines)
	}
	model.LastCheckAt = &checkedAt
	if err := s.db.Save(&model).Error; err != nil {
		return domain.JenkinsInstance{}, false, err
	}
	return toDomainJenkinsInstance(model), true, nil
}

func (s *DatabaseStore) ListAgents(query string) []domain.Agent {
	var models []AgentModel
	db := s.db.Model(&AgentModel{}).Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where(
			"id ILIKE ? OR name ILIKE ? OR status ILIKE ?",
			like, like, like,
		)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.Agent{}
	}
	environmentNames := s.environmentNameMap()
	items := make([]domain.Agent, 0, len(models))
	for _, model := range models {
		environmentName := environmentNames[model.EnvironmentID]
		if trimmed := strings.TrimSpace(query); trimmed != "" {
			haystack := strings.ToLower(model.ID + " " + model.Name + " " + environmentName + " " + strings.Join(model.Capabilities, " ") + " " + model.Status)
			if !strings.Contains(haystack, strings.ToLower(trimmed)) {
				continue
			}
		}
		items = append(items, toDomainAgent(model, environmentName))
	}
	return items
}

func (s *DatabaseStore) GetAgent(id string) (domain.Agent, bool) {
	type row struct {
		AgentModel
		EnvironmentName string
	}
	var result row
	err := s.db.Table("agents").
		Select("agents.*, environments.name AS environment_name").
		Joins("LEFT JOIN environments ON environments.id = agents.environment_id").
		Where("agents.id = ?", id).
		Take(&result).Error
	if err != nil {
		return domain.Agent{}, false
	}
	return toDomainAgent(result.AgentModel, result.EnvironmentName), true
}

func (s *DatabaseStore) CreateAgentRegisterToken(tokenHash string, agentID string, environmentID string, expiresAt time.Time) bool {
	model := AgentRegisterTokenModel{
		ID:            "agtok-" + shortHash(tokenHash+time.Now().String()),
		TokenHash:     tokenHash,
		AgentID:       strings.TrimSpace(agentID),
		EnvironmentID: strings.TrimSpace(environmentID),
		ExpiresAt:     expiresAt,
	}
	return s.db.Create(&model).Error == nil
}

func (s *DatabaseStore) ConsumeAgentRegisterToken(tokenHash string, now time.Time) (string, string, bool) {
	var model AgentRegisterTokenModel
	err := s.db.Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).Take(&model).Error
	if err != nil {
		return "", "", false
	}
	usedAt := now
	if err := s.db.Model(&model).Update("used_at", &usedAt).Error; err != nil {
		return "", "", false
	}
	return model.AgentID, model.EnvironmentID, true
}

func (s *DatabaseStore) RegisterAgent(id string, environmentID string, version string, capabilities []string, tokenHash string) (domain.Agent, bool) {
	id = strings.TrimSpace(id)
	if id == "" || tokenHash == "" {
		return domain.Agent{}, false
	}
	now := time.Now()
	var model AgentModel
	err := s.db.Where("id = ?", id).Take(&model).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return domain.Agent{}, false
	}
	if err == gorm.ErrRecordNotFound {
		model = AgentModel{
			ID:              id,
			Name:            id,
			Version:         fallbackString(version, "dev"),
			Status:          "ONLINE",
			ClaimStatus:     "PENDING_CLAIM",
			TokenHash:       tokenHash,
			Capabilities:    append([]string(nil), capabilities...),
			LastHeartbeatAt: &now,
		}
		if createErr := s.db.Create(&model).Error; createErr != nil {
			return domain.Agent{}, false
		}
		return s.GetAgent(id)
	}

	model.Name = id
	model.TokenHash = tokenHash
	if model.ClaimStatus == "" {
		model.ClaimStatus = "PENDING_CLAIM"
	}
	model.Status = "ONLINE"
	model.LastHeartbeatAt = &now
	if version != "" {
		model.Version = version
	}
	if len(capabilities) > 0 {
		model.Capabilities = append([]string(nil), capabilities...)
	}
	if saveErr := s.db.Save(&model).Error; saveErr != nil {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) ValidateAgentToken(id string, tokenHash string) bool {
	if id == "" || tokenHash == "" {
		return false
	}
	var count int64
	if err := s.db.Model(&AgentModel{}).Where("id = ? AND token_hash = ?", id, tokenHash).Count(&count).Error; err != nil {
		return false
	}
	return count == 1
}

func (s *DatabaseStore) ClaimAgent(id string, environmentID string) (domain.Agent, bool) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(environmentID) == "" {
		return domain.Agent{}, false
	}
	if _, exists := s.GetEnvironment(environmentID); !exists {
		return domain.Agent{}, false
	}
	var model AgentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	previousEnvironmentID := model.EnvironmentID
	model.EnvironmentID = environmentID
	model.ClaimStatus = "CLAIMED"
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	if err := s.rebindEnvironmentAgent(previousEnvironmentID, environmentID, id); err != nil {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) UpsertAgent(id string, environmentID string, version string, capabilities []string, status string) (domain.Agent, bool) {
	if environmentID != "" {
		var environment EnvironmentModel
		if err := s.db.Where("id = ?", environmentID).Take(&environment).Error; err != nil {
			return domain.Agent{}, false
		}
	}

	now := time.Now()
	var model AgentModel
	err := s.db.Where("id = ?", id).Take(&model).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return domain.Agent{}, false
	}

	if err == gorm.ErrRecordNotFound {
		model = AgentModel{
			ID:              id,
			Name:            id,
			EnvironmentID:   environmentID,
			Version:         fallbackString(version, "dev"),
			Status:          fallbackString(status, "ONLINE"),
			ClaimStatus:     "PENDING_CLAIM",
			Capabilities:    append([]string(nil), capabilities...),
			LastHeartbeatAt: &now,
		}
		if createErr := s.db.Create(&model).Error; createErr != nil {
			return domain.Agent{}, false
		}
		if updateEnvErr := s.rebindEnvironmentAgent("", environmentID, id); updateEnvErr != nil {
			return domain.Agent{}, false
		}
		return s.GetAgent(id)
	}

	previousEnvironmentID := model.EnvironmentID
	model.EnvironmentID = environmentID
	model.Name = id
	if version != "" {
		model.Version = version
	}
	if len(capabilities) > 0 {
		model.Capabilities = append([]string(nil), capabilities...)
	}
	if status != "" {
		model.Status = status
	}
	if model.ClaimStatus == "" {
		model.ClaimStatus = "PENDING_CLAIM"
	}
	model.LastHeartbeatAt = &now
	if saveErr := s.db.Save(&model).Error; saveErr != nil {
		return domain.Agent{}, false
	}
	if updateEnvErr := s.rebindEnvironmentAgent(previousEnvironmentID, environmentID, id); updateEnvErr != nil {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) UpdateAgentHeartbeat(id string, environmentID string, version string, capabilities []string, runtimeStatus domain.RuntimeStatus) (domain.Agent, bool) {
	var model AgentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	now := time.Now()
	model.Status = "ONLINE"
	model.LastHeartbeatAt = &now
	if environmentID != "" {
		if _, exists := s.GetEnvironment(environmentID); !exists {
			return domain.Agent{}, false
		}
	}
	if version != "" {
		model.Version = version
	}
	if len(capabilities) > 0 {
		model.Capabilities = append([]string(nil), capabilities...)
	}
	if runtimeStatusHasData(runtimeStatus) {
		model.RuntimeStatus = runtimeStatus
	}
	if model.ClaimStatus == "" {
		model.ClaimStatus = "PENDING_CLAIM"
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) rebindEnvironmentAgent(previousEnvironmentID string, nextEnvironmentID string, agentID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if previousEnvironmentID != "" && previousEnvironmentID != nextEnvironmentID {
			if err := tx.Model(&EnvironmentModel{}).
				Where("id = ? AND agent_id = ?", previousEnvironmentID, agentID).
				Update("agent_id", "").Error; err != nil {
				return err
			}
		}
		if nextEnvironmentID != "" {
			if err := tx.Model(&EnvironmentModel{}).
				Where("id = ?", nextEnvironmentID).
				Update("agent_id", agentID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DatabaseStore) AssignAgentTask(id string, taskID string) (domain.Agent, bool) {
	updates := map[string]any{}
	if taskID == "" {
		updates["current_task_id"] = nil
	} else {
		updates["current_task_id"] = taskID
	}
	result := s.db.Model(&AgentModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) ListOperationLogs(query string, environmentID string, resourceType string) []domain.OperationLog {
	var models []OperationLogModel
	db := s.db.Order("created_at desc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR action ILIKE ? OR resource_type ILIKE ? OR resource_id ILIKE ? OR resource_name ILIKE ? OR project_name ILIKE ? OR product_name ILIKE ? OR namespace ILIKE ? OR workload_name ILIKE ? OR container_name ILIKE ? OR result ILIKE ? OR detail ILIKE ? OR operator_name ILIKE ?", like, like, like, like, like, like, like, like, like, like, like, like, like)
	}
	if trimmed := strings.TrimSpace(environmentID); trimmed != "" {
		db = db.Where("environment_id = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(resourceType); trimmed != "" {
		db = db.Where("resource_type = ?", strings.ToUpper(trimmed))
	}
	if err := db.Limit(500).Find(&models).Error; err != nil {
		return []domain.OperationLog{}
	}
	items := make([]domain.OperationLog, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainOperationLog(model))
	}
	return items
}

func (s *DatabaseStore) CreateOperationLog(input domain.OperationLog) (domain.OperationLog, error) {
	now := time.Now()
	model := OperationLogModel{
		ID:            strings.TrimSpace(input.ID),
		OperatorID:    firstNonEmpty(strings.TrimSpace(input.OperatorID), "system"),
		OperatorName:  firstNonEmpty(strings.TrimSpace(input.OperatorName), "平台"),
		Action:        strings.ToUpper(strings.TrimSpace(input.Action)),
		ResourceType:  strings.ToUpper(strings.TrimSpace(input.ResourceType)),
		ResourceID:    strings.TrimSpace(input.ResourceID),
		ResourceName:  strings.TrimSpace(input.ResourceName),
		ProjectID:     strings.TrimSpace(input.ProjectID),
		ProjectName:   strings.TrimSpace(input.ProjectName),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
		ProductName:   strings.TrimSpace(input.ProductName),
		TaskID:        strings.TrimSpace(input.TaskID),
		Namespace:     strings.TrimSpace(input.Namespace),
		WorkloadType:  strings.TrimSpace(input.WorkloadType),
		WorkloadName:  strings.TrimSpace(input.WorkloadName),
		ContainerName: strings.TrimSpace(input.ContainerName),
		ContainerType: strings.TrimSpace(input.ContainerType),
		Result:        strings.ToUpper(firstNonEmpty(strings.TrimSpace(input.Result), "SUCCESS")),
		Detail:        strings.TrimSpace(input.Detail),
		CreatedAt:     now,
	}
	if model.ID == "" {
		model.ID = fmt.Sprintf("oplog-%s", now.Format("20060102150405.000000000"))
	}
	if model.Action == "" || model.ResourceType == "" || model.ResourceID == "" {
		return domain.OperationLog{}, errors.New("missing required operation log fields")
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.OperationLog{}, err
	}
	return toDomainOperationLog(model), nil
}

func (s *DatabaseStore) GetCurrentUser() domain.CurrentUser {
	return domain.CurrentUser{
		ID:          "dev-admin",
		Username:    "admin",
		DisplayName: "管理员",
		Roles:       []string{"admin"},
		Permissions: []string{"*"},
	}
}

func (s *DatabaseStore) ListUsers(query string) []domain.User {
	return []domain.User{}
}

func (s *DatabaseStore) ListRoles(query string) []domain.Role {
	return []domain.Role{}
}

func (s *DatabaseStore) ListPermissions(query string) []domain.EnvironmentPermission {
	return []domain.EnvironmentPermission{}
}

func (s *DatabaseStore) ListChangelog(query string) []domain.ChangelogEntry {
	return []domain.ChangelogEntry{}
}

func (s *DatabaseStore) CreateBaseline(sourceEnvironmentID string, name string, purpose string) (domain.BaselineDetail, error) {
	sourceEnvironmentID = strings.TrimSpace(sourceEnvironmentID)
	if sourceEnvironmentID == "" {
		return domain.BaselineDetail{}, errors.New("source environment id is required")
	}
	environment, ok := s.GetEnvironment(sourceEnvironmentID)
	if !ok {
		return domain.BaselineDetail{}, errors.New("source environment not found")
	}
	var services []ServiceModel
	if err := s.db.Where("product_id = ?", sourceEnvironmentID).
		Order("namespace asc, workload_type asc, workload_name asc, container_type asc, container_name asc").
		Find(&services).Error; err != nil {
		return domain.BaselineDetail{}, err
	}
	now := time.Now()
	var count int64
	if err := s.db.Model(&EnvironmentBaselineModel{}).
		Where("created_at >= ? AND created_at < ?", now.Format("2006-01-02"), now.AddDate(0, 0, 1).Format("2006-01-02")).
		Count(&count).Error; err != nil {
		return domain.BaselineDetail{}, err
	}
	model := EnvironmentBaselineModel{
		ID:                  fmt.Sprintf("BL-%s-%04d", now.Format("20060102"), count+1),
		Name:                fallbackString(strings.TrimSpace(name), environment.Name+" 快照"),
		SourceEnvironmentID: sourceEnvironmentID,
		ProductID:           sourceEnvironmentID,
		ServiceCount:        len(services),
		Status:              "DRAFT",
		Purpose:             strings.TrimSpace(purpose),
		CreatedBy:           "admin",
		CreatedAt:           now,
	}
	items := make([]BaselineServiceItemModel, 0, len(services))
	for _, service := range services {
		items = append(items, baselineItemModelFromService(model.ID, service))
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.BaselineDetail{}, err
	}
	detail, ok := s.GetBaselineDetail(model.ID)
	if !ok {
		return domain.BaselineDetail{}, errors.New("baseline not found after create")
	}
	return detail, nil
}

func (s *DatabaseStore) ListBaselines(query string) []domain.Baseline {
	type row struct {
		EnvironmentBaselineModel
		SourceEnvironmentName string
	}
	var rows []row
	db := s.db.Table("environment_baselines").
		Select("environment_baselines.*, environments.name AS source_environment_name").
		Joins("LEFT JOIN environments ON environments.id = environment_baselines.source_environment_id").
		Order("environment_baselines.created_at desc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("environment_baselines.id ILIKE ? OR environment_baselines.name ILIKE ? OR environments.name ILIKE ?", like, like, like)
	}
	if err := db.Find(&rows).Error; err != nil {
		return []domain.Baseline{}
	}
	items := make([]domain.Baseline, 0, len(rows))
	for _, row := range rows {
		items = append(items, toDomainBaseline(row.EnvironmentBaselineModel, row.SourceEnvironmentName))
	}
	return items
}

func (s *DatabaseStore) GetBaselineDetail(id string) (domain.BaselineDetail, bool) {
	type row struct {
		EnvironmentBaselineModel
		SourceEnvironmentName string
	}
	var result row
	err := s.db.Table("environment_baselines").
		Select("environment_baselines.*, environments.name AS source_environment_name").
		Joins("LEFT JOIN environments ON environments.id = environment_baselines.source_environment_id").
		Where("environment_baselines.id = ?", strings.TrimSpace(id)).
		Take(&result).Error
	if err != nil {
		return domain.BaselineDetail{}, false
	}
	var itemModels []BaselineServiceItemModel
	if err := s.db.Where("baseline_id = ?", result.ID).
		Order("namespace asc, workload_type asc, workload_name asc").
		Find(&itemModels).Error; err != nil {
		return domain.BaselineDetail{}, false
	}
	return toDomainBaselineDetail(result.EnvironmentBaselineModel, result.SourceEnvironmentName, itemModels), true
}

func (s *DatabaseStore) LockBaseline(id string) (domain.BaselineDetail, bool) {
	trimmedID := strings.TrimSpace(id)
	now := time.Now()
	result := s.db.Model(&EnvironmentBaselineModel{}).Where("id = ?", trimmedID).Updates(map[string]any{
		"status":    "LOCKED",
		"locked_at": &now,
	})
	if result.Error != nil || result.RowsAffected == 0 {
		return domain.BaselineDetail{}, false
	}
	return s.GetBaselineDetail(trimmedID)
}

func (s *DatabaseStore) GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool) {
	detail, ok := s.GetBaselineDetail(id)
	if !ok {
		return domain.DiffResult{}, false
	}
	targetEnvironmentID = strings.TrimSpace(targetEnvironmentID)
	if _, ok := s.GetEnvironment(targetEnvironmentID); !ok {
		return domain.DiffResult{}, false
	}
	targetServices := s.ListManagedServices(targetEnvironmentID)
	targetByServiceID := make(map[string]domain.ManagedService, len(targetServices))
	for _, service := range targetServices {
		targetByServiceID[service.ID] = service
	}
	result := domain.DiffResult{
		SourceBaselineID:    detail.ID,
		TargetEnvironmentID: targetEnvironmentID,
		Items:               make([]domain.DiffItem, 0, len(detail.Items)),
	}
	for _, item := range detail.Items {
		diff := domain.DiffItem{
			ServiceID:   item.ServiceID,
			ServiceName: item.ServiceName,
			Namespace:   item.Namespace,
			SourceTag:   item.Tag,
		}
		target, exists := targetByServiceID[item.ServiceID]
		if !exists {
			diff.DiffStatus = "MISSING_IN_TARGET"
			diff.Publishable = true
			diff.Strategy = "CREATE_OR_DEPLOY"
			result.Summary.MissingInTarget++
			result.Summary.Publishable++
		} else if target.ImageTag != item.Tag {
			targetTag := target.ImageTag
			diff.TargetTag = &targetTag
			diff.DiffStatus = "NEED_UPDATE"
			diff.Publishable = true
			diff.Strategy = "ROLLING_UPDATE"
			result.Summary.NeedUpdate++
			result.Summary.Publishable++
		} else {
			targetTag := target.ImageTag
			diff.TargetTag = &targetTag
			diff.DiffStatus = "CONSISTENT"
			diff.Publishable = false
			diff.Strategy = "NOOP"
			result.Summary.Consistent++
		}
		result.Items = append(result.Items, diff)
	}
	sort.SliceStable(result.Items, func(left int, right int) bool {
		priority := map[string]int{"NEED_UPDATE": 0, "CONSISTENT": 1, "MISSING_IN_TARGET": 2, "WORKLOAD_ERROR": 3}
		leftPriority := priority[result.Items[left].DiffStatus]
		rightPriority := priority[result.Items[right].DiffStatus]
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return result.Items[left].ServiceID < result.Items[right].ServiceID
	})
	return result, true
}

func (s *DatabaseStore) ListReleaseSourceServices(productID string, query string) []domain.ReleaseSourceService {
	var models []ServiceModel
	db := s.db.Where("product_id = ?", strings.TrimSpace(productID)).Order("name asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR namespace ILIKE ? OR workload_name ILIKE ? OR image_repository ILIKE ?", like, like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.ReleaseSourceService{}
	}
	items := make([]domain.ReleaseSourceService, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainReleaseSourceService(model))
	}
	return items
}

func (s *DatabaseStore) ListManagedServices(productID string) []domain.ManagedService {
	var models []ServiceModel
	db := s.db.Where("product_id = ?", strings.TrimSpace(productID)).Order("namespace asc, workload_type asc, workload_name asc, container_type asc, container_name asc")
	if err := db.Find(&models).Error; err != nil {
		return []domain.ManagedService{}
	}
	items := make([]domain.ManagedService, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainManagedService(model))
	}
	return items
}

func (s *DatabaseStore) UpsertManagedServices(productID string, services []domain.DiscoveredService) ([]domain.ManagedService, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, errors.New("product id is required")
	}
	if _, ok := s.GetEnvironment(productID); !ok {
		return nil, errors.New("product not found")
	}
	models := make([]ServiceModel, 0, len(services))
	for _, service := range services {
		model := serviceModelFromDiscovered(productID, service)
		if model.ID == "" || model.Namespace == "" || model.WorkloadName == "" || model.ContainerName == "" || model.Image == "" {
			continue
		}
		models = append(models, model)
	}
	if len(models) == 0 {
		return []domain.ManagedService{}, nil
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, model := range models {
			var existing ServiceModel
			err := tx.Where("id = ? AND product_id = ?", model.ID, model.ProductID).Take(&existing).Error
			if err == nil {
				model.CreatedAt = existing.CreatedAt
				if err := tx.Model(&existing).Updates(map[string]any{
					"name":                       model.Name,
					"namespace":                  model.Namespace,
					"workload_name":              model.WorkloadName,
					"workload_type":              model.WorkloadType,
					"container_name":             model.ContainerName,
					"container_type":             model.ContainerType,
					"image":                      model.Image,
					"image_registry":             model.ImageRegistry,
					"image_project":              model.ImageProject,
					"image_repository":           model.ImageRepository,
					"image_tag":                  model.ImageTag,
					"image_source":               model.ImageSource,
					"private_registry_host":      model.PrivateRegistryHost,
					"private_registry_confirmed": model.PrivateRegistryConfirmed,
					"replicas":                   model.Replicas,
					"ready_replicas":             model.ReadyReplicas,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			var conflicting ServiceModel
			err = tx.Where("id = ?", model.ID).Take(&conflicting).Error
			if err == nil && conflicting.ProductID != model.ProductID {
				model.ID = stableServiceID(model.ProductID, model.Namespace, model.WorkloadType, model.WorkloadName, model.ContainerType, model.ContainerName)
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ListManagedServices(productID), nil
}

func (s *DatabaseStore) RemoveManagedServices(productID string, serviceIDs []string) ([]domain.ManagedService, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, errors.New("product id is required")
	}
	if _, ok := s.GetEnvironment(productID); !ok {
		return nil, errors.New("product not found")
	}
	ids := make([]string, 0, len(serviceIDs))
	seen := map[string]bool{}
	for _, id := range serviceIDs {
		value := strings.TrimSpace(id)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		return s.ListManagedServices(productID), nil
	}
	if err := s.db.Where("product_id = ? AND id IN ?", productID, ids).Delete(&ServiceModel{}).Error; err != nil {
		return nil, err
	}
	return s.ListManagedServices(productID), nil
}

func (s *DatabaseStore) ConfirmManagedServiceRegistry(productID string, registryHost string, harborProjects []string) ([]domain.ManagedService, error) {
	productID = strings.TrimSpace(productID)
	registryHost = normalizeRegistryHost(registryHost)
	if productID == "" {
		return nil, errors.New("product id is required")
	}
	if registryHost == "" {
		return nil, errors.New("private registry host is required")
	}
	projectSet := make(map[string]bool, len(harborProjects))
	for _, project := range harborProjects {
		if value := strings.TrimSpace(project); value != "" {
			projectSet[value] = true
		}
	}
	var models []ServiceModel
	if err := s.db.Where("product_id = ?", productID).Find(&models).Error; err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return []domain.ManagedService{}, nil
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, model := range models {
			imageRegistry := normalizeRegistryHost(model.ImageRegistry)
			imageProject := strings.TrimSpace(model.ImageProject)
			imageSource := "EXTERNAL"
			privateRegistryHost := ""
			privateRegistryConfirmed := false
			if imageRegistry != "" && sameRegistryHostValue(imageRegistry, registryHost) {
				privateRegistryHost = registryHost
				privateRegistryConfirmed = true
				if projectSet[imageProject] {
					imageSource = "PRIVATE"
				} else {
					imageSource = "UNMATCHED_PRIVATE"
				}
			}
			if err := tx.Model(&model).Updates(map[string]any{
				"image_source":               imageSource,
				"private_registry_host":      privateRegistryHost,
				"private_registry_confirmed": privateRegistryConfirmed,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ListManagedServices(productID), nil
}

func (s *DatabaseStore) BindManagedServicePipeline(productID string, serviceID string, jobName string, jobURL string, branch string) (domain.ManagedService, bool, error) {
	productID = strings.TrimSpace(productID)
	serviceID = strings.TrimSpace(serviceID)
	jobName = strings.TrimSpace(jobName)
	jobURL = strings.TrimSpace(jobURL)
	branch = strings.TrimSpace(branch)
	if productID == "" || serviceID == "" {
		return domain.ManagedService{}, false, errors.New("product id and service id are required")
	}
	if jobName == "" {
		return domain.ManagedService{}, false, errors.New("jenkins job name is required")
	}
	var model ServiceModel
	if err := s.db.Where("product_id = ? AND id = ?", productID, serviceID).Take(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ManagedService{}, false, nil
		}
		return domain.ManagedService{}, false, err
	}
	now := time.Now()
	if err := s.db.Model(&model).Updates(map[string]any{
		"jenkins_job_name":  jobName,
		"jenkins_job_url":   jobURL,
		"jenkins_branch":    branch,
		"pipeline_bound_at": now,
	}).Error; err != nil {
		return domain.ManagedService{}, false, err
	}
	model.JenkinsJobName = jobName
	model.JenkinsJobURL = jobURL
	model.JenkinsBranch = branch
	model.PipelineBoundAt = &now
	return toDomainManagedService(model), true, nil
}

func normalizeRegistryHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.TrimRight(strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(value, "http://"), "https://")), "/")
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	return strings.TrimRight(strings.ToLower(host), "/")
}

func sameRegistryHostValue(left string, right string) bool {
	left = normalizeRegistryHost(left)
	right = normalizeRegistryHost(right)
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

func (s *DatabaseStore) CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error) {
	model := ReleaseOrderModel{
		ID:                   strings.TrimSpace(input.ID),
		Type:                 strings.TrimSpace(input.Type),
		SourceBaselineID:     strings.TrimSpace(input.SourceBaselineID),
		ReleaseSource:        strings.TrimSpace(input.ReleaseSource),
		ExecutionMode:        strings.TrimSpace(input.ExecutionMode),
		BuildID:              strings.TrimSpace(input.BuildID),
		BuildStatus:          strings.TrimSpace(input.BuildStatus),
		BuildURL:             strings.TrimSpace(input.BuildURL),
		JenkinsID:            strings.TrimSpace(input.JenkinsID),
		JenkinsJobName:       strings.TrimSpace(input.JenkinsJobName),
		JenkinsJobURL:        strings.TrimSpace(input.JenkinsJobURL),
		ImageRepository:      strings.TrimSpace(input.ImageRepository),
		ImageTag:             strings.TrimSpace(input.ImageTag),
		ImageDigest:          strings.TrimSpace(input.ImageDigest),
		TargetEnvironmentID:  strings.TrimSpace(input.TargetEnvironmentID),
		AgentID:              strings.TrimSpace(input.AgentID),
		ServiceIDs:           encodeStringSlice(input.ServiceIDs),
		ServiceNames:         encodeStringSlice(input.ServiceNames),
		Status:               fallbackString(strings.TrimSpace(input.Status), "PENDING"),
		Progress:             input.Progress,
		SelectedServiceCount: input.SelectedServiceCount,
		CreatedBy:            "admin",
	}
	if model.ID == "" || model.Type == "" || model.TargetEnvironmentID == "" {
		return domain.ReleaseOrder{}, errors.New("missing required fields")
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.ReleaseOrder{}, err
	}
	order, ok := s.releaseOrderByID(model.ID)
	if !ok {
		return domain.ReleaseOrder{}, errors.New("release order not found after create")
	}
	return order, nil
}

func (s *DatabaseStore) ListReleases(query string) []domain.ReleaseOrder {
	type row struct {
		ReleaseOrderModel
		TargetEnvironmentName string
		AgentName             string
	}
	var rows []row
	db := s.db.Table("release_orders").
		Select("release_orders.*, environments.name AS target_environment_name, agents.name AS agent_name").
		Joins("LEFT JOIN environments ON environments.id = release_orders.target_environment_id").
		Joins("LEFT JOIN agents ON agents.id = release_orders.agent_id").
		Order("release_orders.created_at desc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("release_orders.id ILIKE ? OR release_orders.type ILIKE ? OR release_orders.status ILIKE ? OR environments.name ILIKE ? OR agents.name ILIKE ?", like, like, like, like, like)
	}
	if err := db.Find(&rows).Error; err != nil {
		return []domain.ReleaseOrder{}
	}
	items := make([]domain.ReleaseOrder, 0, len(rows))
	for _, item := range rows {
		items = append(items, toDomainReleaseOrder(item.ReleaseOrderModel, item.TargetEnvironmentName, item.AgentName))
	}
	return items
}

func (s *DatabaseStore) ListServiceReleases(productID string, serviceID string) []domain.ReleaseOrder {
	trimmedProductID := strings.TrimSpace(productID)
	trimmedServiceID := strings.TrimSpace(serviceID)
	if trimmedProductID == "" || trimmedServiceID == "" {
		return []domain.ReleaseOrder{}
	}
	type row struct {
		ReleaseOrderModel
		TargetEnvironmentName string
		AgentName             string
	}
	var rows []row
	err := s.db.Table("release_orders").
		Select("release_orders.*, environments.name AS target_environment_name, agents.name AS agent_name").
		Joins("LEFT JOIN environments ON environments.id = release_orders.target_environment_id").
		Joins("LEFT JOIN agents ON agents.id = release_orders.agent_id").
		Where("release_orders.target_environment_id = ?", trimmedProductID).
		Order("release_orders.created_at desc").
		Find(&rows).Error
	if err != nil {
		return []domain.ReleaseOrder{}
	}
	items := make([]domain.ReleaseOrder, 0, len(rows))
	for _, item := range rows {
		order := toDomainReleaseOrder(item.ReleaseOrderModel, item.TargetEnvironmentName, item.AgentName)
		if stringSliceContains(order.ServiceIDs, trimmedServiceID) {
			items = append(items, order)
		}
	}
	return items
}

func (s *DatabaseStore) GetReleaseDetail(id string) (domain.ReleaseDetail, bool) {
	order, ok := s.releaseOrderByID(id)
	if !ok {
		return domain.ReleaseDetail{}, false
	}
	return domain.ReleaseDetail{
		ID:                    order.ID,
		Type:                  order.Type,
		SourceBaselineID:      order.SourceBaselineID,
		ReleaseSource:         order.ReleaseSource,
		ExecutionMode:         order.ExecutionMode,
		BuildID:               order.BuildID,
		BuildStatus:           order.BuildStatus,
		BuildURL:              order.BuildURL,
		JenkinsID:             order.JenkinsID,
		JenkinsJobName:        order.JenkinsJobName,
		JenkinsJobURL:         order.JenkinsJobURL,
		ImageRepository:       order.ImageRepository,
		ImageTag:              order.ImageTag,
		ImageDigest:           order.ImageDigest,
		TargetEnvironmentID:   order.TargetEnvironmentID,
		TargetEnvironmentName: order.TargetEnvironmentName,
		Status:                order.Status,
		Progress:              order.Progress,
		AgentName:             order.AgentName,
		AgentTaskID:           order.ID,
		ServiceIDs:            order.ServiceIDs,
		ServiceNames:          order.ServiceNames,
		Steps:                 []domain.ReleaseStep{},
		Failures:              []domain.ReleaseFailure{},
		ActionRecords:         []domain.ActionRecord{},
		Logs:                  []string{},
	}, true
}

func (s *DatabaseStore) UpdateReleaseBuildStatus(id string, buildID string, buildStatus string, buildURL string, status string, progress int) (domain.ReleaseOrder, bool, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ReleaseOrder{}, false, nil
	}
	updates := map[string]any{}
	if value := strings.TrimSpace(buildID); value != "" {
		updates["build_id"] = value
	}
	if value := strings.TrimSpace(buildStatus); value != "" {
		updates["build_status"] = value
	}
	if value := strings.TrimSpace(buildURL); value != "" {
		updates["build_url"] = value
	}
	if value := strings.TrimSpace(status); value != "" {
		updates["status"] = value
	}
	if progress >= 0 {
		updates["progress"] = progress
	}
	if len(updates) == 0 {
		order, ok := s.releaseOrderByID(trimmedID)
		return order, ok, nil
	}
	result := s.db.Model(&ReleaseOrderModel{}).Where("id = ?", trimmedID).Updates(updates)
	if result.Error != nil {
		return domain.ReleaseOrder{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ReleaseOrder{}, false, nil
	}
	order, ok := s.releaseOrderByID(trimmedID)
	return order, ok, nil
}

func (s *DatabaseStore) UpdateReleaseImage(id string, imageRepository string, imageTag string) (domain.ReleaseOrder, bool, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ReleaseOrder{}, false, nil
	}
	updates := map[string]any{}
	if value := strings.TrimSpace(imageRepository); value != "" {
		updates["image_repository"] = value
	}
	if value := strings.TrimSpace(imageTag); value != "" {
		updates["image_tag"] = value
	}
	if len(updates) == 0 {
		order, ok := s.releaseOrderByID(trimmedID)
		return order, ok, nil
	}
	result := s.db.Model(&ReleaseOrderModel{}).Where("id = ?", trimmedID).Updates(updates)
	if result.Error != nil {
		return domain.ReleaseOrder{}, false, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ReleaseOrder{}, false, nil
	}
	order, ok := s.releaseOrderByID(trimmedID)
	return order, ok, nil
}

func (s *DatabaseStore) ListDeployTasks(query string) []domain.DeployTask {
	return []domain.DeployTask{}
}

func (s *DatabaseStore) GetDeployDetail(id string) (domain.DeployDetail, bool) {
	return domain.DeployDetail{}, false
}

func (s *DatabaseStore) HasEnvironmentAction(environmentID string, action string) bool {
	return true
}

func toDomainProject(model ProjectModel, productCount int) domain.Project {
	createdAt := ""
	if !model.CreatedAt.IsZero() {
		createdAt = model.CreatedAt.Format(time.RFC3339)
	}
	return domain.Project{
		ID:           model.ID,
		Name:         model.Name,
		Code:         model.Code,
		Description:  model.Description,
		Status:       normalizeProjectStatus(model.Status),
		ProductCount: productCount,
		CreatedAt:    createdAt,
	}
}

func toDomainEnvironment(model EnvironmentModel, boundAgentStatus string, project projectLookupInfo) domain.Environment {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	agentStatus := "UNBOUND"
	if model.AgentID != "" {
		agentStatus = fallbackString(boundAgentStatus, "OFFLINE")
	}
	return domain.Environment{
		ID:                  model.ID,
		Name:                model.Name,
		Code:                model.Code,
		ProjectID:           model.ProjectID,
		ProjectName:         project.Name,
		ProductStatus:       normalizeProductStatusWithProject(model.ProductStatus, model.ProjectID, project),
		Type:                model.Type,
		DeployTargetType:    fallbackString(strings.TrimSpace(model.DeployTargetType), "KUBERNETES"),
		NetworkMode:         model.NetworkMode,
		ClusterID:           model.ClusterID,
		Namespace:           model.Namespace,
		RegistryID:          model.RegistryID,
		RegistryProject:     model.RegistryProject,
		PrivateRegistryHost: model.PrivateRegistryHost,
		JenkinsID:           model.JenkinsID,
		JenkinsView:         model.JenkinsView,
		Status:              model.Status,
		AgentStatus:         agentStatus,
		LastCheckAt:         lastCheckAt,
	}
}

func toDomainKubernetesCluster(model KubernetesClusterModel) domain.KubernetesCluster {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	return domain.KubernetesCluster{
		ID:            model.ID,
		Name:          model.Name,
		APIServer:     model.APIServer,
		Context:       model.Context,
		CredentialRef: model.CredentialRef,
		Kubeconfig:    model.Kubeconfig,
		Status:        model.Status,
		LastCheckAt:   lastCheckAt,
		ProbeMessage:  model.ProbeMessage,
		Namespaces:    compactStringList(model.Namespaces),
	}
}

func toDomainHarborRegistry(model HarborRegistryModel) domain.HarborRegistry {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	return domain.HarborRegistry{
		ID:                    model.ID,
		Name:                  model.Name,
		URL:                   model.URL,
		RegistryHost:          model.RegistryHost,
		Scheme:                fallbackString(model.Scheme, schemeFromURL(model.URL)),
		Username:              model.Username,
		CredentialRef:         model.CredentialRef,
		Password:              model.Password,
		InsecureSkipTLSVerify: model.InsecureSkipTLSVerify,
		Status:                model.Status,
		LastCheckAt:           lastCheckAt,
		ProbeMessage:          model.ProbeMessage,
		Projects:              compactStringList(model.Projects),
	}
}

func toDomainJenkinsInstance(model JenkinsInstanceModel) domain.JenkinsInstance {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	return domain.JenkinsInstance{
		ID:                    model.ID,
		Name:                  model.Name,
		URL:                   model.URL,
		Username:              model.Username,
		CredentialRef:         model.CredentialRef,
		Token:                 model.Token,
		InsecureSkipTLSVerify: model.InsecureSkipTLSVerify,
		Status:                model.Status,
		LastCheckAt:           lastCheckAt,
		ProbeMessage:          model.ProbeMessage,
		Views:                 compactStringList(model.Views),
		Jobs:                  compactStringList(model.Jobs),
		Pipelines:             compactJenkinsPipelines(model.Pipelines),
	}
}

func compactStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	output := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		output = append(output, trimmed)
	}
	return output
}

func compactJenkinsPipelines(values []domain.JenkinsPipeline) []domain.JenkinsPipeline {
	if len(values) == 0 {
		return []domain.JenkinsPipeline{}
	}
	seen := map[string]struct{}{}
	output := make([]domain.JenkinsPipeline, 0, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		value.View = strings.TrimSpace(value.View)
		value.ViewURL = strings.TrimSpace(value.ViewURL)
		value.URL = strings.TrimSpace(value.URL)
		if value.Name == "" {
			continue
		}
		parameters := make([]domain.JenkinsPipelineParameter, 0, len(value.Parameters))
		parameterSeen := map[string]struct{}{}
		for _, parameter := range value.Parameters {
			parameter.Name = strings.TrimSpace(parameter.Name)
			parameter.Type = strings.TrimSpace(parameter.Type)
			parameter.DefaultValue = strings.TrimSpace(parameter.DefaultValue)
			parameter.Description = strings.TrimSpace(parameter.Description)
			if parameter.Name == "" {
				continue
			}
			if _, ok := parameterSeen[parameter.Name]; ok {
				continue
			}
			parameterSeen[parameter.Name] = struct{}{}
			parameters = append(parameters, parameter)
		}
		value.Parameters = parameters
		key := value.View + "\x00" + value.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, value)
	}
	sort.Slice(output, func(i, j int) bool {
		if output[i].View == output[j].View {
			return output[i].Name < output[j].Name
		}
		return output[i].View < output[j].View
	})
	return output
}

func stringListContains(values []string, target string) bool {
	trimmedTarget := strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == trimmedTarget {
			return true
		}
	}
	return false
}

func normalizeScheme(scheme string, rawURL string) string {
	value := strings.ToLower(strings.TrimSpace(scheme))
	if value == "http" || value == "https" {
		return value
	}
	return schemeFromURL(rawURL)
}

func schemeFromURL(rawURL string) string {
	value := strings.ToLower(strings.TrimSpace(rawURL))
	if strings.HasPrefix(value, "http://") {
		return "http"
	}
	if strings.HasPrefix(value, "https://") {
		return "https"
	}
	return "https"
}

func normalizeResourceURL(rawURL string, scheme string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" || strings.HasPrefix(strings.ToLower(value), "http://") || strings.HasPrefix(strings.ToLower(value), "https://") {
		return value
	}
	return fallbackString(normalizeScheme(scheme, value), "https") + "://" + value
}

func toDomainAgent(model AgentModel, environmentName string) domain.Agent {
	lastHeartbeatAt := ""
	if model.LastHeartbeatAt != nil {
		lastHeartbeatAt = model.LastHeartbeatAt.Format(time.RFC3339)
	}
	var currentTaskID *string
	if model.CurrentTaskID != "" {
		currentTaskID = &model.CurrentTaskID
	}
	return domain.Agent{
		ID:              model.ID,
		Name:            model.Name,
		EnvironmentID:   model.EnvironmentID,
		EnvironmentName: environmentName,
		Version:         model.Version,
		Status:          effectiveAgentStatus(model.Status, model.LastHeartbeatAt),
		ClaimStatus:     fallbackString(model.ClaimStatus, "PENDING_CLAIM"),
		Capabilities:    model.Capabilities,
		RuntimeStatus:   model.RuntimeStatus,
		LastHeartbeatAt: lastHeartbeatAt,
		CurrentTaskID:   currentTaskID,
	}
}

func toDomainOperationLog(model OperationLogModel) domain.OperationLog {
	return domain.OperationLog{
		ID:            model.ID,
		OperatorID:    model.OperatorID,
		OperatorName:  model.OperatorName,
		Action:        model.Action,
		ResourceType:  model.ResourceType,
		ResourceID:    model.ResourceID,
		ResourceName:  model.ResourceName,
		ProjectID:     model.ProjectID,
		ProjectName:   model.ProjectName,
		EnvironmentID: model.EnvironmentID,
		ProductName:   model.ProductName,
		TaskID:        model.TaskID,
		Namespace:     model.Namespace,
		WorkloadType:  model.WorkloadType,
		WorkloadName:  model.WorkloadName,
		ContainerName: model.ContainerName,
		ContainerType: model.ContainerType,
		Result:        model.Result,
		Detail:        model.Detail,
		CreatedAt:     model.CreatedAt.Format(time.RFC3339),
	}
}

func runtimeStatusHasData(status domain.RuntimeStatus) bool {
	return status.Kubernetes.Status != "" ||
		status.Kubernetes.Message != "" ||
		status.Kubernetes.UpdatedAt != "" ||
		status.Kubernetes.Endpoint != "" ||
		len(status.Kubernetes.Items) > 0 ||
		len(status.Kubernetes.Workloads) > 0 ||
		status.Harbor.Status != "" ||
		status.Harbor.Message != "" ||
		status.Harbor.UpdatedAt != "" ||
		status.Harbor.Endpoint != "" ||
		len(status.Harbor.Items) > 0 ||
		len(status.Harbor.Workloads) > 0
}

func toDomainReleaseSourceService(model ServiceModel) domain.ReleaseSourceService {
	repository := fallbackString(model.ImageRepository, strings.TrimSuffix(model.Image, ":"+model.ImageTag))
	return domain.ReleaseSourceService{
		ServiceID:                model.ID,
		ServiceName:              model.Name,
		Namespace:                model.Namespace,
		WorkloadName:             model.WorkloadName,
		WorkloadType:             model.WorkloadType,
		ImageRegistry:            model.ImageRegistry,
		ImageProject:             model.ImageProject,
		ImageRepository:          repository,
		ImageTag:                 model.ImageTag,
		ImageSource:              fallbackString(model.ImageSource, "UNKNOWN"),
		PrivateRegistryHost:      model.PrivateRegistryHost,
		PrivateRegistryConfirmed: model.PrivateRegistryConfirmed,
		JenkinsJobName:           model.JenkinsJobName,
		JenkinsJobURL:            model.JenkinsJobURL,
		JenkinsBranch:            model.JenkinsBranch,
		JenkinsPipelineBound:     strings.TrimSpace(model.JenkinsJobName) != "",
		PipelineBoundAt:          formatOptionalTime(model.PipelineBoundAt),
		Tags:                     []domain.ReleaseImageTag{},
		Publishable:              false,
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func toDomainManagedService(model ServiceModel) domain.ManagedService {
	return domain.ManagedService{
		ID:                       model.ID,
		ProductID:                model.ProductID,
		Name:                     model.Name,
		Namespace:                model.Namespace,
		WorkloadName:             model.WorkloadName,
		WorkloadType:             model.WorkloadType,
		ContainerName:            model.ContainerName,
		ContainerType:            fallbackString(model.ContainerType, "APP"),
		Image:                    model.Image,
		ImageRegistry:            model.ImageRegistry,
		ImageProject:             model.ImageProject,
		ImageRepository:          model.ImageRepository,
		ImageTag:                 model.ImageTag,
		ImageSource:              fallbackString(model.ImageSource, "UNKNOWN"),
		PrivateRegistryHost:      model.PrivateRegistryHost,
		PrivateRegistryConfirmed: model.PrivateRegistryConfirmed,
		JenkinsJobName:           model.JenkinsJobName,
		JenkinsJobURL:            model.JenkinsJobURL,
		JenkinsBranch:            model.JenkinsBranch,
		JenkinsPipelineBound:     strings.TrimSpace(model.JenkinsJobName) != "",
		PipelineBoundAt:          formatOptionalTime(model.PipelineBoundAt),
		Replicas:                 model.Replicas,
		ReadyReplicas:            model.ReadyReplicas,
		CreatedAt:                model.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                model.UpdatedAt.Format(time.RFC3339),
	}
}

func baselineItemModelFromService(baselineID string, service ServiceModel) BaselineServiceItemModel {
	return BaselineServiceItemModel{
		BaselineID:    baselineID,
		ServiceID:     service.ID,
		ServiceName:   fallbackString(service.Name, service.WorkloadName),
		Namespace:     service.Namespace,
		WorkloadName:  service.WorkloadName,
		WorkloadType:  service.WorkloadType,
		Image:         service.Image,
		Tag:           service.ImageTag,
		Replicas:      service.Replicas,
		ReadyReplicas: service.ReadyReplicas,
		HealthStatus:  serviceHealthStatus(service),
	}
}

func serviceHealthStatus(service ServiceModel) string {
	if service.Replicas > 0 && service.ReadyReplicas < service.Replicas {
		return "DEGRADED"
	}
	return "HEALTHY"
}

func toDomainBaseline(model EnvironmentBaselineModel, sourceEnvironmentName string) domain.Baseline {
	lockedAt := ""
	if model.LockedAt != nil {
		lockedAt = model.LockedAt.Format(time.RFC3339)
	}
	return domain.Baseline{
		ID:                    model.ID,
		Name:                  model.Name,
		SourceEnvironmentID:   model.SourceEnvironmentID,
		SourceEnvironmentName: sourceEnvironmentName,
		ServiceCount:          model.ServiceCount,
		CreatedBy:             model.CreatedBy,
		CreatedAt:             model.CreatedAt.Format(time.RFC3339),
		Status:                model.Status,
		Purpose:               model.Purpose,
		LockedAt:              lockedAt,
		SnapshotSource:        model.SourceEnvironmentID,
		SnapshotCollectedAt:   model.CreatedAt.Format(time.RFC3339),
		SnapshotMode:          "REAL_RUNTIME",
	}
}

func toDomainBaselineDetail(model EnvironmentBaselineModel, sourceEnvironmentName string, itemModels []BaselineServiceItemModel) domain.BaselineDetail {
	baseline := toDomainBaseline(model, sourceEnvironmentName)
	items := make([]domain.BaselineItem, 0, len(itemModels))
	for _, item := range itemModels {
		items = append(items, toDomainBaselineItem(item))
	}
	return domain.BaselineDetail{
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
		SnapshotSource:        baseline.SnapshotSource,
		SnapshotCollectedAt:   baseline.SnapshotCollectedAt,
		SnapshotMode:          baseline.SnapshotMode,
		SnapshotTaskID:        baseline.ID,
		Items:                 items,
	}
}

func toDomainBaselineItem(model BaselineServiceItemModel) domain.BaselineItem {
	return domain.BaselineItem{
		ServiceID:     model.ServiceID,
		ServiceName:   model.ServiceName,
		Namespace:     model.Namespace,
		WorkloadName:  model.WorkloadName,
		WorkloadType:  model.WorkloadType,
		Tag:           model.Tag,
		Digest:        model.Digest,
		Replicas:      model.Replicas,
		ReadyReplicas: model.ReadyReplicas,
		HealthStatus:  model.HealthStatus,
	}
}

func serviceModelFromDiscovered(productID string, service domain.DiscoveredService) ServiceModel {
	namespace := strings.TrimSpace(service.Namespace)
	workloadName := strings.TrimSpace(service.WorkloadName)
	if workloadName == "" {
		workloadName = strings.TrimSpace(service.Name)
	}
	workloadType := fallbackString(strings.TrimSpace(service.WorkloadType), "Deployment")
	containerName := strings.TrimSpace(service.ContainerName)
	containerType := fallbackString(strings.TrimSpace(service.ContainerType), "APP")
	image := strings.TrimSpace(service.Image)
	imageRegistry := strings.TrimSpace(service.ImageRegistry)
	imageProject := strings.TrimSpace(service.ImageProject)
	imageRepository := strings.TrimSpace(service.ImageRepository)
	imageTag := strings.TrimSpace(service.ImageTag)
	imageSource := fallbackString(strings.TrimSpace(service.ImageSource), "UNKNOWN")
	privateRegistryHost := strings.TrimSpace(service.PrivateRegistryHost)
	id := strings.TrimSpace(service.ID)
	if id == "" {
		id = stableServiceID(productID, namespace, workloadType, workloadName, containerType, containerName)
	}
	return ServiceModel{
		ID:                       id,
		ProductID:                productID,
		Name:                     fallbackString(strings.TrimSpace(service.Name), workloadName),
		Namespace:                namespace,
		WorkloadName:             workloadName,
		WorkloadType:             workloadType,
		ContainerName:            containerName,
		ContainerType:            containerType,
		Image:                    image,
		ImageRegistry:            imageRegistry,
		ImageProject:             imageProject,
		ImageRepository:          imageRepository,
		ImageTag:                 imageTag,
		ImageSource:              imageSource,
		PrivateRegistryHost:      privateRegistryHost,
		PrivateRegistryConfirmed: service.PrivateRegistryConfirmed,
		Replicas:                 service.Replicas,
		ReadyReplicas:            service.ReadyReplicas,
	}
}

func stableServiceID(parts ...string) string {
	key := strings.Join(parts, "\x00")
	sum := sha1.Sum([]byte(key))
	return "svc-" + hexString(sum[:8])
}

func hexString(input []byte) string {
	const chars = "0123456789abcdef"
	output := make([]byte, len(input)*2)
	for index, value := range input {
		output[index*2] = chars[value>>4]
		output[index*2+1] = chars[value&0x0f]
	}
	return string(output)
}

func toDomainReleaseOrder(model ReleaseOrderModel, environmentName string, agentName string) domain.ReleaseOrder {
	return domain.ReleaseOrder{
		ID:                    model.ID,
		Type:                  model.Type,
		SourceBaselineID:      model.SourceBaselineID,
		ReleaseSource:         model.ReleaseSource,
		ExecutionMode:         model.ExecutionMode,
		BuildID:               model.BuildID,
		BuildStatus:           model.BuildStatus,
		BuildURL:              model.BuildURL,
		JenkinsID:             model.JenkinsID,
		JenkinsJobName:        model.JenkinsJobName,
		JenkinsJobURL:         model.JenkinsJobURL,
		ImageRepository:       model.ImageRepository,
		ImageTag:              model.ImageTag,
		ImageDigest:           model.ImageDigest,
		TargetEnvironmentID:   model.TargetEnvironmentID,
		TargetEnvironmentName: fallbackString(environmentName, model.TargetEnvironmentID),
		Status:                model.Status,
		Progress:              model.Progress,
		AgentName:             fallbackString(agentName, model.AgentID),
		ServiceIDs:            decodeStringSlice(model.ServiceIDs),
		ServiceNames:          decodeStringSlice(model.ServiceNames),
	}
}

func (s *DatabaseStore) releaseOrderByID(id string) (domain.ReleaseOrder, bool) {
	type row struct {
		ReleaseOrderModel
		TargetEnvironmentName string
		AgentName             string
	}
	var result row
	err := s.db.Table("release_orders").
		Select("release_orders.*, environments.name AS target_environment_name, agents.name AS agent_name").
		Joins("LEFT JOIN environments ON environments.id = release_orders.target_environment_id").
		Joins("LEFT JOIN agents ON agents.id = release_orders.agent_id").
		Where("release_orders.id = ?", id).
		Take(&result).Error
	if err != nil {
		return domain.ReleaseOrder{}, false
	}
	return toDomainReleaseOrder(result.ReleaseOrderModel, result.TargetEnvironmentName, result.AgentName), true
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func encodeStringSlice(values []string) string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return ""
	}
	content, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(content)
}

func decodeStringSlice(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(trimmed), &values); err != nil {
		return []string{}
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if cleaned := strings.TrimSpace(value); cleaned != "" {
			normalized = append(normalized, cleaned)
		}
	}
	return normalized
}

func stringSliceContains(values []string, target string) bool {
	trimmedTarget := strings.TrimSpace(target)
	if trimmedTarget == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == trimmedTarget {
			return true
		}
	}
	return false
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return fmt.Sprintf("%x", sum)[:16]
}

func normalizeEnvironmentCode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, item := range normalized {
		if (item >= 'a' && item <= 'z') || (item >= '0' && item <= '9') {
			builder.WriteRune(item)
			lastDash = false
			continue
		}
		if builder.Len() > 0 && !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func generateEnvironmentCode(name string, environmentType string) string {
	hasNonASCII := false
	for _, item := range name {
		if item > 127 {
			hasNonASCII = true
			break
		}
	}
	if !hasNonASCII {
		if code := normalizeEnvironmentCode(name); code != "" {
			return code
		}
	}
	prefix := "remote"
	if strings.ToUpper(strings.TrimSpace(environmentType)) == "LOCAL" {
		prefix = "local"
	}
	return prefix + "-" + time.Now().Format("20060102150405")
}

func normalizeEnvironmentInput(input domain.Environment) (domain.Environment, error) {
	environmentType := strings.ToUpper(strings.TrimSpace(input.Type))
	code := normalizeEnvironmentCode(input.Code)
	if code == "" {
		code = generateEnvironmentCode(input.Name, environmentType)
	}
	deployTargetType := strings.ToUpper(strings.TrimSpace(input.DeployTargetType))
	if deployTargetType == "" {
		deployTargetType = "KUBERNETES"
	}
	if deployTargetType != "KUBERNETES" && deployTargetType != "DOCKER_COMPOSE" {
		return domain.Environment{}, errors.New("unsupported deploy target type")
	}
	item := domain.Environment{
		ID:                  strings.TrimSpace(input.ID),
		Name:                strings.TrimSpace(input.Name),
		Code:                code,
		ProjectID:           strings.TrimSpace(input.ProjectID),
		ProductStatus:       normalizeProductStatus(input.ProductStatus, input.ProjectID),
		Type:                environmentType,
		DeployTargetType:    deployTargetType,
		NetworkMode:         strings.ToUpper(strings.TrimSpace(input.NetworkMode)),
		ClusterID:           strings.TrimSpace(input.ClusterID),
		Namespace:           strings.TrimSpace(input.Namespace),
		RegistryID:          strings.TrimSpace(input.RegistryID),
		RegistryProject:     strings.TrimSpace(input.RegistryProject),
		PrivateRegistryHost: strings.TrimSpace(input.PrivateRegistryHost),
		JenkinsID:           strings.TrimSpace(input.JenkinsID),
		JenkinsView:         strings.TrimSpace(input.JenkinsView),
		Status:              strings.TrimSpace(input.Status),
	}
	if item.Type == "" {
		return domain.Environment{}, errors.New("environment type is required")
	}
	item.Bindings = normalizeEnvironmentBindings(input.Bindings)
	if len(item.Bindings) == 0 {
		item.Bindings = defaultEnvironmentBindings(item)
	} else {
		normalizeEnvironmentBindingRoles(&item)
		applyDefaultBindingsToLegacyFields(&item)
	}
	if item.Type == "LOCAL" {
		item.NetworkMode = "DIRECT"
		if item.ClusterID == "" || item.Namespace == "" || item.RegistryID == "" || item.RegistryProject == "" {
			return domain.Environment{}, errors.New("local environment requires kubernetes and harbor scopes")
		}
		item.Bindings = normalizeEnvironmentBindings(item.Bindings)
		return item, nil
	}
	item.Type = "PROJECT"
	item.NetworkMode = "AGENT"
	item.ClusterID = ""
	item.Namespace = ""
	item.Bindings = filterEnvironmentBindings(item.Bindings, func(binding domain.EnvironmentResourceBinding) bool {
		if binding.BindingRole == "RUNTIME_TARGET" {
			return binding.ResourceType == "K8S" || binding.ResourceType == "HARBOR"
		}
		return binding.BindingRole == "BUILD_SOURCE" && (binding.ResourceType == "HARBOR" || binding.ResourceType == "JENKINS")
	})
	if item.RegistryID == "" || item.RegistryProject == "" {
		return domain.Environment{}, errors.New("project environment requires harbor scopes")
	}
	if item.JenkinsID == "" || item.JenkinsView == "" {
		return domain.Environment{}, errors.New("project environment requires jenkins scopes")
	}
	item.Bindings = normalizeEnvironmentBindings(item.Bindings)
	return item, nil
}

func normalizeProjectStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ACTIVE", "DISABLED":
		return strings.ToUpper(strings.TrimSpace(status))
	default:
		return "ACTIVE"
	}
}

func normalizeProductStatus(status string, projectID string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	if normalized == "DISABLED" {
		return normalized
	}
	if strings.TrimSpace(projectID) == "" {
		return "UNBOUND"
	}
	return "BOUND"
}

func normalizeProductStatusWithProject(status string, projectID string, project projectLookupInfo) string {
	normalized := normalizeProductStatus(status, projectID)
	if normalized != "BOUND" {
		return normalized
	}
	if !project.Found || normalizeProjectStatus(project.Status) == "DISABLED" {
		return "DISABLED"
	}
	return "BOUND"
}

func defaultEnvironmentBindings(item domain.Environment) []domain.EnvironmentResourceBinding {
	bindings := []domain.EnvironmentResourceBinding{}
	if item.ClusterID != "" && item.Namespace != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "K8S",
			BindingRole:  "BUILD_SOURCE",
			ResourceID:   item.ClusterID,
			ScopeType:    "NAMESPACE",
			ScopeValue:   item.Namespace,
			IsDefault:    true,
		})
	}
	if item.RegistryID != "" && item.RegistryProject != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "HARBOR",
			BindingRole:  "BUILD_SOURCE",
			ResourceID:   item.RegistryID,
			ScopeType:    "PROJECT",
			ScopeValue:   item.RegistryProject,
			IsDefault:    true,
		})
	}
	if item.JenkinsID != "" && item.JenkinsView != "" {
		bindings = append(bindings, domain.EnvironmentResourceBinding{
			ResourceType: "JENKINS",
			BindingRole:  "BUILD_SOURCE",
			ResourceID:   item.JenkinsID,
			ScopeType:    "VIEW",
			ScopeValue:   item.JenkinsView,
			IsDefault:    true,
		})
	}
	return bindings
}

func normalizeEnvironmentBindings(input []domain.EnvironmentResourceBinding) []domain.EnvironmentResourceBinding {
	items := make([]domain.EnvironmentResourceBinding, 0, len(input))
	defaultSeen := map[string]bool{}
	seen := map[string]bool{}
	for _, binding := range input {
		item := domain.EnvironmentResourceBinding{
			ID:            strings.TrimSpace(binding.ID),
			EnvironmentID: strings.TrimSpace(binding.EnvironmentID),
			BindingRole:   normalizeEnvironmentBindingRole(binding.BindingRole),
			ResourceType:  strings.ToUpper(strings.TrimSpace(binding.ResourceType)),
			ResourceID:    strings.TrimSpace(binding.ResourceID),
			ScopeType:     strings.ToUpper(strings.TrimSpace(binding.ScopeType)),
			ScopeValue:    strings.TrimSpace(binding.ScopeValue),
			IsDefault:     binding.IsDefault,
		}
		if item.ResourceType == "" || item.ResourceID == "" || item.ScopeType == "" || item.ScopeValue == "" {
			continue
		}
		key := item.BindingRole + "\x00" + item.ResourceType + "\x00" + item.ResourceID + "\x00" + item.ScopeType + "\x00" + item.ScopeValue
		if seen[key] {
			continue
		}
		seen[key] = true
		defaultKey := item.BindingRole + "\x00" + item.ResourceType
		if !defaultSeen[defaultKey] {
			item.IsDefault = true
			defaultSeen[defaultKey] = true
		} else {
			item.IsDefault = false
		}
		items = append(items, item)
	}
	return items
}

func normalizeEnvironmentBindingRole(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "RUNTIME_TARGET":
		return "RUNTIME_TARGET"
	default:
		return "BUILD_SOURCE"
	}
}

func normalizeEnvironmentBindingRoles(item *domain.Environment) {
	for index := range item.Bindings {
		item.Bindings[index].BindingRole = normalizeEnvironmentBindingRole(item.Bindings[index].BindingRole)
	}
}

func filterEnvironmentBindings(input []domain.EnvironmentResourceBinding, keep func(domain.EnvironmentResourceBinding) bool) []domain.EnvironmentResourceBinding {
	items := make([]domain.EnvironmentResourceBinding, 0, len(input))
	for _, binding := range input {
		if keep(binding) {
			items = append(items, binding)
		}
	}
	return items
}

func applyDefaultBindingsToLegacyFields(item *domain.Environment) {
	for _, binding := range item.Bindings {
		if binding.BindingRole != "" && binding.BindingRole != "BUILD_SOURCE" {
			continue
		}
		if !binding.IsDefault {
			continue
		}
		switch binding.ResourceType {
		case "K8S":
			item.ClusterID = binding.ResourceID
			item.Namespace = binding.ScopeValue
		case "HARBOR":
			item.RegistryID = binding.ResourceID
			item.RegistryProject = binding.ScopeValue
		case "JENKINS":
			item.JenkinsID = binding.ResourceID
			item.JenkinsView = binding.ScopeValue
		}
	}
}

func (s *DatabaseStore) environmentStatusByScopeCache(item domain.Environment, currentStatus string) string {
	if s.environmentHasUnverifiedScopes(item) {
		return "DEGRADED"
	}
	return verifiedEnvironmentStatus(currentStatus)
}

func (s *DatabaseStore) environmentHasUnverifiedScopes(item domain.Environment) bool {
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
			cluster, exists := s.GetKubernetesCluster(binding.ResourceID)
			if !exists || !stringListContains(cluster.Namespaces, binding.ScopeValue) {
				return true
			}
		case "HARBOR":
			if binding.BindingRole == "RUNTIME_TARGET" {
				continue
			}
			registry, exists := s.GetHarborRegistry(binding.ResourceID)
			if !exists || !stringListContains(registry.Projects, binding.ScopeValue) {
				return true
			}
		case "JENKINS":
			if binding.BindingRole == "RUNTIME_TARGET" {
				continue
			}
			instance, exists := s.GetJenkinsInstance(binding.ResourceID)
			if !exists || !stringListContains(instance.Views, binding.ScopeValue) {
				return true
			}
		}
	}
	return false
}

func (s *DatabaseStore) refreshEnvironmentStatuses(items []domain.Environment) []domain.Environment {
	for index := range items {
		items[index].Status = s.environmentStatusByScopeCache(items[index], items[index].Status)
	}
	return items
}

func verifiedEnvironmentStatus(currentStatus string) string {
	status := strings.TrimSpace(currentStatus)
	if status == "" || status == "DEGRADED" {
		return "UNKNOWN"
	}
	return status
}

func withEnvironmentID(bindings []domain.EnvironmentResourceBinding, environmentID string) []domain.EnvironmentResourceBinding {
	items := make([]domain.EnvironmentResourceBinding, 0, len(bindings))
	for _, binding := range bindings {
		item := binding
		item.EnvironmentID = environmentID
		item.BindingRole = normalizeEnvironmentBindingRole(item.BindingRole)
		if item.ID == "" {
			item.ID = environmentBindingID(item)
		}
		items = append(items, item)
	}
	return items
}

func environmentBindingID(item domain.EnvironmentResourceBinding) string {
	raw := strings.Join([]string{
		item.EnvironmentID,
		strings.ToLower(normalizeEnvironmentBindingRole(item.BindingRole)),
		strings.ToLower(item.ResourceType),
		item.ResourceID,
		strings.ToLower(item.ScopeType),
		item.ScopeValue,
	}, "\x00")
	sum := sha1.Sum([]byte(raw))
	return fmt.Sprintf("%s:%x", item.EnvironmentID, sum)
}

func replaceEnvironmentBindings(tx *gorm.DB, environmentID string, bindings []domain.EnvironmentResourceBinding) error {
	if err := tx.Where("environment_id = ?", environmentID).Delete(&EnvironmentResourceBindingModel{}).Error; err != nil {
		return err
	}
	models := make([]EnvironmentResourceBindingModel, 0, len(bindings))
	for _, binding := range withEnvironmentID(bindings, environmentID) {
		models = append(models, EnvironmentResourceBindingModel{
			ID:            binding.ID,
			EnvironmentID: binding.EnvironmentID,
			BindingRole:   normalizeEnvironmentBindingRole(binding.BindingRole),
			ResourceType:  binding.ResourceType,
			ResourceID:    binding.ResourceID,
			ScopeType:     binding.ScopeType,
			ScopeValue:    binding.ScopeValue,
			IsDefault:     binding.IsDefault,
		})
	}
	if len(models) == 0 {
		return nil
	}
	return tx.Create(&models).Error
}

func (s *DatabaseStore) attachEnvironmentBindings(items []domain.Environment) []domain.Environment {
	if len(items) == 0 {
		return items
	}
	ids := make([]string, 0, len(items))
	byID := make(map[string]int, len(items))
	for index := range items {
		ids = append(ids, items[index].ID)
		byID[items[index].ID] = index
	}
	var models []EnvironmentResourceBindingModel
	if err := s.db.Where("environment_id IN ?", ids).Order("created_at asc").Find(&models).Error; err != nil {
		return items
	}
	for _, model := range models {
		index, ok := byID[model.EnvironmentID]
		if !ok {
			continue
		}
		items[index].Bindings = append(items[index].Bindings, domain.EnvironmentResourceBinding{
			ID:            model.ID,
			EnvironmentID: model.EnvironmentID,
			BindingRole:   normalizeEnvironmentBindingRole(model.BindingRole),
			ResourceType:  model.ResourceType,
			ResourceID:    model.ResourceID,
			ScopeType:     model.ScopeType,
			ScopeValue:    model.ScopeValue,
			IsDefault:     model.IsDefault,
		})
	}
	for index := range items {
		if len(items[index].Bindings) == 0 {
			items[index].Bindings = defaultEnvironmentBindings(items[index])
		}
		normalizeEnvironmentBindingRoles(&items[index])
		applyDefaultBindingsToLegacyFields(&items[index])
	}
	return items
}

func effectiveAgentStatus(status string, lastHeartbeatAt *time.Time) string {
	normalized := fallbackString(strings.TrimSpace(status), "OFFLINE")
	if normalized != "ONLINE" {
		return normalized
	}
	if lastHeartbeatAt == nil || time.Since(*lastHeartbeatAt) > agentHeartbeatTimeout {
		return "OFFLINE"
	}
	return "ONLINE"
}

func (s *DatabaseStore) environmentNameMap() map[string]string {
	var environments []EnvironmentModel
	if err := s.db.Find(&environments).Error; err != nil {
		return map[string]string{}
	}
	items := make(map[string]string, len(environments))
	for _, environment := range environments {
		items[environment.ID] = environment.Name
	}
	return items
}

func (s *DatabaseStore) productCountByProjectID() map[string]int {
	type result struct {
		ProjectID string
		Count     int
	}
	var rows []result
	if err := s.db.Model(&EnvironmentModel{}).
		Select("project_id, count(*) as count").
		Where("project_id <> ''").
		Group("project_id").
		Scan(&rows).Error; err != nil {
		return map[string]int{}
	}
	items := make(map[string]int, len(rows))
	for _, row := range rows {
		items[row.ProjectID] = row.Count
	}
	return items
}

func (s *DatabaseStore) projectInfoByIDMap() map[string]projectLookupInfo {
	var projects []ProjectModel
	if err := s.db.Find(&projects).Error; err != nil {
		return map[string]projectLookupInfo{}
	}
	items := make(map[string]projectLookupInfo, len(projects))
	for _, project := range projects {
		items[project.ID] = projectLookupInfo{Name: project.Name, Status: normalizeProjectStatus(project.Status), Found: true}
	}
	return items
}

func (s *DatabaseStore) getProjectInfo(projectID string) projectLookupInfo {
	if strings.TrimSpace(projectID) == "" {
		return projectLookupInfo{}
	}
	var project ProjectModel
	if err := s.db.Where("id = ?", projectID).Take(&project).Error; err != nil {
		return projectLookupInfo{}
	}
	return projectLookupInfo{Name: project.Name, Status: normalizeProjectStatus(project.Status), Found: true}
}

func (s *DatabaseStore) agentStatusByIDMap() map[string]string {
	var agents []AgentModel
	if err := s.db.Find(&agents).Error; err != nil {
		return map[string]string{}
	}
	items := make(map[string]string, len(agents))
	for _, agent := range agents {
		items[agent.ID] = effectiveAgentStatus(agent.Status, agent.LastHeartbeatAt)
	}
	return items
}

func (s *DatabaseStore) getAgentStatus(agentID string) string {
	if strings.TrimSpace(agentID) == "" {
		return ""
	}
	var agent AgentModel
	if err := s.db.Where("id = ?", agentID).Take(&agent).Error; err != nil {
		return "OFFLINE"
	}
	return effectiveAgentStatus(agent.Status, agent.LastHeartbeatAt)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
