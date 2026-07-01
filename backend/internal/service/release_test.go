package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
)

func TestCreateReleaseWithLocalHarborImageUsesRegistry(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	result, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "LOCAL_HARBOR_IMAGE",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
		Image: ReleaseImage{
			Repository: "harbor.local/project-x/user-service",
			Tag:        "20260607-a1b2c3",
		},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if result.ExecutionMode != "AGENT_IMAGE_SYNC" {
		t.Fatalf("expected AGENT_IMAGE_SYNC, got %s", result.ExecutionMode)
	}
	if result.AgentTaskID == "" || result.Status != "PENDING_IMAGE_SYNC" {
		t.Fatalf("expected image sync task metadata, got %+v", result)
	}
}

func TestCreateReleaseWithLocalJenkinsJobDoesNotEnqueueAgentTask(t *testing.T) {
	var enqueued bool
	jenkins := &recordingJenkinsAdapter{
		result: integration.BuildResult{
			BuildID: "42",
			Status:  "BUILDING",
			URL:     "http://jenkins.local/view/local-release/job/local-service-release/42/",
		},
	}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "env-local-prod",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "local-release",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{
					Name: "local-service-release",
					View: "local-release",
					URL:  "http://jenkins.local/view/local-release/job/local-service-release/",
				},
			},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, func(ctx context.Context, id string, taskType string, action string, agentID string, environmentID string) {
		enqueued = true
	})

	result, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-local-prod",
		Jenkins: ReleaseJenkins{
			JobName: "local-service-release",
			Branch:  "main",
		},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if result.ExecutionMode != "JENKINS_ONLY" {
		t.Fatalf("expected JENKINS_ONLY, got %s", result.ExecutionMode)
	}
	if result.Status != "JENKINS_TRIGGERING" || result.BuildStatus != "TRIGGERING" || result.BuildID != "" {
		t.Fatalf("expected immediate Jenkins triggering result, got %+v", result)
	}
	waitForJenkinsRequest(t, jenkins)
	if jenkins.request.JobName != "local-service-release" || jenkins.request.JobURL == "" {
		t.Fatalf("expected bound pipeline to be triggered, got %+v", jenkins.request)
	}
	if result.AgentTaskID != "" {
		t.Fatalf("expected no agent task id, got %s", result.AgentTaskID)
	}
	if enqueued {
		t.Fatal("expected local Jenkins release not to enqueue agent task")
	}
}

func TestCreateReleaseWithRealJenkinsTriggersBoundPipeline(t *testing.T) {
	jenkins := &recordingJenkinsAdapter{
		result: integration.BuildResult{
			BuildID: "88",
			Status:  "BUILDING",
			URL:     "http://jenkins.local/job/release-view/job/user-service-release/88/",
		},
		parameters: []domain.JenkinsPipelineParameter{
			{Name: "BRANCH", Type: "StringParameterDefinition", Required: true},
		},
	}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-remote",
			Name:        "远程产品",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:       "jenkins-local",
			URL:      "http://jenkins.local",
			Username: "builder",
			Token:    "secret",
			Pipelines: []domain.JenkinsPipeline{
				{
					Name: "user-service-release",
					View: "release-view",
					URL:  "http://jenkins.local/view/release-view/job/user-service-release/",
				},
			},
		},
		managedServices: []domain.ManagedService{
			{ID: "svc-user", Name: "user-service"},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)
	creator.now = func() time.Time { return time.Date(2026, 6, 26, 11, 0, 0, 0, time.UTC) }

	result, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-remote",
		ServiceIDs:          []string{"svc-user"},
		Jenkins: ReleaseJenkins{
			JobName: "user-service-release",
			Parameters: map[string]string{
				"BRANCH": "release/20260626",
			},
		},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if result.ExecutionMode != "JENKINS_ONLY" || result.BuildID != "" || result.BuildStatus != "TRIGGERING" || result.Status != "JENKINS_TRIGGERING" {
		t.Fatalf("unexpected release result: %+v", result)
	}
	waitForJenkinsRequest(t, jenkins)
	if jenkins.request.JenkinsURL != "http://jenkins.local" || jenkins.request.Username != "builder" || jenkins.request.Token != "secret" {
		t.Fatalf("expected Jenkins connection details, got %+v", jenkins.request)
	}
	if jenkins.request.JobName != "user-service-release" || jenkins.request.JobURL != "http://jenkins.local/view/release-view/job/user-service-release/" {
		t.Fatalf("expected bound pipeline, got %+v", jenkins.request)
	}
	if jenkins.request.Parameters["BRANCH"] != "release/20260626" {
		t.Fatalf("expected release parameter to be forwarded, got %+v", jenkins.request.Parameters)
	}
	if !jenkins.request.Parameterized {
		t.Fatalf("expected parameterized Jenkins build request")
	}
	if repo.created.ID == "" || repo.created.JenkinsID != "jenkins-local" || repo.created.BuildID != "" || repo.created.BuildStatus != "TRIGGERING" {
		t.Fatalf("expected persisted Jenkins release fields, got %+v", repo.created)
	}
	waitForReleaseBuildStatus(t, repo, "88", "BUILDING")
	if len(repo.created.ServiceIDs) != 1 || repo.created.ServiceIDs[0] != "svc-user" || repo.created.ServiceNames[0] != "user-service" {
		t.Fatalf("expected persisted service relation, got %+v", repo.created)
	}
}

