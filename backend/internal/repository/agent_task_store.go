package repository

import (
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ops-release-platform/backend/internal/agent"
)

type DatabaseProtocolStore struct {
	db *gorm.DB
}

func NewDatabaseProtocolStore(db *gorm.DB) *DatabaseProtocolStore {
	return &DatabaseProtocolStore{db: db}
}

func (s *DatabaseProtocolStore) Enqueue(task agent.Task) agent.ProtocolTask {
	now := time.Now()
	model := AgentTaskModel{
		ID:            task.ID,
		Type:          task.Type,
		Action:        task.Action,
		Status:        "PENDING",
		Step:          "queued",
		AgentID:       task.AgentID,
		EnvironmentID: task.EnvironmentID,
		Payload:       copyPayload(task.Payload),
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     now,
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	if model.Payload == nil {
		model.Payload = map[string]string{}
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"type":           model.Type,
			"action":         model.Action,
			"status":         model.Status,
			"step":           model.Step,
			"agent_id":       model.AgentID,
			"environment_id": model.EnvironmentID,
			"lease_id":       "",
			"lease_until":    nil,
			"payload":        model.Payload,
			"step_url":       "",
			"log_url":        "",
			"result_url":     "",
			"updated_at":     now,
		}),
	}).Create(&model).Error
	if err != nil {
		return agent.ProtocolTask{}
	}
	return toProtocolTask(model, nil)
}

func (s *DatabaseProtocolStore) Pull(agentID string) (agent.ProtocolTask, bool) {
	result := s.Lease(agent.LeaseRequest{AgentID: agentID, LeaseSeconds: 300})
	if !result.Leased || result.Task == nil {
		return agent.ProtocolTask{}, false
	}
	return *result.Task, true
}

func (s *DatabaseProtocolStore) Lease(request agent.LeaseRequest) agent.LeaseResult {
	if request.LeaseSeconds <= 0 {
		request.LeaseSeconds = 300
	}
	if request.MaxTasks <= 0 {
		request.MaxTasks = 1
	}
	if request.MaxTasks > 1 {
		request.MaxTasks = 1
	}

	now := time.Now()
	var leasedTask agent.ProtocolTask
	var result agent.LeaseResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if hasRunningLease(tx, request.AgentID, now) {
			result = agent.LeaseResult{Leased: false, Message: "agent already has a running leased task", Task: nil}
			return nil
		}

		expiredTasks := make([]AgentTaskModel, 0)
		if err := tx.Where("status = ? AND lease_until IS NOT NULL AND lease_until <= ?", "RUNNING", now).Find(&expiredTasks).Error; err != nil {
			return err
		}
		for _, task := range expiredTasks {
			if err := tx.Model(&AgentTaskModel{}).Where("id = ?", task.ID).Updates(map[string]any{
				"status":      "PENDING",
				"step":        "lease-expired",
				"lease_id":    "",
				"lease_until": nil,
				"updated_at":  now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&AgentTaskLogModel{
				TaskID: task.ID,
				Line:   "previous lease expired; task returned to pending queue",
			}).Error; err != nil {
				return err
			}
		}

		var tasks []AgentTaskModel
		query := tx.Where("status = ?", "PENDING")
		query = query.Where("agent_id = ? OR agent_id = ?", "", request.AgentID)
		if request.EnvironmentID != "" {
			query = query.Where("environment_id = ? OR environment_id = ?", "", request.EnvironmentID)
		}
		if err := query.Order("created_at asc").Limit(1).Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			result = agent.LeaseResult{Leased: false, Task: nil}
			return nil
		}
		task := tasks[0]

		task.AgentID = request.AgentID
		task.Status = "RUNNING"
		task.Step = "leased"
		task.LeaseID = "lease-" + task.ID
		leaseUntil := now.Add(time.Duration(request.LeaseSeconds) * time.Second)
		task.LeaseUntil = &leaseUntil
		task.StepURL = request.CallbackBase + "/api/agent-tasks/" + task.ID + "/steps"
		task.LogURL = request.CallbackBase + "/api/agent-tasks/" + task.ID + "/logs"
		task.ResultURL = request.CallbackBase + "/api/agent-tasks/" + task.ID + "/result"
		task.UpdatedAt = now
		if err := tx.Save(&task).Error; err != nil {
			return err
		}
		leasedTask = toProtocolTask(task, nil)
		result = agent.LeaseResult{Leased: true, LeaseID: task.LeaseID, Task: &leasedTask}
		return nil
	})
	if err != nil {
		return agent.LeaseResult{Leased: false, Message: "agent task lease failed", Task: nil}
	}
	return result
}

