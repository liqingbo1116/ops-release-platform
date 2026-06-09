package repository

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ops-release-platform/backend/internal/domain"
)

//go:embed mockdata/*.json
var mockFiles embed.FS

type MockRepository struct {
	environments    []domain.Environment
	agents          []domain.Agent
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

func NewMockRepository() (*MockRepository, error) {
	repo := &MockRepository{}
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
	return repo, nil
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
	return filter(r.environments, query, func(item domain.Environment) string {
		return item.ID + " " + item.Name + " " + item.Code + " " + item.Type + " " + item.NetworkMode + " " + item.Status
	})
}

func (r *MockRepository) ListAgents(query string) []domain.Agent {
	return filter(r.agents, query, func(item domain.Agent) string {
		return item.ID + " " + item.Name + " " + item.EnvironmentName + " " + strings.Join(item.Capabilities, " ") + " " + item.Status
	})
}

func (r *MockRepository) GetAgent(id string) (domain.Agent, bool) {
	for _, agent := range r.agents {
		if agent.ID == id {
			return agent, true
		}
	}
	return domain.Agent{}, false
}

func (r *MockRepository) UpdateAgentHeartbeat(id string, version string, capabilities []string) (domain.Agent, bool) {
	for index := range r.agents {
		if r.agents[index].ID != id {
			continue
		}
		r.agents[index].Status = "ONLINE"
		r.agents[index].LastHeartbeatAt = time.Now().Format(time.RFC3339)
		if version != "" {
			r.agents[index].Version = version
		}
		if len(capabilities) > 0 {
			r.agents[index].Capabilities = capabilities
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