func TestCreateReleaseSelectsPipelineFromBoundJenkinsView(t *testing.T) {
	jenkins := &recordingJenkinsAdapter{
		result: integration.BuildResult{
			BuildID: "90",
			Status:  "BUILDING",
			URL:     "http://jenkins.local/view/release-view/job/user-service-release/90/",
		},
		parameters: []domain.JenkinsPipelineParameter{
			{Name: "git_branch", Type: "StringParameterDefinition", Required: true},
			{Name: "VERSION", Type: "StringParameterDefinition", Required: true},
		},
	}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-remote",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{
					Name: "user-service-release",
					View: "other-view",
					URL:  "http://jenkins.local/view/other-view/job/user-service-release/",
				},
				{
					Name: "user-service-release",
					View: "release-view",
					URL:  "http://jenkins.local/view/release-view/job/user-service-release/",
					Parameters: []domain.JenkinsPipelineParameter{
						{Name: "git_branch", Type: "StringParameterDefinition", Required: true},
						{Name: "VERSION", Type: "StringParameterDefinition", Required: true},
					},
				},
			},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-remote",
		Jenkins: ReleaseJenkins{
			JobName: "user-service-release",
			Branch:  "release/20260628",
			Parameters: map[string]string{
				"VERSION": "v1.0.0",
			},
		},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	waitForJenkinsRequest(t, jenkins)
	expectedJobURL := "http://jenkins.local/view/release-view/job/user-service-release/"
	if jenkins.request.JobURL != expectedJobURL {
		t.Fatalf("expected bound view job URL %s, got %+v", expectedJobURL, jenkins.request)
	}
	if jenkins.request.Parameters["git_branch"] != "release/20260628" || jenkins.request.Parameters["VERSION"] != "v1.0.0" {
		t.Fatalf("expected Jenkins parameters to be forwarded, got %+v", jenkins.request.Parameters)
	}
}

func TestCreateReleaseRequiresKnownJenkinsParameters(t *testing.T) {
	jenkins := &recordingJenkinsAdapter{}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-remote",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{
					Name: "user-service-release",
					View: "release-view",
					URL:  "http://jenkins.local/view/release-view/job/user-service-release/",
					Parameters: []domain.JenkinsPipelineParameter{
						{Name: "git_branch", Type: "StringParameterDefinition", Required: true},
						{Name: "VERSION", Type: "StringParameterDefinition", Required: true},
					},
				},
			},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-remote",
		Jenkins: ReleaseJenkins{
			JobName: "user-service-release",
			Branch:  "release/20260628",
		},
	})
	var triggerErr JenkinsTriggerError
	if !errors.As(err, &triggerErr) || !strings.Contains(triggerErr.Reason, "请填写 Jenkins 参数：VERSION") {
		t.Fatalf("expected missing VERSION validation error, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("expected release order not to be created")
	}
	if jenkins.request.JobName != "" {
		t.Fatalf("expected Jenkins build not to be triggered, got %+v", jenkins.request)
	}
}

