package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/repository"
)

func TestCoreMockAPIs(t *testing.T) {
	router := newTestRouter()
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		statusCode int
	}{
		{name: "health", method: http.MethodGet, path: "/api/healthz", statusCode: http.StatusOK},
		{name: "environments", method: http.MethodGet, path: "/api/environments", statusCode: http.StatusOK},
		{name: "agents", method: http.MethodGet, path: "/api/agents", statusCode: http.StatusOK},
		{name: "baselines", method: http.MethodGet, path: "/api/baselines", statusCode: http.StatusOK},
		{name: "create baseline", method: http.MethodPost, path: "/api/baselines", body: `{"sourceEnvironmentId":"env-project-x-prod","name":"project-x-prod-20260608-2200","purpose":"远程部署前快照"}`, statusCode: http.StatusCreated},
		{name: "baseline detail", method: http.MethodGet, path: "/api/baselines/BL-20260607-0001", statusCode: http.StatusOK},
		{name: "lock baseline", method: http.MethodPost, path: "/api/baselines/BL-20260607-0002/lock", body: `{}`, statusCode: http.StatusOK},
		{name: "baseline compare", method: http.MethodPost, path: "/api/baselines/BL-20260607-0001/compare", body: "{}", statusCode: http.StatusOK},
		{name: "release list", method: http.MethodGet, path: "/api/releases", statusCode: http.StatusOK},
		{name: "create release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`, statusCode: http.StatusCreated},
		{name: "create service release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`, statusCode: http.StatusCreated},
		{name: "create image release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"LOCAL_HARBOR_IMAGE","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","image":{"repository":"harbor.local/project-x/user-service","tag":"20260607-a1b2c3"}}`, statusCode: http.StatusCreated},
		{name: "release detail", method: http.MethodGet, path: "/api/releases/REL-20260607-031", statusCode: http.StatusOK},
		{name: "created release detail", method: http.MethodGet, path: "/api/releases/REL-20260607-MOCK", statusCode: http.StatusOK},
		{name: "deploy tasks", method: http.MethodGet, path: "/api/deploy-tasks", statusCode: http.StatusOK},
		{name: "create deploy task", method: http.MethodPost, path: "/api/deploy-tasks", body: `{"type":"SERVICE_DEPLOYMENT","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x"}`, statusCode: http.StatusCreated},
		{name: "deploy detail", method: http.MethodGet, path: "/api/deploy-tasks/DEP-20260607-009", statusCode: http.StatusOK},
		{name: "created deploy detail", method: http.MethodGet, path: "/api/deploy-tasks/DEP-20260607-MOCK", statusCode: http.StatusOK},
		{name: "auth login", method: http.MethodPost, path: "/api/auth/login", body: `{"username":"admin","password":"mock"}`, statusCode: http.StatusOK},
		{name: "users", method: http.MethodGet, path: "/api/users", statusCode: http.StatusOK},
		{name: "roles", method: http.MethodGet, path: "/api/roles", statusCode: http.StatusOK},
		{name: "permissions", method: http.MethodGet, path: "/api/permissions", statusCode: http.StatusOK},
		{name: "changelog", method: http.MethodGet, path: "/api/changelog", statusCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(tt.body)
			request := httptest.NewRequest(tt.method, tt.path, body)
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d: %s", tt.statusCode, recorder.Code, recorder.Body.String())
			}
			assertOKResponse(t, recorder.Body.Bytes())
		})
	}
}

func TestAgentTaskStatusWithoutRedis(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-tasks/TASK-1/status", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != "OK" {
		t.Fatalf("expected OK response, got %s", response.Code)
	}
}

