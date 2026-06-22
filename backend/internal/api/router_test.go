package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ops-release-platform/backend/internal/agent"
	"ops-release-platform/backend/internal/domain"
	"ops-release-platform/backend/internal/integration"
	"ops-release-platform/backend/internal/repository"
)

var testAgentTokens sync.Map

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
		{name: "baseline compare", method: http.MethodPost, path: "/api/baselines/BL-20260607-0001/compare", body: `{"targetEnvironmentId":"env-project-x-prod"}`, statusCode: http.StatusOK},
		{name: "release list", method: http.MethodGet, path: "/api/releases", statusCode: http.StatusOK},
		{name: "create release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`, statusCode: http.StatusCreated},
		{name: "create service release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`, statusCode: http.StatusCreated},
		{name: "create image release", method: http.MethodPost, path: "/api/releases", body: `{"type":"SERVICE_RELEASE","releaseSource":"LOCAL_HARBOR_IMAGE","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","image":{"repository":"harbor.local/project-x/user-service","tag":"20260607-a1b2c3"}}`, statusCode: http.StatusCreated},
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

func TestReleaseDetailUsesCreatedOrderOnly(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"LOCAL_HARBOR_IMAGE","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","image":{"repository":"harbor.local/project-x/user-service","tag":"20260607-a1b2c3"}}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	var createPayload struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.ID == "" {
		t.Fatal("expected release id")
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/api/releases/"+createPayload.Data.ID, nil)
	detailRecorder := httptest.NewRecorder()
	router.ServeHTTP(detailRecorder, detailRequest)
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("expected release detail status 200, got %d: %s", detailRecorder.Code, detailRecorder.Body.String())
	}
	assertOKResponse(t, detailRecorder.Body.Bytes())

	missingRequest := httptest.NewRequest(http.MethodGet, "/api/releases/REL-20260607-MOCK", nil)
	missingRecorder := httptest.NewRecorder()
	router.ServeHTTP(missingRecorder, missingRequest)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing release detail status 404, got %d: %s", missingRecorder.Code, missingRecorder.Body.String())
	}
	assertErrorResponse(t, missingRecorder.Body.Bytes(), "NOT_FOUND", "release not found")
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

	agentToken := registerTestAgent(t, router, "agent-project-x", "env-project-x-prod")
	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/heartbeat", strings.NewReader(`{"version":"1.3.3","capabilities":["image-sync","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+agentToken)
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	pullRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/tasks/pull", strings.NewReader(`{}`))
	pullRequest.Header.Set("Content-Type", "application/json")
	pullRequest.Header.Set("Authorization", "Bearer "+agentToken)
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
	stepRequest.Header.Set("Authorization", "Bearer "+agentToken)
	stepRecorder := httptest.NewRecorder()
	router.ServeHTTP(stepRecorder, stepRequest)
	if stepRecorder.Code != http.StatusOK {
		t.Fatalf("expected step status 200, got %d: %s", stepRecorder.Code, stepRecorder.Body.String())
	}

	logRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/logs", strings.NewReader(`{"line":"sync image mock log"}`))
	logRequest.Header.Set("Content-Type", "application/json")
	logRequest.Header.Set("Authorization", "Bearer "+agentToken)
	logRecorder := httptest.NewRecorder()
	router.ServeHTTP(logRecorder, logRequest)
	if logRecorder.Code != http.StatusOK {
		t.Fatalf("expected log status 200, got %d: %s", logRecorder.Code, logRecorder.Body.String())
	}

	resultRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+createPayload.Data.AgentTaskID+"/result", strings.NewReader(`{"status":"SUCCESS","message":"release mock flow finished"}`))
	resultRequest.Header.Set("Content-Type", "application/json")
	resultRequest.Header.Set("Authorization", "Bearer "+agentToken)
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

func TestAgentTaskLeaseFlow(t *testing.T) {
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

	agentToken := registerTestAgent(t, router, "agent-project-x", "env-project-x-prod")
	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/heartbeat", strings.NewReader(`{"version":"v1-mock","capabilities":["mock-executor","image-sync","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+agentToken)
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	leaseRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"agent-project-x","environmentId":"env-project-x-prod","maxTasks":1,"leaseSeconds":300}`))
	leaseRequest.Header.Set("Content-Type", "application/json")
	leaseRequest.Header.Set("Authorization", "Bearer "+agentToken)
	leaseRequest.Header.Set("X-Forwarded-Proto", "https")
	leaseRequest.Header.Set("X-Forwarded-Host", "platform.example.com")
	leaseRecorder := httptest.NewRecorder()
	router.ServeHTTP(leaseRecorder, leaseRequest)
	if leaseRecorder.Code != http.StatusOK {
		t.Fatalf("expected lease status 200, got %d: %s", leaseRecorder.Code, leaseRecorder.Body.String())
	}
	var leasePayload struct {
		Data struct {
			Leased  bool   `json:"leased"`
			LeaseID string `json:"leaseId"`
			Task    struct {
				ID            string `json:"id"`
				Status        string `json:"status"`
				AgentID       string `json:"agentId"`
				EnvironmentID string `json:"environmentId"`
				LeaseID       string `json:"leaseId"`
				Callback      struct {
					StepURL   string `json:"stepUrl"`
					LogURL    string `json:"logUrl"`
					ResultURL string `json:"resultUrl"`
				} `json:"callback"`
			} `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(leaseRecorder.Body.Bytes(), &leasePayload); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}
	if !leasePayload.Data.Leased || leasePayload.Data.Task.ID != createPayload.Data.AgentTaskID {
		t.Fatalf("unexpected lease payload: %+v", leasePayload.Data)
	}
	if leasePayload.Data.LeaseID == "" || leasePayload.Data.Task.LeaseID != leasePayload.Data.LeaseID || leasePayload.Data.Task.Status != "RUNNING" {
		t.Fatalf("expected running leased task, got %+v", leasePayload.Data)
	}
	if leasePayload.Data.Task.AgentID != "agent-project-x" || leasePayload.Data.Task.EnvironmentID != "env-project-x-prod" {
		t.Fatalf("unexpected task binding: %+v", leasePayload.Data.Task)
	}
	if leasePayload.Data.Task.Callback.StepURL == "" || leasePayload.Data.Task.Callback.LogURL == "" || leasePayload.Data.Task.Callback.ResultURL == "" {
		t.Fatalf("expected callback urls, got %+v", leasePayload.Data.Task.Callback)
	}
	if !strings.HasPrefix(leasePayload.Data.Task.Callback.ResultURL, "https://platform.example.com/api/agent-tasks/") {
		t.Fatalf("unexpected callback base url: %+v", leasePayload.Data.Task.Callback)
	}
}