func TestCreateReleaseWithJenkinsViewRejectsJobsFallbackWhenPipelinesEmpty(t *testing.T) {
	jenkins := &recordingJenkinsAdapter{
		result: integration.BuildResult{
			BuildID: "89",
			Status:  "QUEUED",
			URL:     "http://jenkins.local/job/user-service-release/89/",
		},
	}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-local",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:   "jenkins-local",
			URL:  "http://jenkins.local",
			Jobs: []string{"user-service-release"},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-local",
		Jenkins: ReleaseJenkins{
			JobName: "user-service-release",
			Branch:  "main",
		},
	})
	var triggerErr JenkinsTriggerError
	if !errors.As(err, &triggerErr) || !strings.Contains(triggerErr.Reason, "未在当前产品绑定的 Jenkins view 中找到 Pipeline") {
		t.Fatalf("expected bound view pipeline validation error, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("expected release order not to be created")
	}
	if jenkins.request.JobName != "" {
		t.Fatalf("expected Jenkins build not to be triggered, got %+v", jenkins.request)
	}
}

func TestCreateReleaseRejectsPipelineFromOtherJenkinsView(t *testing.T) {
	jenkins := &recordingJenkinsAdapter{
		result: integration.BuildResult{
			BuildID: "91",
			Status:  "QUEUED",
		},
	}
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-local",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{
					Name: "user-service-release",
					View: "other-view",
					URL:  "http://jenkins.local/view/other-view/job/user-service-release/",
				},
			},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    jenkins,
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-local",
		Jenkins: ReleaseJenkins{
			JobName: "user-service-release",
			Branch:  "main",
		},
	})
	var triggerErr JenkinsTriggerError
	if !errors.As(err, &triggerErr) || !strings.Contains(triggerErr.Reason, "未在当前产品绑定的 Jenkins view 中找到 Pipeline") {
		t.Fatalf("expected other view pipeline validation error, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("expected release order not to be created")
	}
	if jenkins.request.JobName != "" {
		t.Fatalf("expected Jenkins build not to be triggered, got %+v", jenkins.request)
	}
}

func TestCreateReleaseWithRealJenkinsFailureMarksOrderFailed(t *testing.T) {
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-remote",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{Name: "user-service-release", View: "release-view"},
			},
		},
	}
	creator := NewReleaseCreator(integration.Suite{
		Jenkins:    &recordingJenkinsAdapter{err: errors.New("403 forbidden")},
		Registry:   integration.UnsupportedRegistryAdapter{},
		Kubernetes: integration.UnsupportedKubernetesAdapter{},
	}, repo, nil, nil)

	result, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-remote",
		Jenkins:             ReleaseJenkins{JobName: "user-service-release"},
	})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if result.Status != "JENKINS_TRIGGERING" || !repo.createCalled {
		t.Fatalf("expected release order before async trigger failure, got result=%+v created=%v", result, repo.createCalled)
	}
	waitForReleaseBuildStatus(t, repo, "", "TRIGGER_FAILED")
	if repo.updatedStatus != "FAILED" {
		t.Fatalf("expected failed release after trigger failure, got status=%s buildStatus=%s", repo.updatedStatus, repo.updatedBuildStatus)
	}
}

func TestCreateReleaseWithJenkinsResolveFailureDoesNotFallbackToGeneratedJob(t *testing.T) {
	repo := &releaseTestRepository{
		environment: domain.Environment{
			ID:          "product-remote",
			NetworkMode: "DIRECT",
			JenkinsID:   "jenkins-local",
			JenkinsView: "release-view",
		},
		jenkins: domain.JenkinsInstance{
			ID:  "jenkins-local",
			URL: "http://jenkins.local",
			Pipelines: []domain.JenkinsPipeline{
				{Name: "other-pipeline", View: "release-view"},
			},
		},
	}
	creator := NewReleaseCreator(newTestIntegrationSuite(), repo, nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "product-remote",
		Jenkins:             ReleaseJenkins{JobName: "missing-pipeline"},
	})
	var triggerErr JenkinsTriggerError
	if !errors.As(err, &triggerErr) {
		t.Fatalf("expected JenkinsTriggerError, got %v", err)
	}
	if repo.createCalled {
		t.Fatalf("expected no release order when bound Jenkins pipeline cannot be resolved")
	}
}

func TestCreateDeployTaskUsesKubernetesProbe(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), testDiffReader{
		result: domain.DiffResult{
			Items: []domain.DiffItem{
				{ServiceID: "svc-web", DiffStatus: "MISSING_IN_TARGET"},
			},
		},
	}, nil)

	result, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		SourceBaselineID:    "BL-20260607-0001",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if err != nil {
		t.Fatalf("create deploy task: %v", err)
	}
	if result.ExecutionMode != "AGENT" || result.AgentTaskID == "" {
		t.Fatalf("unexpected deploy result: %+v", result)
	}
}

