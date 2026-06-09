package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/repository"
	"ops-release-platform/backend/internal/service"
)

type Handler struct {
	repo         *repository.MockRepository
	queue        *agent.Queue
	protocol     *agent.ProtocolStore
	integrations integration.Suite
	service      *service.ReleaseCreator
}

func NewHandler(repo *repository.MockRepository, queue *agent.Queue, protocol *agent.ProtocolStore, integrations integration.Suite) *Handler {
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

func (h *Handler) CheckEnvironment(c *gin.Context) {
	environmentID := c.Param("id")
	checks := make([]integration.IntegrationCheck, 0, 2)
	if h.integrations.Kubernetes != nil {
		check, err := h.integrations.Kubernetes.CheckConnection(c.Request.Context(), environmentID)
		if err != nil {
			BadRequest(c, "kubernetes check failed")
			return
		}
		checks = append(checks, check)
	}
	if h.integrations.Registry != nil {
		check, err := h.integrations.Registry.CheckConnection(c.Request.Context(), environmentID)
		if err != nil {
			BadRequest(c, "registry check failed")
			return
		}
		checks = append(checks, check)
	}
	OK(c, gin.H{
		"environmentId": environmentID,
		"status":        "HEALTHY",
		"checkedAt":     time.Now().Format(time.RFC3339),
		"checks":        checks,
	})
}

func (h *Handler) ListAgents(c *gin.Context) {
	OK(c, paginate(h.repo.ListAgents(c.Query("keyword")), c))
}

func (h *Handler) CreateAgentRegisterToken(c *gin.Context) {
	token := "agt_7f92c1b8_20260607"
	Created(c, gin.H{
		"token":     token,
		"expiresAt": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
		"installCommand": "curl -fsSL https://platform.local/agent/install.sh | bash -s -- --token " +
			token + " --server https://platform.local",
	})
}

func (h *Handler) AgentHeartbeat(c *gin.Context) {
	var request struct {
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "invalid heartbeat request")
		return
	}
	agentItem, ok := h.repo.UpdateAgentHeartbeat(c.Param("id"), request.Version, request.Capabilities)
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
		BadRequest(c, "agent does not belong to target environment")
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
		detail, ok = h.repo.GetReleaseDetail("")
		if !ok {
			NotFound(c, "release not found")
			return
		}
		detail.ID = c.Param("id")
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