func TestAgentTaskLeaseReturnsEmptyWhenAgentAlreadyRunningTask(t *testing.T) {
	router := newTestRouter()
	taskID := createReleaseAgentTask(t, router)
	heartbeatAgent(t, router, "agent-project-x")

	firstLease := leaseAgentTask(t, router, `{"agentId":"agent-project-x","environmentId":"env-project-x-prod","maxTasks":1,"leaseSeconds":300}`)
	if !firstLease.Data.Leased || firstLease.Data.Task.ID != taskID {
		t.Fatalf("expected first lease for created task, got %+v", firstLease.Data)
	}

	secondLease := leaseAgentTask(t, router, `{"agentId":"agent-project-x","environmentId":"env-project-x-prod","maxTasks":1,"leaseSeconds":300}`)
	if secondLease.Data.Leased || secondLease.Data.Task != nil {
		t.Fatalf("expected empty lease while task is running, got %+v", secondLease.Data)
	}
	if secondLease.Data.Message == "" {
		t.Fatalf("expected running task message, got %+v", secondLease.Data)
	}
}

func TestAgentTaskLeaseRejectsInvalidAgentState(t *testing.T) {
	router := newTestRouter()
	_ = createReleaseAgentTask(t, router)
	projectXToken := registerTestAgent(t, router, "agent-project-x", "env-project-x-prod")
	projectZToken := "agent-project-z-test-token"

	offlineRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"agent-project-z","environmentId":"env-project-z-prod","maxTasks":1,"leaseSeconds":300}`))
	offlineRequest.Header.Set("Content-Type", "application/json")
	offlineRequest.Header.Set("Authorization", "Bearer "+projectZToken)
	offlineRecorder := httptest.NewRecorder()
	router.ServeHTTP(offlineRecorder, offlineRequest)
	if offlineRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected offline agent lease status 400, got %d: %s", offlineRecorder.Code, offlineRecorder.Body.String())
	}

	mismatchRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"agent-project-x","environmentId":"env-project-z-prod","maxTasks":1,"leaseSeconds":300}`))
	mismatchRequest.Header.Set("Content-Type", "application/json")
	mismatchRequest.Header.Set("Authorization", "Bearer "+projectXToken)
	mismatchRecorder := httptest.NewRecorder()
	router.ServeHTTP(mismatchRecorder, mismatchRequest)
	if mismatchRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected mismatch lease status 400, got %d: %s", mismatchRecorder.Code, mismatchRecorder.Body.String())
	}

	trimmedRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"agent-project-x","environmentId":"  env-project-x-prod  ","maxTasks":1,"leaseSeconds":300}`))
	trimmedRequest.Header.Set("Content-Type", "application/json")
	trimmedRequest.Header.Set("Authorization", "Bearer "+projectXToken)
	trimmedRecorder := httptest.NewRecorder()
	router.ServeHTTP(trimmedRecorder, trimmedRequest)
	if trimmedRecorder.Code != http.StatusOK {
		t.Fatalf("expected trimmed environment lease status 200, got %d: %s", trimmedRecorder.Code, trimmedRecorder.Body.String())
	}
}

func TestAgentTaskExpiredLeaseCanBeLeasedAgain(t *testing.T) {
	router := newTestRouter()
	taskID := createReleaseAgentTask(t, router)
	heartbeatAgent(t, router, "agent-project-x")

	firstLease := leaseAgentTask(t, router, `{"agentId":"agent-project-x","environmentId":"env-project-x-prod","maxTasks":1,"leaseSeconds":1}`)
	if !firstLease.Data.Leased || firstLease.Data.Task.ID != taskID {
		t.Fatalf("expected first lease for created task, got %+v", firstLease.Data)
	}

	time.Sleep(1100 * time.Millisecond)

	secondLease := leaseAgentTask(t, router, `{"agentId":"agent-project-x","environmentId":"env-project-x-prod","maxTasks":1,"leaseSeconds":300}`)
	if !secondLease.Data.Leased || secondLease.Data.Task.ID != taskID {
		t.Fatalf("expected expired task to be leased again, got %+v", secondLease.Data)
	}
	assertAgentStatus(t, router, taskID, "leased", "RUNNING")
}

func TestEnvironmentCRUD(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-new-prod","name":"新生产环境","code":"new-prod","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"project-x","registryId":"harbor-local-prod","registryProject":"project-x","status":"HEALTHY"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	autoCodeRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"name":"Auto Code Prod","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"project-x","registryId":"harbor-local-prod","registryProject":"project-x","status":"HEALTHY"}`))
	autoCodeRequest.Header.Set("Content-Type", "application/json")
	autoCodeRecorder := httptest.NewRecorder()
	router.ServeHTTP(autoCodeRecorder, autoCodeRequest)
	if autoCodeRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment with auto code status 201, got %d: %s", autoCodeRecorder.Code, autoCodeRecorder.Body.String())
	}
	var autoCodeResponse struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(autoCodeRecorder.Body.Bytes(), &autoCodeResponse); err != nil {
		t.Fatalf("decode auto code response: %v", err)
	}
	if autoCodeResponse.Data.Code != "auto-code-prod" || autoCodeResponse.Data.ID != "env-auto-code-prod" {
		t.Fatalf("expected auto-generated environment code/id, got %+v", autoCodeResponse.Data)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/environments/env-new-prod", nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected get environment status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/api/environments/env-new-prod", strings.NewReader(`{"name":"新生产环境-已更新","status":"MAINTENANCE"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update environment status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
}

func TestEnvironmentWithUnverifiedScopesIsSavedAsDegraded(t *testing.T) {
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-unverified-scope","name":"未验证范围环境","code":"unverified-scope","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"namespace-not-probed","registryId":"harbor-local-prod","registryProject":"project-not-probed","status":"HEALTHY"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResponse.Data.Status != "DEGRADED" {
		t.Fatalf("expected unverified scopes to create degraded environment, got %+v", createResponse.Data)
	}
}

func TestEnvironmentVerifiedScopesClearDegradedOnSave(t *testing.T) {
	repo := newTestRepository()
	router := newTestRouterWithRepository(repo)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-verified-later","name":"后续验证环境","code":"verified-later","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"namespace-later","registryId":"harbor-local-prod","registryProject":"project-later","status":"HEALTHY"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createResponse struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResponse.Data.Status != "DEGRADED" {
		t.Fatalf("expected initial unverified scopes to be degraded, got %+v", createResponse.Data)
	}

	if _, _, err := repo.UpdateHarborRegistryProbe("harbor-local-prod", "HEALTHY", "", []string{"project-x", "project-later"}, time.Now()); err != nil {
		t.Fatalf("update harbor probe: %v", err)
	}
	if _, _, err := repo.UpdateKubernetesClusterProbe("k8s-local-prod", "HEALTHY", "", []string{"default", "project-x", "namespace-later"}, time.Now()); err != nil {
		t.Fatalf("update kubernetes probe: %v", err)
	}

	updateRequest := httptest.NewRequest(http.MethodPut, "/api/environments/env-verified-later", strings.NewReader(`{"name":"后续验证环境-已更新"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update environment status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updateResponse struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &updateResponse); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateResponse.Data.Status != "UNKNOWN" {
		t.Fatalf("expected verified scopes to clear degraded status to unknown, got %+v", updateResponse.Data)
	}
}

func TestResourceManagementCreatesAndRefreshesKubernetesFromKubeconfig(t *testing.T) {
	kubernetesAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/readyz":
			_, _ = w.Write([]byte(`ok`))
		case "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"project-x"}},{"metadata":{"name":"default"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer kubernetesAPI.Close()
	router := newTestRouter()
	kubeconfig := strings.ReplaceAll(`apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: __SERVER__
    insecure-skip-tls-verify: true
users:
- name: dev
  user:
    token: test-token
contexts:
- name: local-context
  context:
    cluster: local
    user: dev
current-context: local-context
`, "__SERVER__", kubernetesAPI.URL)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/kubernetes-clusters", strings.NewReader(`{"id":"k8s-dev","name":"开发集群","kubeconfig":`+strconv.Quote(kubeconfig)+`}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected kubernetes create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var createPayload struct {
		Data domain.KubernetesCluster `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode kubernetes create response: %v", err)
	}
	if createPayload.Data.APIServer != kubernetesAPI.URL {
		t.Fatalf("expected kubernetes api server %q, got %q", kubernetesAPI.URL, createPayload.Data.APIServer)
	}
	if strings.Contains(createRecorder.Body.String(), "test-token") || strings.Contains(createRecorder.Body.String(), "kubeconfig") {
		t.Fatalf("expected kubeconfig secret fields not to be returned: %s", createRecorder.Body.String())
	}

	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/kubernetes-clusters/k8s-dev/refresh", nil)
	refreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(refreshRecorder, refreshRequest)
	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected kubernetes refresh status 200, got %d: %s", refreshRecorder.Code, refreshRecorder.Body.String())
	}
	var payload struct {
		Data domain.KubernetesCluster `json:"data"`
	}
	if err := json.Unmarshal(refreshRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode kubernetes refresh response: %v", err)
	}
	if payload.Data.Status != "HEALTHY" || !stringSliceContains(payload.Data.Namespaces, "project-x") {
		t.Fatalf("expected healthy kubernetes namespace cache, got %+v", payload.Data)
	}
}

func TestResourceManagementDoesNotExposeHarborAndJenkinsSecrets(t *testing.T) {
	harborAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/systeminfo":
			_, _ = w.Write([]byte(`{"harbor_version":"dev"}`))
		case "/api/v2.0/projects":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"name":"project-x"},{"name":"project-y"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer harborAPI.Close()
	jenkinsAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mode":"NORMAL","views":[{"name":"project-x"}],"jobs":[{"name":"build-project-x"}]}`))
	}))
	defer jenkinsAPI.Close()
	router := newTestRouter()

	harborSecret := "harbor-secret-for-test"
	harborCreate := httptest.NewRequest(http.MethodPost, "/api/harbor-registries", strings.NewReader(`{"id":"harbor-dev","name":"开发 Harbor","url":"`+harborAPI.URL+`","scheme":"http","username":"dev","password":"`+harborSecret+`"}`))
	harborCreate.Header.Set("Content-Type", "application/json")
	harborCreateRecorder := httptest.NewRecorder()
	router.ServeHTTP(harborCreateRecorder, harborCreate)
	if harborCreateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected harbor create status 201, got %d: %s", harborCreateRecorder.Code, harborCreateRecorder.Body.String())
	}
	if strings.Contains(harborCreateRecorder.Body.String(), harborSecret) || strings.Contains(harborCreateRecorder.Body.String(), "password") {
		t.Fatalf("expected harbor password not to be returned: %s", harborCreateRecorder.Body.String())
	}
	harborRefresh := httptest.NewRequest(http.MethodPost, "/api/harbor-registries/harbor-dev/refresh", nil)
	harborRefreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(harborRefreshRecorder, harborRefresh)
	if harborRefreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected harbor refresh status 200, got %d: %s", harborRefreshRecorder.Code, harborRefreshRecorder.Body.String())
	}
	var harborPayload struct {
		Data domain.HarborRegistry `json:"data"`
	}
	if err := json.Unmarshal(harborRefreshRecorder.Body.Bytes(), &harborPayload); err != nil {
		t.Fatalf("decode harbor refresh response: %v", err)
	}
	if harborPayload.Data.Status != "HEALTHY" || !stringSliceContains(harborPayload.Data.Projects, "project-x") {
		t.Fatalf("expected healthy harbor project cache, got %+v", harborPayload.Data)
	}

	jenkinsSecret := "jenkins-secret-for-test"
	jenkinsCreate := httptest.NewRequest(http.MethodPost, "/api/jenkins-instances", strings.NewReader(`{"id":"jenkins-dev","name":"开发 Jenkins","url":"`+jenkinsAPI.URL+`","username":"dev","token":"`+jenkinsSecret+`"}`))
	jenkinsCreate.Header.Set("Content-Type", "application/json")
	jenkinsCreateRecorder := httptest.NewRecorder()
	router.ServeHTTP(jenkinsCreateRecorder, jenkinsCreate)
	if jenkinsCreateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected jenkins create status 201, got %d: %s", jenkinsCreateRecorder.Code, jenkinsCreateRecorder.Body.String())
	}
	if strings.Contains(jenkinsCreateRecorder.Body.String(), jenkinsSecret) || strings.Contains(jenkinsCreateRecorder.Body.String(), "token") {
		t.Fatalf("expected jenkins token not to be returned: %s", jenkinsCreateRecorder.Body.String())
	}
	jenkinsRefresh := httptest.NewRequest(http.MethodPost, "/api/jenkins-instances/jenkins-dev/refresh", nil)
	jenkinsRefreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(jenkinsRefreshRecorder, jenkinsRefresh)
	if jenkinsRefreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected jenkins refresh status 200, got %d: %s", jenkinsRefreshRecorder.Code, jenkinsRefreshRecorder.Body.String())
	}
	var jenkinsPayload struct {
		Data domain.JenkinsInstance `json:"data"`
	}
	if err := json.Unmarshal(jenkinsRefreshRecorder.Body.Bytes(), &jenkinsPayload); err != nil {
		t.Fatalf("decode jenkins refresh response: %v", err)
	}
	if jenkinsPayload.Data.Status != "HEALTHY" || !stringSliceContains(jenkinsPayload.Data.Views, "project-x") || !stringSliceContains(jenkinsPayload.Data.Jobs, "build-project-x") {
		t.Fatalf("expected healthy jenkins cache, got %+v", jenkinsPayload.Data)
	}
}