func TestCreateDeployTaskReturnsWorkloadProbeError(t *testing.T) {
	creator := NewReleaseCreator(integration.Suite{
		Kubernetes: failingKubernetesAdapter{err: errors.New("boom")},
	}, newTestAgentReader(), testDiffReader{
		result: domain.DiffResult{
			Items: []domain.DiffItem{
				{ServiceID: "svc-web", DiffStatus: "MISSING_IN_TARGET"},
			},
		},
	}, nil)

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		SourceBaselineID:    "BL-20260607-0001",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if !errors.Is(err, ErrWorkloadProbe) {
		t.Fatalf("expected ErrWorkloadProbe, got %v", err)
	}
}

func TestCreateReleaseReturnsAgentNotFound(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-missing",
	})
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("expected ErrAgentNotFound, got %v", err)
	}
}

func TestCreateReleaseReturnsAgentOffline(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-project-z-prod",
		AgentID:             "agent-project-z",
	})
	if !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected ErrAgentOffline, got %v", err)
	}
}

func TestCreateReleaseReturnsEnvironmentPermissionDenied(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil, testPermissionReader{
		"env-project-x-prod:deploy": true,
	})

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if !errors.Is(err, ErrEnvironmentPermission) {
		t.Fatalf("expected ErrEnvironmentPermission, got %v", err)
	}
}

func TestCreateDeployTaskReturnsAgentEnvironmentMismatch(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-y",
	})
	if !errors.Is(err, ErrAgentEnvironment) {
		t.Fatalf("expected ErrAgentEnvironment, got %v", err)
	}
}

func TestCreateDeployTaskReturnsEnvironmentPermissionDenied(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil, testPermissionReader{
		"env-project-x-prod:release": true,
	})

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		SourceBaselineID:    "BL-20260607-0001",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if !errors.Is(err, ErrEnvironmentPermission) {
		t.Fatalf("expected ErrEnvironmentPermission, got %v", err)
	}
}

func TestCreateReleaseRejectsSourceBaseline(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	_, err := creator.CreateRelease(context.Background(), CreateReleaseRequest{
		Type:                "SERVICE_RELEASE",
		ReleaseSource:       "JENKINS_JOB",
		SourceBaselineID:    "BL-20260607-0001",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
		ServiceIDs:          []string{"svc-user"},
	})
	if !errors.Is(err, ErrReleaseBaselineUnsupported) {
		t.Fatalf("expected ErrReleaseBaselineUnsupported, got %v", err)
	}
}

func TestCreateDeployTaskRequiresSourceBaseline(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), nil, nil)

	_, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
		ServiceIDs:          []string{"svc-web"},
	})
	if !errors.Is(err, ErrDeployBaselineRequired) {
		t.Fatalf("expected ErrDeployBaselineRequired, got %v", err)
	}
}

func TestCreateDeployTaskUsesMissingInTargetDiffSelection(t *testing.T) {
	creator := NewReleaseCreator(newTestIntegrationSuite(), newTestAgentReader(), testDiffReader{
		result: domain.DiffResult{
			Items: []domain.DiffItem{
				{ServiceID: "svc-user", DiffStatus: "NEED_UPDATE"},
				{ServiceID: "svc-web", DiffStatus: "MISSING_IN_TARGET"},
			},
		},
	}, nil)

	result, err := creator.CreateDeployTask(context.Background(), CreateDeployTaskRequest{
		Type:                "SERVICE_DEPLOYMENT",
		SourceBaselineID:    "BL-20260607-0001",
		TargetEnvironmentID: "env-project-x-prod",
		AgentID:             "agent-project-x",
	})
	if err != nil {
		t.Fatalf("create deploy task: %v", err)
	}
	if result.ExecutionMode != "AGENT" || result.AgentTaskID == "" {
		t.Fatalf("unexpected deploy result: %+v", result)
	}
}

type testAgentReader map[string]string

type testEnvironmentAgentReader struct {
	testAgentReader
	environment domain.Environment
}

type testPermissionReader map[string]bool

type testDiffReader struct {
	result domain.DiffResult
	ok     bool
}

func (m testDiffReader) GetDiffResult(id string, targetEnvironmentID string) (domain.DiffResult, bool) {
	if m.ok {
		return m.result, true
	}
	if len(m.result.Items) > 0 {
		return m.result, true
	}
	return domain.DiffResult{}, false
}