func TestAgentProtocolMockFlow(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createPayload struct {
		Data struct {
			AgentTaskID string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.AgentTaskID == "" {
		t.Fatal("expected agent task id")
	}

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/heartbeat", strings.NewReader(`{"version":"1.3.3","capabilities":["image-sync","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	pullRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/tasks/pull", strings.NewReader(`{}`))
	pullRequest.Header.Set("Content-Type", "application/json")
	pullRecorder := httptest.NewRecorder()
	router.ServeHTTP(pullRecorder, pullRequest)
	if pullRecorder.Code != http.StatusOK {
		t.Fatalf("expected pull status 200, got %d: %s", pullRecorder.Code, pullRecorder.Body.String())
	}
	var pullPayload struct {
		Data struct {
			Task struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Step   string `json:"step"`
			} `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(pullRecorder.Body.Bytes(), &pullPayload); err != nil {
		t.Fatalf("decode pull response: %v", err)
	}
	if pullPayload.Data.Task.ID != createPayload.Data.AgentTaskID || pullPayload.Data.Task.Status != "RUNNING" {
		t.Fatalf("unexpected pulled task: %+v", pullPayload.Data.Task)
	}

	stepRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/steps", strings.NewReader(`{"step":"sync-image","status":"RUNNING"}`))
	stepRequest.Header.Set("Content-Type", "application/json")
	stepRecorder := httptest.NewRecorder()
	router.ServeHTTP(stepRecorder, stepRequest)
	if stepRecorder.Code != http.StatusOK {
		t.Fatalf("expected step status 200, got %d: %s", stepRecorder.Code, stepRecorder.Body.String())
	}

	logRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/logs", strings.NewReader(`{"line":"sync image mock log"}`))
	logRequest.Header.Set("Content-Type", "application/json")
	logRecorder := httptest.NewRecorder()
	router.ServeHTTP(logRecorder, logRequest)
	if logRecorder.Code != http.StatusOK {
		t.Fatalf("expected log status 200, got %d: %s", logRecorder.Code, logRecorder.Body.String())
	}

	resultRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/result", strings.NewReader(`{"status":"SUCCESS","message":"release mock flow finished"}`))
	resultRequest.Header.Set("Content-Type", "application/json")
	resultRecorder := httptest.NewRecorder()
	router.ServeHTTP(resultRecorder, resultRequest)
	if resultRecorder.Code != http.StatusOK {
		t.Fatalf("expected result status 200, got %d: %s", resultRecorder.Code, resultRecorder.Body.String())
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/status", nil)
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, statusRequest)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("expected status query 200, got %d: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var statusPayload struct {
		Data struct {
			Status map[string]string `json:"status"`
			Logs   []string          `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusPayload.Data.Status["status"] != "SUCCESS" || statusPayload.Data.Status["step"] != "finished" {
		t.Fatalf("unexpected status payload: %+v", statusPayload.Data.Status)
	}
	if len(statusPayload.Data.Logs) != 2 {
		t.Fatalf("expected reported log and result message, got %+v", statusPayload.Data.Logs)
	}
}

func TestReleaseFailureActionsUpdateAgentTaskStatus(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createPayload struct {
		Data struct {
			AgentTaskID string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	retryRequest := httptest.NewRequest(http.MethodPost, "/api/releases/"+createPayload.Data.AgentTaskID+"/retry", nil)
	retryRecorder := httptest.NewRecorder()
	router.ServeHTTP(retryRecorder, retryRequest)
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("expected retry status 200, got %d: %s", retryRecorder.Code, retryRecorder.Body.String())
	}
	assertAgentStatus(t, router, createPayload.Data.AgentTaskID, "retry", "RUNNING")

	rollbackRequest := httptest.NewRequest(http.MethodPost, "/api/releases/"+createPayload.Data.AgentTaskID+"/rollback", nil)
	rollbackRecorder := httptest.NewRecorder()
	router.ServeHTTP(rollbackRecorder, rollbackRequest)
	if rollbackRecorder.Code != http.StatusOK {
		t.Fatalf("expected rollback status 200, got %d: %s", rollbackRecorder.Code, rollbackRecorder.Body.String())
	}
	assertAgentStatus(t, router, createPayload.Data.AgentTaskID, "rollback", "ROLLED_BACK")
}

func TestDeployStepActionsUpdateAgentTaskStatus(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","serviceIds":["svc-web"]}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected deploy create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createPayload struct {
		Data struct {
			AgentTaskID string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	tests := []struct {
		name   string
		action string
		status string
	}{
		{name: "retry", action: "retry", status: "RUNNING"},
		{name: "skip", action: "skip", status: "SKIPPED"},
		{name: "confirm", action: "confirm", status: "SUCCESS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks/"+createPayload.Data.AgentTaskID+"/steps/step-2/"+tt.action, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected step action status 200, got %d: %s", recorder.Code, recorder.Body.String())
			}
			assertAgentStatus(t, router, createPayload.Data.AgentTaskID, "step-2", tt.status)
		})
	}
}

func TestEnvironmentCheckUsesMockIntegrations(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/environments/env-project-x-prod/check", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			EnvironmentID string                         `json:"environmentId"`
			Status        string                         `json:"status"`
			Checks        []integration.IntegrationCheck `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "OK" || payload.Data.EnvironmentID != "env-project-x-prod" {
		t.Fatalf("unexpected response: %+v", payload)
	}
	if payload.Data.Status != "HEALTHY" || len(payload.Data.Checks) != 2 {
		t.Fatalf("expected healthy k8s and registry checks, got %+v", payload.Data)
	}
}

func TestCreateReleaseReturnsAgentTaskID(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			ExecutionMode string `json:"executionMode"`
			AgentTaskID   string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "OK" {
		t.Fatalf("expected OK response, got %s", payload.Code)
	}
	if payload.Data.ExecutionMode != "JENKINS_AGENT" {
		t.Fatalf("expected executionMode JENKINS_AGENT, got %s", payload.Data.ExecutionMode)
	}
	if payload.Data.AgentTaskID == "" || payload.Data.AgentTaskID != payload.Data.ID {
		t.Fatalf("expected agentTaskId to match id, got %+v", payload.Data)
	}
}

func TestCreateDeployTaskReturnsAgentTaskID(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			ExecutionMode string `json:"executionMode"`
			AgentTaskID   string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "OK" {
		t.Fatalf("expected OK response, got %s", payload.Code)
	}
	if payload.Data.ExecutionMode != "AGENT" {
		t.Fatalf("expected executionMode AGENT, got %s", payload.Data.ExecutionMode)
	}
	if payload.Data.AgentTaskID == "" || payload.Data.AgentTaskID != payload.Data.ID {
		t.Fatalf("expected agentTaskId to match id, got %+v", payload.Data)
	}
}

func TestCreateImageReleaseReturnsRegistrySyncMetadata(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"LOCAL_HARBOR_IMAGE","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","image":{"repository":"harbor.local/project-x/user-service","tag":"20260607-a1b2c3"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			ExecutionMode string `json:"executionMode"`
			BuildID       string `json:"buildId"`
			BuildStatus   string `json:"buildStatus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.ExecutionMode != "AGENT_IMAGE_SYNC" {
		t.Fatalf("expected AGENT_IMAGE_SYNC, got %s", payload.Data.ExecutionMode)
	}
	if payload.Data.BuildID == "" || payload.Data.BuildStatus != "SUCCESS" {
		t.Fatalf("expected registry sync metadata, got %+v", payload.Data)
	}
}

func TestBaselineCompareUsesRequestedTargetEnvironment(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/baselines/BL-20260607-0001/compare",
		strings.NewReader(`{"targetEnvironmentId":"env-project-z-prod","refreshTargetRuntime":true}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			SourceBaselineID    string `json:"sourceBaselineId"`
			TargetEnvironmentID string `json:"targetEnvironmentId"`
			Summary             struct {
				Consistent      int `json:"consistent"`
				NeedUpdate      int `json:"needUpdate"`
				MissingInTarget int `json:"missingInTarget"`
				WorkloadError   int `json:"workloadError"`
				Publishable     int `json:"publishable"`
			} `json:"summary"`
			Items []struct {
				ServiceID  string  `json:"serviceId"`
				DiffStatus string  `json:"diffStatus"`
				TargetTag  *string `json:"targetTag"`
				Strategy   string  `json:"strategy"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "OK" {
		t.Fatalf("expected OK response, got %s", payload.Code)
	}
	if payload.Data.SourceBaselineID != "BL-20260607-0001" {
		t.Fatalf("unexpected source baseline id: %+v", payload.Data)
	}
	if payload.Data.TargetEnvironmentID != "env-project-z-prod" {
		t.Fatalf("expected requested target environment, got %+v", payload.Data)
	}
	if payload.Data.Summary.Consistent != 1 || payload.Data.Summary.NeedUpdate != 1 || payload.Data.Summary.MissingInTarget != 1 || payload.Data.Summary.WorkloadError != 0 || payload.Data.Summary.Publishable != 2 {
		t.Fatalf("unexpected diff summary: %+v", payload.Data.Summary)
	}
	if len(payload.Data.Items) != 3 {
		t.Fatalf("expected 3 diff items, got %+v", payload.Data.Items)
	}
	if payload.Data.Items[0].DiffStatus != "NEED_UPDATE" || payload.Data.Items[1].DiffStatus != "CONSISTENT" || payload.Data.Items[2].DiffStatus != "MISSING_IN_TARGET" {
		t.Fatalf("unexpected diff items: %+v", payload.Data.Items)
	}
}

func TestCreateBaselineReturnsRuntimeSnapshot(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/baselines",
		strings.NewReader(`{"sourceEnvironmentId":"env-project-x-prod","name":"project-x-prod-20260608-2200","purpose":"远程部署前快照"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			ID                  string `json:"id"`
			Name                string `json:"name"`
			SourceEnvironmentID string `json:"sourceEnvironmentId"`
			Status              string `json:"status"`
			ServiceCount        int    `json:"serviceCount"`
			SnapshotSource      string `json:"snapshotSource"`
			SnapshotCollectedAt string `json:"snapshotCollectedAt"`
			SnapshotMode        string `json:"snapshotMode"`
			SnapshotTaskID      string `json:"snapshotTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "OK" {
		t.Fatalf("expected OK response, got %s", payload.Code)
	}
	if payload.Data.SourceEnvironmentID != "env-project-x-prod" || payload.Data.Status != "DRAFT" {
		t.Fatalf("unexpected baseline payload: %+v", payload.Data)
	}
	if payload.Data.ID == "" || payload.Data.ServiceCount == 0 {
		t.Fatalf("expected generated baseline snapshot, got %+v", payload.Data)
	}
	if payload.Data.SnapshotSource == "" || payload.Data.SnapshotCollectedAt == "" || payload.Data.SnapshotMode != "MOCK_RUNTIME" || payload.Data.SnapshotTaskID == "" {
		t.Fatalf("expected runtime snapshot metadata, got %+v", payload.Data)
	}
}

func TestLockBaselineUpdatesStatus(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/baselines/BL-20260607-0002/lock", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
		Data struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			LockedAt string `json:"lockedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.ID != "BL-20260607-0002" || payload.Data.Status != "LOCKED" || payload.Data.LockedAt == "" {
		t.Fatalf("unexpected lock response: %+v", payload.Data)
	}
}

func TestCreateReleaseRejectsOfflineAgent(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-z-prod","agentId":"agent-project-z","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateReleaseRejectsAgentEnvironmentMismatch(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-y","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateDeployTaskRejectsOfflineAgent(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","targetEnvironmentId":"env-project-z-prod","agentId":"agent-project-z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateDeployTaskRejectsAgentEnvironmentMismatch(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-y"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateReleaseRejectsSourceBaseline(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","serviceIds":["svc-user"],"jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateDeployRequiresSourceBaseline(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","serviceIds":["svc-web"]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateDeployRejectsNeedUpdateServiceSelection(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","serviceIds":["svc-user"]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateReleaseReturnsForbiddenWithoutEnvironmentPermission(t *testing.T) {
	router := newPermissionTestRouter(t, domain.CurrentUser{
		ID:          "user-viewer",
		Username:    "viewer",
		DisplayName: "只读用户",
		Roles:       []string{"VIEWER"},
		Permissions: []string{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertErrorResponse(t, recorder.Body.Bytes(), "FORBIDDEN", "environment permission denied")
}

func TestCreateDeployTaskReturnsForbiddenWithoutEnvironmentPermission(t *testing.T) {
	router := newPermissionTestRouter(t, domain.CurrentUser{
		ID:          "user-viewer",
		Username:    "viewer",
		DisplayName: "只读用户",
		Roles:       []string{"VIEWER"},
		Permissions: []string{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/deploy-tasks", strings.NewReader(`{"type":"SERVICE_DEPLOYMENT","sourceBaselineId":"BL-20260607-0001","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","serviceIds":["svc-project-x-web"]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertErrorResponse(t, recorder.Body.Bytes(), "FORBIDDEN", "environment permission denied")
}

func TestUnknownRoute(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func newTestRouter() http.Handler {
	return NewRouter(nil, agent.NewProtocolStore(), integration.NewMockSuite())
}

func newPermissionTestRouter(t *testing.T, user domain.CurrentUser) http.Handler {
	t.Helper()
	repo, err := repository.NewMockRepository()
	if err != nil {
		t.Fatalf("load mock repository: %v", err)
	}
	repo.SetCurrentUserForTest(user)
	handler := NewHandler(repo, nil, agent.NewProtocolStore(), integration.NewMockSuite())
	router := gin.New()
	router.POST("/api/releases", handler.CreateRelease)
	router.POST("/api/deploy-tasks", handler.CreateDeployTask)
	return router
}

func assertOKResponse(t *testing.T, payload []byte) {
	t.Helper()
	var response Response
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != "OK" {
		t.Fatalf("expected code OK, got %s", response.Code)
	}
	if response.RequestID == "" {
		t.Fatal("expected requestId")
	}
}

func assertErrorResponse(t *testing.T, payload []byte, code string, message string) {
	t.Helper()
	var response Response
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != code || response.Message != message {
		t.Fatalf("expected %s %q, got %+v", code, message, response)
	}
	if response.RequestID == "" {
		t.Fatal("expected requestId")
	}
}

func assertAgentStatus(t *testing.T, router http.Handler, taskID string, step string, status string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/agent-tasks/"+taskID+"/status", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected agent status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Status map[string]string `json:"status"`
			Logs   []string          `json:"logs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if payload.Data.Status["step"] != step || payload.Data.Status["status"] != status {
		t.Fatalf("expected step %s status %s, got %+v", step, status, payload.Data.Status)
	}
	if len(payload.Data.Logs) == 0 {
		t.Fatalf("expected action log, got %+v", payload.Data.Logs)
	}
}
