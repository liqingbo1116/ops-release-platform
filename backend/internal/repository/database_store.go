package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"ops-release-platform/backend/internal/domain"
)

type DatabaseStore struct {
	db   *gorm.DB
	mock *MockRepository
}

const agentHeartbeatTimeout = 60 * time.Second

func NewDatabaseStore(db *gorm.DB, mock *MockRepository) *DatabaseStore {
	return &DatabaseStore{db: db, mock: mock}
}

func (s *DatabaseStore) ListEnvironments(query string) []domain.Environment {
	var models []EnvironmentModel
	db := s.db.Order("created_at asc")
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		like := "%" + trimmed + "%"
		db = db.Where("id ILIKE ? OR name ILIKE ? OR code ILIKE ? OR type ILIKE ? OR status ILIKE ?", like, like, like, like, like)
	}
	if err := db.Find(&models).Error; err != nil {
		return []domain.Environment{}
	}
	agentStatuses := s.agentStatusByIDMap()
	items := make([]domain.Environment, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainEnvironment(model, agentStatuses[model.AgentID]))
	}
	return items
}

func (s *DatabaseStore) GetEnvironment(id string) (domain.Environment, bool) {
	var model EnvironmentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Environment{}, false
	}
	return toDomainEnvironment(model, s.getAgentStatus(model.AgentID)), true
}

