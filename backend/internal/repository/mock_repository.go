package repository

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"

	"ops-release-platform/backend/internal/domain"
)

//go:embed mockdata/*.json
var mockFiles embed.FS

type MockRepository struct {
	environments   []domain.Environment
	agents         []domain.Agent
	baselines      []domain.Baseline
	baselineDetail domain.BaselineDetail
	diffResult     domain.DiffResult
	releaseDetail  domain.ReleaseDetail
	deployTasks    []domain.DeployTask
	deployDetail   domain.DeployDetail
}

func NewMockRepository() (*MockRepository, error) {
	repo := &MockRepository{}
	loaders := []func() error{
		func() error { return loadJSON("mockdata/environments.json", &repo.environments) },
		func() error { return loadJSON("mockdata/agents.json", &repo.agents) },
		func() error { return loadJSON("mockdata/baselines.json", &repo.baselines) },
		func() error { return loadJSON("mockdata/baseline-detail.json", &repo.baselineDetail) },
		func() error { return loadJSON("mockdata/diff-result.json", &repo.diffResult) },
		func() error { return loadJSON("mockdata/release-detail.json", &repo.releaseDetail) },
		func() error { return loadJSON("mockdata/deploy-tasks.json", &repo.deployTasks) },
		func() error { return loadJSON("mockdata/deploy-detail.json", &repo.deployDetail) },
	}
	for _, load := range loaders {
		if err := load(); err != nil {
			return nil, err
		}
	}
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

func (r *MockRepository) ListBaselines(query string) []domain.Baseline {
	return filter(r.baselines, query, func(item domain.Baseline) string {
		return item.ID + " " + item.Name + " " + item.SourceEnvironmentName + " " + item.Purpose + " " + item.Status
	})
}

func (r *MockRepository) GetBaselineDetail(id string) (domain.BaselineDetail, bool) {
	if id != "" && id != r.baselineDetail.ID {
		return domain.BaselineDetail{}, false
	}
	return r.baselineDetail, true
}

func (r *MockRepository) GetDiffResult(id string) (domain.DiffResult, bool) {
	if id != "" && id != r.diffResult.SourceBaselineID {
		return domain.DiffResult{}, false
	}
	return r.diffResult, true
}

func (r *MockRepository) GetReleaseDetail(id string) (domain.ReleaseDetail, bool) {
	if id != "" && id != r.releaseDetail.ID {
		return domain.ReleaseDetail{}, false
	}
	return r.releaseDetail, true
}

func (r *MockRepository) ListDeployTasks(query string) []domain.DeployTask {
	return filter(r.deployTasks, query, func(item domain.DeployTask) string {
		return item.ID + " " + item.ProductName + " " + item.TargetEnvironmentName + " " + item.Source + " " + item.Status
	})
}

func (r *MockRepository) GetDeployDetail(id string) (domain.DeployDetail, bool) {
	if id != "" && id != r.deployDetail.ID {
		return domain.DeployDetail{}, false
	}
	return r.deployDetail, true
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
