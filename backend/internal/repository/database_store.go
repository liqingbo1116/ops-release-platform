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
	items := make([]domain.Environment, 0, len(models))
	for _, model := range models {
		items = append(items, toDomainEnvironment(model))
	}
	return items
}

func (s *DatabaseStore) GetEnvironment(id string) (domain.Environment, bool) {
	var model EnvironmentModel
	if err := s.db.Where("id = ?", id).Take(&model).Error; err != nil {
		return domain.Environment{}, false
	}
	return toDomainEnvironment(model), true
}

func (s *DatabaseStore) CreateEnvironment(input domain.Environment) (domain.Environment, error) {
	model := EnvironmentModel{
		ID:          strings.TrimSpace(input.ID),
		Name:        strings.TrimSpace(input.Name),
		Code:        strings.TrimSpace(input.Code),
		Type:        strings.TrimSpace(input.Type),
		NetworkMode: strings.TrimSpace(input.NetworkMode),
		Status:      fallbackString(strings.TrimSpace(input.Status), "HEALTHY"),
	}
	if model.ID == "" || model.Name == "" || model.Code == "" || model.Type == "" || model.NetworkMode == "" {
		return domain.Environment{}, errors.New("missing required fields")
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.Environment{}, err
	}
	return toDomainEnvironment(model), nil
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
	if code := strings.TrimSpace(input.Code); code != "" {
		model.Code = code
	}
	if envType := strings.TrimSpace(input.Type); envType != "" {
		model.Type = envType
	}
	if networkMode := strings.TrimSpace(input.NetworkMode); networkMode != "" {
		model.NetworkMode = networkMode
	}
	if status := strings.TrimSpace(input.Status); status != "" {
		model.Status = status
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Environment{}, false, err
	}
	return toDomainEnvironment(model), true, nil
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

func (s *DatabaseStore) ListReleases(query string) []domain.ReleaseOrder {
	return s.mock.ListReleases(query)
}

func (s *DatabaseStore) GetReleaseDetail(id string) (domain.ReleaseDetail, bool) {
	return s.mock.GetReleaseDetail(id)
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

func toDomainEnvironment(model EnvironmentModel) domain.Environment {
	lastCheckAt := ""
	if model.LastCheckAt != nil {
		lastCheckAt = model.LastCheckAt.Format(time.RFC3339)
	}
	agentStatus := "UNBOUND"
	if model.AgentID != "" {
		agentStatus = "ONLINE"
	}
	return domain.Environment{
		ID:          model.ID,
		Name:        model.Name,
		Code:        model.Code,
		Type:        model.Type,
		NetworkMode: model.NetworkMode,
		Status:      model.Status,
		AgentStatus: agentStatus,
		LastCheckAt: lastCheckAt,
	}
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
		Status:          model.Status,
		Capabilities:    model.Capabilities,
		LastHeartbeatAt: lastHeartbeatAt,
		CurrentTaskID:   currentTaskID,
	}
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
