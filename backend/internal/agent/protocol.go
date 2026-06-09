package agent

import (
	"sort"
	"sync"
	"time"
)

type ProtocolStore struct {
	mu    sync.Mutex
	tasks map[string]*ProtocolTask
}

type ProtocolTask struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	Step      string            `json:"step"`
	AgentID   string            `json:"agentId,omitempty"`
	Payload   map[string]string `json:"payload,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Logs      []string          `json:"-"`
}

func NewProtocolStore() *ProtocolStore {
	return &ProtocolStore{tasks: make(map[string]*ProtocolTask)}
}

func (s *ProtocolStore) Enqueue(task Task) ProtocolTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	protocolTask := &ProtocolTask{
		ID:        task.ID,
		Type:      task.Type,
		Action:    task.Action,
		Status:    "PENDING",
		Step:      "queued",
		CreatedAt: task.CreatedAt,
		UpdatedAt: now,
	}
	if protocolTask.CreatedAt.IsZero() {
		protocolTask.CreatedAt = now
	}
	s.tasks[task.ID] = protocolTask
	return *protocolTask
}

func (s *ProtocolStore) Pull(agentID string) (ProtocolTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending := make([]*ProtocolTask, 0)
	for _, task := range s.tasks {
		if task.Status == "PENDING" {
			pending = append(pending, task)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt.Before(pending[j].CreatedAt)
	})
	if len(pending) == 0 {
		return ProtocolTask{}, false
	}

	task := pending[0]
	task.AgentID = agentID
	task.Status = "RUNNING"
	task.Step = "pulled"
	task.UpdatedAt = time.Now()
	return *task, true
}

func (s *ProtocolStore) ReportStep(taskID string, step string, status string) (ProtocolTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ProtocolTask{}, false
	}
	task.Step = step
	task.Status = status
	task.UpdatedAt = time.Now()
	return *task, true
}

func (s *ProtocolStore) AppendLog(taskID string, line string) (ProtocolTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ProtocolTask{}, false
	}
	task.Logs = append(task.Logs, line)
	task.UpdatedAt = time.Now()
	return *task, true
}

func (s *ProtocolStore) ReportResult(taskID string, status string, message string) (ProtocolTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return ProtocolTask{}, false
	}
	task.Status = status
	task.Step = "finished"
	if message != "" {
		task.Logs = append(task.Logs, message)
	}
	task.UpdatedAt = time.Now()
	return *task, true
}

func (s *ProtocolStore) Status(taskID string) (map[string]string, []string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, nil, false
	}
	status := map[string]string{
		"taskId":    task.ID,
		"type":      task.Type,
		"action":    task.Action,
		"status":    task.Status,
		"step":      task.Step,
		"agentId":   task.AgentID,
		"updatedAt": task.UpdatedAt.Format(time.RFC3339),
	}
	logs := append([]string(nil), task.Logs...)
	return status, logs, true
}