func (s *DatabaseProtocolStore) ReportStep(taskID string, step string, status string) (agent.ProtocolTask, bool) {
	var model AgentTaskModel
	if err := s.db.Where("id = ?", taskID).Take(&model).Error; err != nil {
		return agent.ProtocolTask{}, false
	}
	model.Step = step
	model.Status = status
	model.UpdatedAt = time.Now()
	if err := s.db.Save(&model).Error; err != nil {
		return agent.ProtocolTask{}, false
	}
	return toProtocolTask(model, nil), true
}

func (s *DatabaseProtocolStore) AppendLog(taskID string, line string) (agent.ProtocolTask, bool) {
	var model AgentTaskModel
	if err := s.db.Where("id = ?", taskID).Take(&model).Error; err != nil {
		return agent.ProtocolTask{}, false
	}
	model.UpdatedAt = time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		return tx.Create(&AgentTaskLogModel{TaskID: taskID, Line: line}).Error
	}); err != nil {
		return agent.ProtocolTask{}, false
	}
	logs := s.logs(taskID)
	return toProtocolTask(model, logs), true
}

func (s *DatabaseProtocolStore) ReportResult(taskID string, status string, message string) (agent.ProtocolTask, bool) {
	var model AgentTaskModel
	if err := s.db.Where("id = ?", taskID).Take(&model).Error; err != nil {
		return agent.ProtocolTask{}, false
	}
	model.Status = status
	model.Step = "finished"
	model.UpdatedAt = time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&model).Error; err != nil {
			return err
		}
		if message != "" {
			return tx.Create(&AgentTaskLogModel{TaskID: taskID, Line: message}).Error
		}
		return nil
	}); err != nil {
		return agent.ProtocolTask{}, false
	}
	return toProtocolTask(model, s.logs(taskID)), true
}

func (s *DatabaseProtocolStore) Status(taskID string) (map[string]string, []string, bool) {
	var model AgentTaskModel
	if err := s.db.Where("id = ?", taskID).Take(&model).Error; err != nil {
		return nil, nil, false
	}
	status := map[string]string{
		"taskId":        model.ID,
		"type":          model.Type,
		"action":        model.Action,
		"status":        model.Status,
		"step":          model.Step,
		"agentId":       model.AgentID,
		"environmentId": model.EnvironmentID,
		"leaseId":       model.LeaseID,
		"updatedAt":     model.UpdatedAt.Format(time.RFC3339),
	}
	return status, s.logs(taskID), true
}

func hasRunningLease(tx *gorm.DB, agentID string, now time.Time) bool {
	var count int64
	tx.Model(&AgentTaskModel{}).
		Where("agent_id = ? AND status = ?", agentID, "RUNNING").
		Where("lease_until IS NULL OR lease_until > ?", now).
		Count(&count)
	return count > 0
}

func (s *DatabaseProtocolStore) logs(taskID string) []string {
	var models []AgentTaskLogModel
	if err := s.db.Where("task_id = ?", taskID).Order("created_at asc, id asc").Find(&models).Error; err != nil {
		return []string{}
	}
	logs := make([]string, 0, len(models))
	for _, model := range models {
		logs = append(logs, model.Line)
	}
	return logs
}

func toProtocolTask(model AgentTaskModel, logs []string) agent.ProtocolTask {
	task := agent.ProtocolTask{
		ID:            model.ID,
		Type:          model.Type,
		Action:        model.Action,
		Status:        model.Status,
		Step:          model.Step,
		AgentID:       model.AgentID,
		EnvironmentID: model.EnvironmentID,
		LeaseID:       model.LeaseID,
		Payload:       copyPayload(model.Payload),
		Callback: agent.Callback{
			StepURL:   model.StepURL,
			LogURL:    model.LogURL,
			ResultURL: model.ResultURL,
		},
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
		Logs:      append([]string(nil), logs...),
	}
	if model.LeaseUntil != nil {
		task.LeaseUntil = *model.LeaseUntil
	}
	return task
}

func copyPayload(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		output[key] = input[key]
	}
	return output
}