func newTestAgentReader() testAgentReader {
	return testAgentReader{
		"agent-project-x": "env-project-x-prod:ONLINE",
		"agent-project-y": "env-project-y-pre:ONLINE",
		"agent-project-z": "env-project-z-prod:OFFLINE",
	}
}

func (m testAgentReader) GetAgent(id string) (domain.Agent, bool) {
	raw, ok := m[id]
	if !ok {
		return domain.Agent{}, false
	}
	parts := strings.SplitN(raw, ":", 2)
	return domain.Agent{
		ID:            id,
		EnvironmentID: parts[0],
		Status:        parts[1],
	}, true
}

func (m testEnvironmentAgentReader) GetEnvironment(id string) (domain.Environment, bool) {
	if m.environment.ID == id {
		return m.environment, true
	}
	return domain.Environment{}, false
}

func (m testPermissionReader) HasEnvironmentAction(environmentID string, action string) bool {
	return m[environmentID+":"+action]
}

func newTestIntegrationSuite() integration.Suite {
	return integration.Suite{
		Jenkins: &recordingJenkinsAdapter{result: integration.BuildResult{
			BuildID: "test-build",
			Status:  "QUEUED",
			URL:     "http://jenkins.local/job/test-build/",
		}},
		Registry:   testRegistryAdapter{},
		Kubernetes: testKubernetesAdapter{},
	}
}

type testRegistryAdapter struct{}

func (testRegistryAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (integration.IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return integration.IntegrationCheck{}, err
	}
	return integration.IntegrationCheck{Component: "registry", Status: "HEALTHY", CheckedAt: time.Now().Format(time.RFC3339)}, nil
}

func (testRegistryAdapter) ListImageTags(ctx context.Context, environment domain.Environment, repository string) ([]integration.ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []integration.ImageInfo{{Image: repository, Tag: "20260607-a1b2c3", Digest: "sha256:test", Exists: true}}, nil
}

func (testRegistryAdapter) GetImage(ctx context.Context, image string, tag string) (integration.ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return integration.ImageInfo{}, err
	}
	return integration.ImageInfo{Image: image, Tag: tag, Digest: "sha256:test", Exists: true}, nil
}

func (testRegistryAdapter) SyncImage(ctx context.Context, req integration.SyncImageRequest) (integration.SyncImageResult, error) {
	if err := ctx.Err(); err != nil {
		return integration.SyncImageResult{}, err
	}
	return integration.SyncImageResult{TaskID: "sync-test", Status: "SUCCESS", Digest: "sha256:test"}, nil
}

type testKubernetesAdapter struct{}

func (testKubernetesAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (integration.IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return integration.IntegrationCheck{}, err
	}
	return integration.IntegrationCheck{Component: "kubernetes", Status: "HEALTHY", CheckedAt: time.Now().Format(time.RFC3339)}, nil
}

func (testKubernetesAdapter) ListWorkloads(ctx context.Context, environment domain.Environment) ([]integration.Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	namespace := strings.TrimSpace(environment.Namespace)
	if namespace == "" {
		namespace = "default"
	}
	return []integration.Workload{{
		Namespace:     namespace,
		Name:          "user-service",
		Type:          "deployment",
		Replicas:      1,
		ReadyReplicas: 1,
		Containers: []integration.WorkloadContainer{
			{Name: "app", Type: "container", Image: "harbor.local/project-x/user-service:20260607-a1b2c3"},
		},
	}}, nil
}

func (testKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req integration.SetImageRequest) error {
	return ctx.Err()
}

