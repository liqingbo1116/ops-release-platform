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
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Action        string            `json:"action"`
	Status        string            `json:"status"`
	Step          string            `json:"step"`
	AgentID       string            `json:"agentId,omitempty"`
	EnvironmentID string            `json:"environmentId,omitempty"`
	LeaseID       string            `json:"leaseId,omitempty"`
	LeaseUntil    time.Time         `json:"leaseUntil,omitempty"`
	Payload       map[string]string `json:"payload,omitempty"`
	Callback      Callback          `json:"callback,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	Logs          []string          `json:"-"`
}

type Callback struct {
	StepURL   string `json:"stepUrl,omitempty"`
	LogURL    string `json:"logUrl,omitempty"`
	ResultURL string `json:"resultUrl,omitempty"`
}

type LeaseRequest struct {
	AgentID       string
	EnvironmentID string
	MaxTasks      int
	LeaseSeconds  int
	CallbackBase  string
}

type LeaseResult struct {
	Leased  bool          `json:"leased"`
	LeaseID string        `json:"leaseId,omitempty"`
	Message string        `json:"message,omitempty"`
	Task    *ProtocolTask `json:"task"`
}

func NewProtocolStore() *ProtocolStore {
	return &ProtocolStore{tasks: make(map[string]*ProtocolTask)}
}

func (s *ProtocolStore) Enqueue(task Task) ProtocolTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	protocolTask := &ProtocolTask{
		ID:            task.ID,
		Type:          task.Type,
		Action:        task.Action,
		Status:        "PENDING",
		Step:          "queued",
		AgentID:       task.AgentID,
		EnvironmentID: task.EnvironmentID,
		Payload:       task.Payload,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     now,
	}
	if protocolTask.CreatedAt.IsZero() {
		protocolTask.CreatedAt = now
	}
	s.tasks[task.ID] = protocolTask
	return *protocolTask
}

func (s *ProtocolStore) Pull(agentID string) (ProtocolTask, bool) {
	result := s.Lease(LeaseRequest{AgentID: agentID, LeaseSeconds: 300})
	if !result.Leased || result.Task == nil {
		return ProtocolTask{}, false
	}
	return *result.Task, true
}

func (s *ProtocolStore) Lease(request LeaseRequest) LeaseResult {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	pending := make([]*ProtocolTask, 0)
	for _, task := range s.tasks {
		if task.AgentID != "" && task.AgentID != request.AgentID {
			continue
		}
		if task.EnvironmentID != "" && request.EnvironmentID != "" && task.EnvironmentID != request.EnvironmentID {
			continue
		}
		if task.Status == "RUNNING" {
			if task.LeaseUntil.IsZero() || task.LeaseUntil.After(now) {
				if task.AgentID == request.AgentID {
					return LeaseResult{Leased: false, Message: "agent already has a running leased task", Task: nil}
				}
				continue
			}
			task.Status = "PENDING"
			task.Step = "lease-expired"
			task.LeaseID = ""
			task.LeaseUntil = time.Time{}
			task.Logs = append(task.Logs, "previous lease expired; task returned to pending queue")
			task.UpdatedAt = now
		}
		if task.Status == "PENDING" {
			if task.AgentID != "" && task.AgentID != request.AgentID {
				continue
			}
			pending = append(pending, task)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt.Before(pending[j].CreatedAt)
	})
	if len(pending) == 0 {
		return LeaseResult{Leased: false, Task: nil}
	}

	task := pending[0]
	task.AgentID = request.AgentID
	task.Status = "RUNNING"
	task.Step = "leased"
	task.LeaseID = "lease-" + task.ID
	task.LeaseUntil = now.Add(time.Duration(request.LeaseSeconds) * time.Second)
	task.Callback = Callback{
		StepURL:   request.CallbackBase + "/api/agent-tasks/" + task.ID + "/steps",
		LogURL:    request.CallbackBase + "/api/agent-tasks/" + task.ID + "/logs",
		ResultURL: request.CallbackBase + "/api/agent-tasks/" + task.ID + "/result",
	}
	task.UpdatedAt = now
	copyTask := *task
	return LeaseResult{Leased: true, LeaseID: task.LeaseID, Task: &copyTask}
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
		"taskId":        task.ID,
		"type":          task.Type,
		"action":        task.Action,
		"status":        task.Status,
		"step":          task.Step,
		"agentId":       task.AgentID,
		"environmentId": task.EnvironmentID,
		"leaseId":       task.LeaseID,
		"updatedAt":     task.UpdatedAt.Format(time.RFC3339),
	}
	logs := append([]string(nil), task.Logs...)
	return status, logs, true
}