func TestResourceRefreshFailureKeepsPreviousCache(t *testing.T) {
	harborAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2.0/systeminfo":
			_, _ = w.Write([]byte(`{"harbor_version":"dev"}`))
		case "/api/v2.0/projects":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"name":"project-x"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	router := newTestRouter()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/harbor-registries", strings.NewReader(`{"id":"harbor-cache","name":"缓存 Harbor","url":"`+harborAPI.URL+`","scheme":"http"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected harbor create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/harbor-registries/harbor-cache/refresh", nil)
	refreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(refreshRecorder, refreshRequest)
	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected first harbor refresh status 200, got %d: %s", refreshRecorder.Code, refreshRecorder.Body.String())
	}

	harborAPI.Close()
	failedRefreshRequest := httptest.NewRequest(http.MethodPost, "/api/harbor-registries/harbor-cache/refresh", nil)
	failedRefreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(failedRefreshRecorder, failedRefreshRequest)
	if failedRefreshRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected failed harbor refresh status 400, got %d: %s", failedRefreshRecorder.Code, failedRefreshRecorder.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/harbor-registries", nil)
	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected harbor list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var payload struct {
		Data struct {
			Items []domain.HarborRegistry `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode harbor list response: %v", err)
	}
	for _, item := range payload.Data.Items {
		if item.ID == "harbor-cache" {
			if item.Status != "UNHEALTHY" || !stringSliceContains(item.Projects, "project-x") || item.ProbeMessage == "" {
				t.Fatalf("expected failed refresh to keep project cache and record failure, got %+v", item)
			}
			return
		}
	}
	t.Fatal("expected harbor-cache resource in list")
}

func TestAgentHeartbeatRejectsUnknownEnvironment(t *testing.T) {
	router := newTestRouter()
	agentToken := registerTestAgent(t, router, "agent-new", "")
	request := httptest.NewRequest(http.MethodPost, "/api/agents/agent-new/heartbeat", strings.NewReader(`{"environmentId":"env-missing","version":"v1-mock","capabilities":["mock-executor"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+agentToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected heartbeat status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertErrorResponse(t, recorder.Body.Bytes(), "VALIDATION_ERROR", "environment not found")
}

func TestAgentHeartbeatRejectsEnvironmentRebind(t *testing.T) {
	router := newTestRouter()

	createEnvironmentRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-remote-agent-rebind-test","name":"远程 Agent 绑定测试环境","code":"remote-agent-rebind-test","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"project-x","registryId":"harbor-local-prod","registryProject":"project-x","status":"HEALTHY"}`))
	createEnvironmentRequest.Header.Set("Content-Type", "application/json")
	createEnvironmentRecorder := httptest.NewRecorder()
	router.ServeHTTP(createEnvironmentRecorder, createEnvironmentRequest)
	if createEnvironmentRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createEnvironmentRecorder.Code, createEnvironmentRecorder.Body.String())
	}

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-project-x/heartbeat", strings.NewReader(`{"environmentId":"env-remote-agent-rebind-test","version":"v1-mock","capabilities":["mock-executor","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+agentTokenForTest(t, router, "agent-project-x", "env-project-x-prod"))
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected heartbeat status 400, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}
	assertErrorResponse(t, heartbeatRecorder.Body.Bytes(), "VALIDATION_ERROR", "agent environment does not match claimed environment")

	agentRequest := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	agentRecorder := httptest.NewRecorder()
	router.ServeHTTP(agentRecorder, agentRequest)
	if agentRecorder.Code != http.StatusOK {
		t.Fatalf("expected list agents status 200, got %d: %s", agentRecorder.Code, agentRecorder.Body.String())
	}

	var payload struct {
		Data struct {
			Items []domain.Agent `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(agentRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode agent list response: %v", err)
	}
	for _, item := range payload.Data.Items {
		if item.ID == "agent-project-x" {
			if item.EnvironmentID != "env-project-x-prod" {
				t.Fatalf("expected original environment id, got %+v", item)
			}
			return
		}
	}
	t.Fatal("expected rebound agent in list")
}

func TestAgentRegisterClaimHeartbeatAndLeaseFlow(t *testing.T) {
	router := newTestRouter()
	agentID := "agent-registration-claim-test"
	environmentID := "env-project-x-prod"

	createTokenRequest := httptest.NewRequest(http.MethodPost, "/api/agents/register-token", strings.NewReader(`{"agentId":"`+agentID+`","environmentId":"`+environmentID+`","ttlMinutes":10}`))
	createTokenRequest.Header.Set("Content-Type", "application/json")
	createTokenRecorder := httptest.NewRecorder()
	router.ServeHTTP(createTokenRecorder, createTokenRequest)
	if createTokenRecorder.Code != http.StatusCreated {
		t.Fatalf("expected register token status 201, got %d: %s", createTokenRecorder.Code, createTokenRecorder.Body.String())
	}
	var tokenPayload struct {
		Data struct {
			Token          string `json:"token"`
			InstallCommand string `json:"installCommand"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createTokenRecorder.Body.Bytes(), &tokenPayload); err != nil {
		t.Fatalf("decode register token response: %v", err)
	}
	if tokenPayload.Data.Token == "" {
		t.Fatal("expected register token")
	}
	if strings.Contains(tokenPayload.Data.InstallCommand, "AGENT_ENVIRONMENT_ID=") {
		t.Fatalf("install command must not prebind environment: %s", tokenPayload.Data.InstallCommand)
	}

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{"agentId":"`+agentID+`","environmentId":"`+environmentID+`","registerToken":"`+tokenPayload.Data.Token+`","version":"v1-remote-probe","capabilities":["remote-probe","kubectl","http-check"]}`))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	router.ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf("expected agent register status 201, got %d: %s", registerRecorder.Code, registerRecorder.Body.String())
	}
	var registerPayload struct {
		Data struct {
			Agent      domain.Agent `json:"agent"`
			AgentToken string       `json:"agentToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(registerRecorder.Body.Bytes(), &registerPayload); err != nil {
		t.Fatalf("decode agent register response: %v", err)
	}
	if registerPayload.Data.AgentToken == "" {
		t.Fatal("expected agent runtime token")
	}
	if registerPayload.Data.Agent.EnvironmentID != "" || registerPayload.Data.Agent.ClaimStatus != "PENDING_CLAIM" {
		t.Fatalf("expected registered agent to wait for page-side claim, got %+v", registerPayload.Data.Agent)
	}

	preClaimLeaseRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"`+agentID+`","environmentId":"`+environmentID+`","maxTasks":1,"leaseSeconds":300}`))
	preClaimLeaseRequest.Header.Set("Content-Type", "application/json")
	preClaimLeaseRequest.Header.Set("Authorization", "Bearer "+registerPayload.Data.AgentToken)
	preClaimLeaseRecorder := httptest.NewRecorder()
	router.ServeHTTP(preClaimLeaseRecorder, preClaimLeaseRequest)
	if preClaimLeaseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected pre-claim lease status 400, got %d: %s", preClaimLeaseRecorder.Code, preClaimLeaseRecorder.Body.String())
	}
	assertErrorResponse(t, preClaimLeaseRecorder.Body.Bytes(), "VALIDATION_ERROR", "agent is online but not claimed by an environment")

	claimRequest := httptest.NewRequest(http.MethodPost, "/api/agents/"+agentID+"/claim", strings.NewReader(`{"environmentId":"`+environmentID+`"}`))
	claimRequest.Header.Set("Content-Type", "application/json")
	claimRecorder := httptest.NewRecorder()
	router.ServeHTTP(claimRecorder, claimRequest)
	if claimRecorder.Code != http.StatusOK {
		t.Fatalf("expected claim status 200, got %d: %s", claimRecorder.Code, claimRecorder.Body.String())
	}
	var claimPayload struct {
		Data domain.Agent `json:"data"`
	}
	if err := json.Unmarshal(claimRecorder.Body.Bytes(), &claimPayload); err != nil {
		t.Fatalf("decode claim response: %v", err)
	}
	if claimPayload.Data.EnvironmentID != environmentID || claimPayload.Data.ClaimStatus != "CLAIMED" {
		t.Fatalf("expected claimed agent, got %+v", claimPayload.Data)
	}

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/"+agentID+"/heartbeat", strings.NewReader(`{"version":"v1-remote-probe","capabilities":["remote-probe","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+registerPayload.Data.AgentToken)
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}
	var heartbeatPayload struct {
		Data struct {
			Agent domain.Agent `json:"agent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(heartbeatRecorder.Body.Bytes(), &heartbeatPayload); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if heartbeatPayload.Data.Agent.EnvironmentID != environmentID || heartbeatPayload.Data.Agent.ClaimStatus != "CLAIMED" {
		t.Fatalf("expected heartbeat to return claimed environment, got %+v", heartbeatPayload.Data.Agent)
	}

	createReleaseRequest := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"`+environmentID+`","agentId":"`+agentID+`","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	createReleaseRequest.Header.Set("Content-Type", "application/json")
	createReleaseRecorder := httptest.NewRecorder()
	router.ServeHTTP(createReleaseRecorder, createReleaseRequest)
	if createReleaseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release create status 201, got %d: %s", createReleaseRecorder.Code, createReleaseRecorder.Body.String())
	}
	var createReleasePayload struct {
		Data struct {
			AgentTaskID string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createReleaseRecorder.Body.Bytes(), &createReleasePayload); err != nil {
		t.Fatalf("decode release response: %v", err)
	}
	if createReleasePayload.Data.AgentTaskID == "" {
		t.Fatal("expected agent task id")
	}

	leaseRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(`{"agentId":"`+agentID+`","maxTasks":1,"leaseSeconds":300}`))
	leaseRequest.Header.Set("Content-Type", "application/json")
	leaseRequest.Header.Set("Authorization", "Bearer "+registerPayload.Data.AgentToken)
	leaseRecorder := httptest.NewRecorder()
	router.ServeHTTP(leaseRecorder, leaseRequest)
	if leaseRecorder.Code != http.StatusOK {
		t.Fatalf("expected lease status 200, got %d: %s", leaseRecorder.Code, leaseRecorder.Body.String())
	}
	var leasePayload leaseResponsePayload
	if err := json.Unmarshal(leaseRecorder.Body.Bytes(), &leasePayload); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}
	if !leasePayload.Data.Leased || leasePayload.Data.Task == nil || leasePayload.Data.Task.ID != createReleasePayload.Data.AgentTaskID {
		t.Fatalf("expected lease for created task, got %+v", leasePayload.Data)
	}
	if leasePayload.Data.Task.EnvironmentID != environmentID {
		t.Fatalf("expected lease to use claimed environment, got %+v", leasePayload.Data.Task)
	}
}

func TestEnvironmentRemoteProbeRequiresClaimedOnlineAgent(t *testing.T) {
	router := newTestRouter()
	createProjectProbeEnvironment(t, router, "env-probe-no-agent", "probe-no-agent", "project-x")

	request := httptest.NewRequest(http.MethodPost, "/api/environments/env-probe-no-agent/remote-probe", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected remote probe status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertErrorResponse(t, recorder.Body.Bytes(), "VALIDATION_ERROR", "未找到已认领且在线的 Agent，请先在 Agent 管理中完成注册、启动和认领")
}

func TestEnvironmentRemoteProbeEnqueuesTaskForClaimedOnlineAgent(t *testing.T) {
	router := newTestRouter()
	createProjectProbeEnvironment(t, router, "env-probe-task", "probe-task", "project-x")
	agentToken := registerTestAgent(t, router, "agent-probe-x", "env-probe-task")

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-probe-x/heartbeat", strings.NewReader(`{"version":"v1-remote-probe","capabilities":["remote-probe","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+agentToken)
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	claimRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-probe-x/claim", strings.NewReader(`{"environmentId":"env-probe-task"}`))
	claimRequest.Header.Set("Content-Type", "application/json")
	claimRecorder := httptest.NewRecorder()
	router.ServeHTTP(claimRecorder, claimRequest)
	if claimRecorder.Code != http.StatusOK {
		t.Fatalf("expected claim status 200, got %d: %s", claimRecorder.Code, claimRecorder.Body.String())
	}

	probeRequest := httptest.NewRequest(http.MethodPost, "/api/environments/env-probe-task/remote-probe", nil)
	probeRecorder := httptest.NewRecorder()
	router.ServeHTTP(probeRecorder, probeRequest)
	if probeRecorder.Code != http.StatusOK {
		t.Fatalf("expected remote probe status 200, got %d: %s", probeRecorder.Code, probeRecorder.Body.String())
	}
	var probePayload struct {
		Data struct {
			TaskID        string `json:"taskId"`
			AgentID       string `json:"agentId"`
			EnvironmentID string `json:"environmentId"`
			Status        string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(probeRecorder.Body.Bytes(), &probePayload); err != nil {
		t.Fatalf("decode probe response: %v", err)
	}
	if probePayload.Data.AgentID != "agent-probe-x" || probePayload.Data.EnvironmentID != "env-probe-task" || probePayload.Data.Status != "PENDING" {
		t.Fatalf("unexpected probe response: %+v", probePayload.Data)
	}

	lease := leaseAgentTask(t, router, `{"agentId":"agent-probe-x","environmentId":"env-probe-task","maxTasks":1,"leaseSeconds":300}`)
	if !lease.Data.Leased || lease.Data.Task == nil {
		t.Fatalf("expected remote probe task lease, got %+v", lease.Data)
	}
	if lease.Data.Task.ID != probePayload.Data.TaskID || lease.Data.Task.Type != "probe" || lease.Data.Task.Action != "remote_resource_probe" {
		t.Fatalf("unexpected probe task: %+v", lease.Data.Task)
	}
	if lease.Data.Task.Payload["harborProjects"] != "project-x" {
		t.Fatalf("unexpected probe payload: %+v", lease.Data.Task.Payload)
	}
	if _, ok := lease.Data.Task.Payload["jenkinsViews"]; ok {
		t.Fatalf("unexpected probe payload: %+v", lease.Data.Task.Payload)
	}
}

func TestEnvironmentRemoteProbeResultUpdatesEnvironmentStatus(t *testing.T) {
	router := newTestRouter()
	createProjectProbeEnvironment(t, router, "env-probe-result", "probe-result", "project-x")
	agentToken := registerTestAgent(t, router, "agent-probe-result-x", "env-probe-result")

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-probe-result-x/heartbeat", strings.NewReader(`{"version":"v1-remote-probe","capabilities":["remote-probe","kubectl","http-check"]}`))
	heartbeatRequest.Header.Set("Content-Type", "application/json")
	heartbeatRequest.Header.Set("Authorization", "Bearer "+agentToken)
	heartbeatRecorder := httptest.NewRecorder()
	router.ServeHTTP(heartbeatRecorder, heartbeatRequest)
	if heartbeatRecorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	claimRequest := httptest.NewRequest(http.MethodPost, "/api/agents/agent-probe-result-x/claim", strings.NewReader(`{"environmentId":"env-probe-result"}`))
	claimRequest.Header.Set("Content-Type", "application/json")
	claimRecorder := httptest.NewRecorder()
	router.ServeHTTP(claimRecorder, claimRequest)
	if claimRecorder.Code != http.StatusOK {
		t.Fatalf("expected claim status 200, got %d: %s", claimRecorder.Code, claimRecorder.Body.String())
	}

	probeRequest := httptest.NewRequest(http.MethodPost, "/api/environments/env-probe-result/remote-probe", nil)
	probeRecorder := httptest.NewRecorder()
	router.ServeHTTP(probeRecorder, probeRequest)
	if probeRecorder.Code != http.StatusOK {
		t.Fatalf("expected remote probe status 200, got %d: %s", probeRecorder.Code, probeRecorder.Body.String())
	}
	var probePayload struct {
		Data struct {
			TaskID string `json:"taskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(probeRecorder.Body.Bytes(), &probePayload); err != nil {
		t.Fatalf("decode probe response: %v", err)
	}

	lease := leaseAgentTask(t, router, `{"agentId":"agent-probe-result-x","environmentId":"env-probe-result","maxTasks":1,"leaseSeconds":300}`)
	if !lease.Data.Leased || lease.Data.Task == nil || lease.Data.Task.ID != probePayload.Data.TaskID {
		t.Fatalf("expected remote probe task lease, got %+v", lease.Data)
	}

	resultMessage := `{"status":"DEGRADED","checks":[{"component":"Harbor 镜像项目","status":"DEGRADED","message":"项目环境 Harbor 无法访问 project-x","checkedAt":"2026-06-22T12:00:00+08:00"}]}`
	resultRequest := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/"+probePayload.Data.TaskID+"/result", strings.NewReader(`{"status":"SUCCESS","message":`+strconv.Quote(resultMessage)+`}`))
	resultRequest.Header.Set("Content-Type", "application/json")
	resultRequest.Header.Set("Authorization", "Bearer "+agentToken)
	resultRecorder := httptest.NewRecorder()
	router.ServeHTTP(resultRecorder, resultRequest)
	if resultRecorder.Code != http.StatusOK {
		t.Fatalf("expected result report status 200, got %d: %s", resultRecorder.Code, resultRecorder.Body.String())
	}

	environmentRequest := httptest.NewRequest(http.MethodGet, "/api/environments/env-probe-result", nil)
	environmentRecorder := httptest.NewRecorder()
	router.ServeHTTP(environmentRecorder, environmentRequest)
	if environmentRecorder.Code != http.StatusOK {
		t.Fatalf("expected environment detail status 200, got %d: %s", environmentRecorder.Code, environmentRecorder.Body.String())
	}
	var environmentPayload struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(environmentRecorder.Body.Bytes(), &environmentPayload); err != nil {
		t.Fatalf("decode environment response: %v", err)
	}
	if environmentPayload.Data.Status != "DEGRADED" {
		t.Fatalf("expected environment status DEGRADED, got %+v", environmentPayload.Data)
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/api/agent-tasks/"+probePayload.Data.TaskID+"/status", nil)
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, statusRequest)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("expected task status 200, got %d: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var statusPayload struct {
		Data struct {
			Probe remoteProbeResult `json:"probe"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode task status response: %v", err)
	}
	if statusPayload.Data.Probe.Status != "DEGRADED" || len(statusPayload.Data.Probe.Checks) != 1 {
		t.Fatalf("expected structured remote probe result, got %+v", statusPayload.Data.Probe)
	}
	if statusPayload.Data.Probe.Checks[0].Message != "项目环境 Harbor 无法访问 project-x" {
		t.Fatalf("unexpected probe check: %+v", statusPayload.Data.Probe.Checks[0])
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

func TestEnvironmentCheckUsesCachedScopesWithMockIntegrations(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/environments/env-local-prod/check", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Status string                         `json:"status"`
			Checks []integration.IntegrationCheck `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if payload.Data.Status != "HEALTHY" {
		t.Fatalf("expected cached local check healthy, got %+v", payload.Data)
	}
	if len(payload.Data.Checks) != 2 {
		t.Fatalf("expected cached local checks for configured scopes, got %+v", payload.Data.Checks)
	}
}

func TestEnvironmentCheckUsesCachedScopesForRemoteEnvironment(t *testing.T) {
	router := newTestRouter()
	request := httptest.NewRequest(http.MethodPost, "/api/environments/env-project-x-prod/check", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Status string                         `json:"status"`
			Checks []integration.IntegrationCheck `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if payload.Data.Status != "HEALTHY" {
		t.Fatalf("expected cached remote check healthy, got %+v", payload.Data)
	}
	if len(payload.Data.Checks) != 1 {
		t.Fatalf("expected remote harbor cached check only, got %+v", payload.Data.Checks)
	}
}

func TestEnvironmentCheckIgnoresRemoteJenkinsView(t *testing.T) {
	router := newTestRouter()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-missing-jenkins-view","name":"缺失 Jenkins 视图环境","code":"missing-jenkins-view","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"project-x","registryId":"harbor-local-prod","registryProject":"project-x","jenkinsId":"jenkins-local-prod","jenkinsView":"view-not-found","status":"HEALTHY"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	request := httptest.NewRequest(http.MethodPost, "/api/environments/env-missing-jenkins-view/check", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Status string                         `json:"status"`
			Checks []integration.IntegrationCheck `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if payload.Data.Status != "HEALTHY" {
		t.Fatalf("expected remote jenkins view to be ignored by V1 project environment checks, got %+v", payload.Data)
	}
}

func TestLocalEnvironmentMissingNamespaceIsListedAsDegraded(t *testing.T) {
	router := newTestRouter()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(`{"id":"env-missing-namespace","name":"缺失命名空间环境","code":"missing-namespace","type":"LOCAL","networkMode":"DIRECT","clusterId":"k8s-local-prod","namespace":"namespace-not-found","registryId":"harbor-local-prod","registryProject":"project-x","jenkinsId":"jenkins-local-prod","jenkinsView":"project-x","status":"HEALTHY"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected create environment status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/environments/env-missing-namespace", nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected get environment status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}
	var payload struct {
		Data domain.Environment `json:"data"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if payload.Data.Status != "DEGRADED" {
		t.Fatalf("expected missing namespace to be listed as degraded, got %+v", payload.Data)
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

func TestCreateImageReleaseQueuesAgentSync(t *testing.T) {
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
			Status        string `json:"status"`
			AgentTaskID   string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.ExecutionMode != "AGENT_IMAGE_SYNC" {
		t.Fatalf("expected AGENT_IMAGE_SYNC, got %s", payload.Data.ExecutionMode)
	}
	if payload.Data.Status != "PENDING_IMAGE_SYNC" || payload.Data.AgentTaskID == "" {
		t.Fatalf("expected queued agent image sync, got %+v", payload.Data)
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

type leaseResponsePayload struct {
	Data struct {
		Leased  bool   `json:"leased"`
		LeaseID string `json:"leaseId"`
		Message string `json:"message"`
		Task    *struct {
			ID            string            `json:"id"`
			Type          string            `json:"type"`
			Action        string            `json:"action"`
			Status        string            `json:"status"`
			AgentID       string            `json:"agentId"`
			EnvironmentID string            `json:"environmentId"`
			LeaseID       string            `json:"leaseId"`
			Payload       map[string]string `json:"payload"`
		} `json:"task"`
	} `json:"data"`
}

func createReleaseAgentTask(t *testing.T, router http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/releases", strings.NewReader(`{"type":"SERVICE_RELEASE","releaseSource":"JENKINS_JOB","targetEnvironmentId":"env-project-x-prod","agentId":"agent-project-x","jenkins":{"jobName":"mock-release","branch":"main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected release create status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			AgentTaskID string `json:"agentTaskId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if payload.Data.AgentTaskID == "" {
		t.Fatal("expected agent task id")
	}
	return payload.Data.AgentTaskID
}

func heartbeatAgent(t *testing.T, router http.Handler, agentID string) {
	t.Helper()
	agentToken := agentTokenForTest(t, router, agentID, "env-project-x-prod")
	request := httptest.NewRequest(http.MethodPost, "/api/agents/"+agentID+"/heartbeat", strings.NewReader(`{"version":"v1-mock","capabilities":["mock-executor","image-sync","kubectl","http-check"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+agentToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected heartbeat status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func leaseAgentTask(t *testing.T, router http.Handler, body string) leaseResponsePayload {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/agent-tasks/lease", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+agentTokenForLeaseBody(t, router, body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected lease status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload leaseResponsePayload
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}
	return payload
}

func agentTokenForLeaseBody(t *testing.T, router http.Handler, body string) string {
	t.Helper()
	var request struct {
		AgentID       string `json:"agentId"`
		EnvironmentID string `json:"environmentId"`
	}
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("decode lease body: %v", err)
	}
	return agentTokenForTest(t, router, request.AgentID, strings.TrimSpace(request.EnvironmentID))
}

func agentTokenForTest(t *testing.T, router http.Handler, agentID string, environmentID string) string {
	t.Helper()
	if value, ok := testAgentTokens.Load(agentID); ok {
		return value.(string)
	}
	return registerTestAgent(t, router, agentID, environmentID)
}

func registerTestAgent(t *testing.T, router http.Handler, agentID string, environmentID string) string {
	t.Helper()
	if value, ok := testAgentTokens.Load(agentID); ok {
		return value.(string)
	}
	createTokenRequest := httptest.NewRequest(http.MethodPost, "/api/agents/register-token", strings.NewReader(`{"agentId":"`+agentID+`","environmentId":"`+environmentID+`","ttlMinutes":10}`))
	createTokenRequest.Header.Set("Content-Type", "application/json")
	createTokenRecorder := httptest.NewRecorder()
	router.ServeHTTP(createTokenRecorder, createTokenRequest)
	if createTokenRecorder.Code != http.StatusCreated {
		t.Fatalf("expected register token status 201, got %d: %s", createTokenRecorder.Code, createTokenRecorder.Body.String())
	}
	var tokenPayload struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createTokenRecorder.Body.Bytes(), &tokenPayload); err != nil {
		t.Fatalf("decode register token response: %v", err)
	}
	if tokenPayload.Data.Token == "" {
		t.Fatal("expected register token")
	}

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/agents/register", strings.NewReader(`{"agentId":"`+agentID+`","environmentId":"`+environmentID+`","registerToken":"`+tokenPayload.Data.Token+`","version":"v1-test","capabilities":["mock-executor","image-sync","kubectl","http-check"]}`))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerRecorder := httptest.NewRecorder()
	router.ServeHTTP(registerRecorder, registerRequest)
	if registerRecorder.Code != http.StatusCreated {
		t.Fatalf("expected agent register status 201, got %d: %s", registerRecorder.Code, registerRecorder.Body.String())
	}
	var registerPayload struct {
		Data struct {
			AgentToken string `json:"agentToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(registerRecorder.Body.Bytes(), &registerPayload); err != nil {
		t.Fatalf("decode agent register response: %v", err)
	}
	if registerPayload.Data.AgentToken == "" {
		t.Fatal("expected agent runtime token")
	}
	testAgentTokens.Store(agentID, registerPayload.Data.AgentToken)
	return registerPayload.Data.AgentToken
}

func createProjectProbeEnvironment(t *testing.T, router http.Handler, id string, code string, scope string) {
	t.Helper()
	body := `{"id":"` + id + `","name":"项目探测测试环境","code":"` + code + `","type":"PROJECT","networkMode":"AGENT","clusterId":"k8s-local-prod","namespace":"` + scope + `","registryId":"harbor-local-prod","registryProject":"` + scope + `","status":"HEALTHY"}`
	request := httptest.NewRequest(http.MethodPost, "/api/environments", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected create probe environment status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func newTestRouter() http.Handler {
	return newTestRouterWithRepository(newTestRepository())
}

func newTestRepository() *repository.MockRepository {
	repo, err := repository.NewMockRepository()
	if err != nil {
		panic(err)
	}
	seedTestEnvironments(repo)
	seedTestAgentTokens()
	return repo
}

func seedTestAgentTokens() {
	testAgentTokens.Range(func(key any, _ any) bool {
		testAgentTokens.Delete(key)
		return true
	})
	testAgentTokens.Store("agent-project-x", "agent-project-x-test-token")
	testAgentTokens.Store("agent-project-y", "agent-project-y-test-token")
	testAgentTokens.Store("agent-project-z", "agent-project-z-test-token")
}

func newTestRouterWithRepository(repo repository.Store) http.Handler {
	return NewRouter(repo, nil, agent.NewProtocolStore(), integration.NewMockSuite())
}

func newPermissionTestRouter(t *testing.T, user domain.CurrentUser) http.Handler {
	t.Helper()
	repo, err := repository.NewMockRepository()
	if err != nil {
		t.Fatalf("load mock repository: %v", err)
	}
	seedTestEnvironments(repo)
	repo.SetCurrentUserForTest(user)
	handler := NewHandler(repo, nil, agent.NewProtocolStore(), integration.NewMockSuite())
	router := gin.New()
	router.POST("/api/releases", handler.CreateRelease)
	router.POST("/api/deploy-tasks", handler.CreateDeployTask)
	return router
}

func seedTestEnvironments(repo *repository.MockRepository) {
	if _, err := repo.CreateKubernetesCluster(domain.KubernetesCluster{
		ID:        "k8s-local-prod",
		Name:      "本地生产 K8s",
		APIServer: "https://k8s.local",
	}); err != nil {
		panic(err)
	}
	if _, err := repo.CreateHarborRegistry(domain.HarborRegistry{
		ID:   "harbor-local-prod",
		Name: "本地生产 Harbor",
		URL:  "https://harbor.local",
	}); err != nil {
		panic(err)
	}
	if _, err := repo.CreateJenkinsInstance(domain.JenkinsInstance{
		ID:   "jenkins-local-prod",
		Name: "本地生产 Jenkins",
		URL:  "https://jenkins.local",
	}); err != nil {
		panic(err)
	}
	if _, ok, err := repo.UpdateKubernetesClusterProbe("k8s-local-prod", "HEALTHY", "", []string{"default", "project-x"}, time.Now()); err != nil || !ok {
		panic("seed kubernetes probe failed")
	}
	if _, ok, err := repo.UpdateHarborRegistryProbe("harbor-local-prod", "HEALTHY", "", []string{"project-x"}, time.Now()); err != nil || !ok {
		panic("seed harbor probe failed")
	}
	if _, ok, err := repo.UpdateJenkinsInstanceProbe("jenkins-local-prod", "HEALTHY", "", []string{"project-x"}, []string{"mock-release"}, time.Now()); err != nil || !ok {
		panic("seed jenkins probe failed")
	}
	items := []domain.Environment{
		{
			ID:              "env-local-prod",
			Name:            "本地生产环境",
			Code:            "local-prod",
			Type:            "LOCAL",
			NetworkMode:     "DIRECT",
			ClusterID:       "k8s-local-prod",
			Namespace:       "default",
			RegistryID:      "harbor-local-prod",
			RegistryProject: "project-x",
			Status:          "HEALTHY",
		},
		{
			ID:              "env-project-x-prod",
			Name:            "项目 X 生产",
			Code:            "project-x-prod",
			Type:            "PROJECT",
			NetworkMode:     "AGENT",
			ClusterID:       "k8s-local-prod",
			Namespace:       "project-x",
			RegistryID:      "harbor-local-prod",
			RegistryProject: "project-x",
			Status:          "HEALTHY",
		},
		{
			ID:              "env-project-z-prod",
			Name:            "项目 Z 生产",
			Code:            "project-z-prod",
			Type:            "PROJECT",
			NetworkMode:     "AGENT",
			ClusterID:       "k8s-local-prod",
			Namespace:       "project-z",
			RegistryID:      "harbor-local-prod",
			RegistryProject: "project-z",
			Status:          "UNKNOWN",
		},
	}
	for _, item := range items {
		if _, err := repo.CreateEnvironment(item); err != nil {
			panic(err)
		}
	}
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

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