func (testKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (integration.RolloutStatus, error) {
	if err := ctx.Err(); err != nil {
		return integration.RolloutStatus{}, err
	}
	return integration.RolloutStatus{Workload: workload, Status: "SUCCESS", Replicas: 1, ReadyReplicas: 1}, nil
}

type failingKubernetesAdapter struct {
	err error
}

type recordingJenkinsAdapter struct {
	request       integration.BuildRequest
	result        integration.BuildResult
	err           error
	parameters    []domain.JenkinsPipelineParameter
	parametersErr error
}

func (r *recordingJenkinsAdapter) TriggerBuild(ctx context.Context, req integration.BuildRequest) (integration.BuildResult, error) {
	if err := ctx.Err(); err != nil {
		return integration.BuildResult{}, err
	}
	r.request = req
	if r.err != nil {
		return integration.BuildResult{}, r.err
	}
	return r.result, nil
}

func (r *recordingJenkinsAdapter) GetBuildStatus(ctx context.Context, req integration.BuildStatusRequest) (integration.BuildStatus, error) {
	if err := ctx.Err(); err != nil {
		return integration.BuildStatus{}, err
	}
	return integration.BuildStatus{}, nil
}

func (r *recordingJenkinsAdapter) GetJobParameters(ctx context.Context, req integration.JobParametersRequest) ([]domain.JenkinsPipelineParameter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.parametersErr != nil {
		return nil, r.parametersErr
	}
	if r.parameters != nil {
		return r.parameters, nil
	}
	return nil, nil
}

type releaseTestRepository struct {
	environment        domain.Environment
	jenkins            domain.JenkinsInstance
	managedServices    []domain.ManagedService
	createCalled       bool
	created            domain.CreateReleaseOrderInput
	updatedBuildID     string
	updatedBuildStatus string
	updatedBuildURL    string
	updatedStatus      string
	updatedProgress    int
}

func (r *releaseTestRepository) GetAgent(id string) (domain.Agent, bool) {
	if strings.TrimSpace(id) == "" {
		return domain.Agent{}, false
	}
	return domain.Agent{}, false
}

func (r *releaseTestRepository) GetEnvironment(id string) (domain.Environment, bool) {
	if r.environment.ID == id {
		return r.environment, true
	}
	return domain.Environment{}, false
}

func (r *releaseTestRepository) GetJenkinsInstance(id string) (domain.JenkinsInstance, bool) {
	if r.jenkins.ID == id {
		return r.jenkins, true
	}
	return domain.JenkinsInstance{}, false
}

func (r *releaseTestRepository) CreateReleaseOrder(input domain.CreateReleaseOrderInput) (domain.ReleaseOrder, error) {
	r.createCalled = true
	r.created = input
	return domain.ReleaseOrder{
		ID:                  input.ID,
		Type:                input.Type,
		ReleaseSource:       input.ReleaseSource,
		ExecutionMode:       input.ExecutionMode,
		BuildID:             input.BuildID,
		BuildStatus:         input.BuildStatus,
		BuildURL:            input.BuildURL,
		JenkinsID:           input.JenkinsID,
		JenkinsJobName:      input.JenkinsJobName,
		JenkinsJobURL:       input.JenkinsJobURL,
		TargetEnvironmentID: input.TargetEnvironmentID,
		Status:              input.Status,
		Progress:            input.Progress,
		ServiceIDs:          input.ServiceIDs,
		ServiceNames:        input.ServiceNames,
	}, nil
}

func (r *releaseTestRepository) UpdateReleaseBuildStatus(id string, buildID string, buildStatus string, buildURL string, status string, progress int) (domain.ReleaseOrder, bool, error) {
	if !r.createCalled || r.created.ID != id {
		return domain.ReleaseOrder{}, false, nil
	}
	r.updatedBuildID = buildID
	r.updatedBuildStatus = buildStatus
	r.updatedBuildURL = buildURL
	r.updatedStatus = status
	r.updatedProgress = progress
	return domain.ReleaseOrder{
		ID:          id,
		BuildID:     buildID,
		BuildStatus: buildStatus,
		BuildURL:    buildURL,
		Status:      status,
		Progress:    progress,
	}, true, nil
}

func (r *releaseTestRepository) ListManagedServices(productID string) []domain.ManagedService {
	if r.environment.ID != productID {
		return nil
	}
	return r.managedServices
}

func (f failingKubernetesAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (integration.IntegrationCheck, error) {
	return integration.IntegrationCheck{}, f.err
}

func (f failingKubernetesAdapter) ListWorkloads(ctx context.Context, environment domain.Environment) ([]integration.Workload, error) {
	return nil, f.err
}

func (f failingKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req integration.SetImageRequest) error {
	return f.err
}

func (f failingKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (integration.RolloutStatus, error) {
	return integration.RolloutStatus{}, f.err
}

func waitForJenkinsRequest(t *testing.T, adapter *recordingJenkinsAdapter) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if adapter.request.JobName != "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for Jenkins trigger, got %+v", adapter.request)
}

func waitForReleaseBuildStatus(t *testing.T, repo *releaseTestRepository, buildID string, buildStatus string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if repo.updatedBuildStatus == buildStatus && (buildID == "" || repo.updatedBuildID == buildID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for build status %s/%s, got buildID=%s buildStatus=%s status=%s", buildID, buildStatus, repo.updatedBuildID, repo.updatedBuildStatus, repo.updatedStatus)
}