func (s *DatabaseStore) CreateEnvironment(input domain.Environment) (domain.Environment, error) {
	code := normalizeEnvironmentCode(input.Code)
	id := strings.TrimSpace(input.ID)
	if id == "" && code != "" {
		id = "env-" + code
	}
	model := EnvironmentModel{
		ID:              id,
		Name:            strings.TrimSpace(input.Name),
		Code:            code,
		Type:            strings.TrimSpace(input.Type),
		NetworkMode:     strings.TrimSpace(input.NetworkMode),
		ClusterID:       strings.TrimSpace(input.ClusterID),
		Namespace:       strings.TrimSpace(input.Namespace),
		RegistryID:      strings.TrimSpace(input.RegistryID),
		RegistryProject: strings.TrimSpace(input.RegistryProject),
		JenkinsID:       strings.TrimSpace(input.JenkinsID),
		JenkinsView:     strings.TrimSpace(input.JenkinsView),
		Status:          fallbackString(strings.TrimSpace(input.Status), "HEALTHY"),
	}
	if model.ID == "" || model.Name == "" || model.Code == "" || model.Type == "" || model.NetworkMode == "" {
		return domain.Environment{}, errors.New("missing required fields")
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.Environment{}, err
	}
	return toDomainEnvironment(model, ""), nil
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
	if envType := strings.TrimSpace(input.Type); envType != "" {
		model.Type = envType
	}
	if networkMode := strings.TrimSpace(input.NetworkMode); networkMode != "" {
		model.NetworkMode = networkMode
	}
	model.ClusterID = strings.TrimSpace(input.ClusterID)
	model.Namespace = strings.TrimSpace(input.Namespace)
	model.RegistryID = strings.TrimSpace(input.RegistryID)
	model.RegistryProject = strings.TrimSpace(input.RegistryProject)
	model.JenkinsID = strings.TrimSpace(input.JenkinsID)
	model.JenkinsView = strings.TrimSpace(input.JenkinsView)
	if status := strings.TrimSpace(input.Status); status != "" {
		model.Status = status
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Environment{}, false, err
	}
	return toDomainEnvironment(model, s.getAgentStatus(model.AgentID)), true, nil
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
	return toDomainEnvironment(model, s.getAgentStatus(model.AgentID)), true, nil
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

func (s *DatabaseStore) UpdateHarborRegistryProbe(id string, status string, message string, projects []string, checkedAt time.Time) (domain.HarborRegistry, bool, error) {
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

func (s *DatabaseStore) UpdateJenkinsInstanceProbe(id string, status string, message string, views []string, jobs []string, checkedAt time.Time) (domain.JenkinsInstance, bool, error) {
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

func (s *DatabaseStore) UpsertAgent(id string, environmentID string, version string, capabilities []string, status string) (domain.Agent, bool) {
	var environment EnvironmentModel
	if err := s.db.Where("id = ?", environmentID).Take(&environment).Error; err != nil {
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
			EnvironmentID:   environmentID,
			Version:         fallbackString(version, "dev"),
			Status:          fallbackString(status, "ONLINE"),
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
	model.LastHeartbeatAt = &now
	if saveErr := s.db.Save(&model).Error; saveErr != nil {
		return domain.Agent{}, false
	}
	if updateEnvErr := s.rebindEnvironmentAgent(previousEnvironmentID, environmentID, id); updateEnvErr != nil {
		return domain.Agent{}, false
	}
	return s.GetAgent(id)
}

func (s *DatabaseStore) UpdateAgentHeartbeat(id string, environmentID string, version string, capabilities []string) (domain.Agent, bool) {
	var model AgentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	now := time.Now()
	previousEnvironmentID := model.EnvironmentID
	model.Status = "ONLINE"
	model.LastHeartbeatAt = &now
	if environmentID != "" {
		model.EnvironmentID = environmentID
	}
	if version != "" {
		model.Version = version
	}
	if len(capabilities) > 0 {
		model.Capabilities = append([]string(nil), capabilities...)
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Agent{}, false
	}
	if model.EnvironmentID != "" {
		if err := s.rebindEnvironmentAgent(previousEnvironmentID, model.EnvironmentID, id); err != nil {
			return domain.Agent{}, false
		}
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

func (s *DatabaseStore) GetCurrentUser() domain.CurrentUser {
	return s.mock.GetCurrentUser()
}

func (s *DatabaseStore) ListUsers(query string) []domain.User {
	return s.mock.ListUsers(query)
}

func (s *DatabaseStore) ListRoles(query string) []domain.Role {
	return s.mock.ListRoles(query)
}

func (s *DatabaseStore) ListPermissions(query string) []domain.EnvironmentPermission {
	return s.mock.ListPermissions(query)
}

func (s *DatabaseStore) ListChangelog(query string) []domain.ChangelogEntry {
	return s.mock.ListChangelog(query)
}

func (s *DatabaseStore) CreateBaseline(sourceEnvironmentID string, name string, purpose string) (domain.BaselineDetail, error) {
	return s.mock.CreateBaseline(sourceEnvironmentID, name, purpose)
}

func (s *DatabaseStore) ListBaselines(query string) []domain.Baseline {
	return s.mock.ListBaselines(query)
}

func (s *DatabaseStore) GetBaselineDetail(id string) (domain.BaselineDetail, bool) {
	return s.mock.GetBaselineDetail(id)
}

func (s *DatabaseStore) LockBaseline(id string) (domain.BaselineDetail, bool) {
	return s.mock.LockBaseline(id)
}

func (s *DatabaseStore) GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool) {
	return s.mock.GetDiffResult(id, targetEnvironmentID)
}

func (s *DatabaseStore) ListReleaseSourceServices(query string) []domain.ReleaseSourceService {
	var models []ServiceModel
	db := s.db.Order("name asc")
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
		ImageRepository:      strings.TrimSpace(input.ImageRepository),
		ImageTag:             strings.TrimSpace(input.ImageTag),
		ImageDigest:          strings.TrimSpace(input.ImageDigest),
		TargetEnvironmentID:  strings.TrimSpace(input.TargetEnvironmentID),
		AgentID:              strings.TrimSpace(input.AgentID),
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
	}, true
}

func (s *DatabaseStore) ListDeployTasks(query string) []domain.DeployTask {
	return s.mock.ListDeployTasks(query)
}

func (s *DatabaseStore) GetDeployDetail(id string) (domain.DeployDetail, bool) {
	return s.mock.GetDeployDetail(id)
}

func (s *DatabaseStore) HasEnvironmentAction(environmentID string, action string) bool {
	return s.mock.HasEnvironmentAction(environmentID, action)
}

func toDomainEnvironment(model EnvironmentModel, boundAgentStatus string) domain.Environment {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	agentStatus := "UNBOUND"
	if model.AgentID != "" {
		agentStatus = fallbackString(boundAgentStatus, "OFFLINE")
	}
	return domain.Environment{
		ID:              model.ID,
		Name:            model.Name,
		Code:            model.Code,
		Type:            model.Type,
		NetworkMode:     model.NetworkMode,
		ClusterID:       model.ClusterID,
		Namespace:       model.Namespace,
		RegistryID:      model.RegistryID,
		RegistryProject: model.RegistryProject,
		JenkinsID:       model.JenkinsID,
		JenkinsView:     model.JenkinsView,
		Status:          model.Status,
		AgentStatus:     agentStatus,
		LastCheckAt:     lastCheckAt,
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
		Capabilities:    model.Capabilities,
		LastHeartbeatAt: lastHeartbeatAt,
		CurrentTaskID:   currentTaskID,
	}
}

func toDomainReleaseSourceService(model ServiceModel) domain.ReleaseSourceService {
	return domain.ReleaseSourceService{
		ServiceID:       model.ID,
		ServiceName:     model.Name,
		Namespace:       model.Namespace,
		WorkloadName:    model.WorkloadName,
		WorkloadType:    model.WorkloadType,
		ImageRepository: model.ImageRepository,
		Tags:            []domain.ReleaseImageTag{},
		Publishable:     false,
	}
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
		ImageRepository:       model.ImageRepository,
		ImageTag:              model.ImageTag,
		ImageDigest:           model.ImageDigest,
		TargetEnvironmentName: fallbackString(environmentName, model.TargetEnvironmentID),
		Status:                model.Status,
		Progress:              model.Progress,
		AgentName:             fallbackString(agentName, model.AgentID),
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

func normalizeEnvironmentCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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
