package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/repository"
)

type Handler struct {
	repo *repository.MockRepository
}

func NewHandler(repo *repository.MockRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) ListEnvironments(c *gin.Context) {
	OK(c, paginate(h.repo.ListEnvironments(c.Query("keyword")), c))
}

func (h *Handler) CheckEnvironment(c *gin.Context) {
	OK(c, gin.H{
		"environmentId": c.Param("id"),
		"status":        "HEALTHY",
		"checkedAt":     time.Now().Format(time.RFC3339),
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

func (h *Handler) CreateBaseline(c *gin.Context) {
	Created(c, gin.H{
		"id":        "BL-20260607-MOCK",
		"status":    "DRAFT",
		"createdAt": time.Now().Format(time.RFC3339),
	})
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
	OK(c, gin.H{
		"id":     c.Param("id"),
		"status": "LOCKED",
	})
}

func (h *Handler) CompareBaseline(c *gin.Context) {
	result, ok := h.repo.GetDiffResult(c.Param("id"))
	if !ok {
		NotFound(c, "baseline not found")
		return
	}
	OK(c, result)
}

func (h *Handler) CreateRelease(c *gin.Context) {
	Created(c, gin.H{
		"id":        "REL-20260607-MOCK",
		"status":    "PENDING_CONFIRM",
		"createdAt": time.Now().Format(time.RFC3339),
	})
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
	OK(c, gin.H{
		"id":     c.Param("id"),
		"status": "RUNNING",
		"action": "retry",
	})
}

func (h *Handler) RollbackRelease(c *gin.Context) {
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
	Created(c, gin.H{
		"id":        "DEP-20260607-MOCK",
		"status":    "PENDING",
		"createdAt": time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) GetDeployTask(c *gin.Context) {
	detail, ok := h.repo.GetDeployDetail(c.Param("id"))
	if !ok {
		NotFound(c, "deploy task not found")
		return
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
	OK(c, gin.H{
		"taskId": c.Param("id"),
		"stepId": c.Param("stepId"),
		"action": action,
		"status": status,
	})
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
