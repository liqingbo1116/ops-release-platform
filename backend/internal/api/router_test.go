package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCoreMockAPIs(t *testing.T) {
	router := NewRouter(nil)
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
		{name: "baseline detail", method: http.MethodGet, path: "/api/baselines/BL-20260607-0001", statusCode: http.StatusOK},
		{name: "baseline compare", method: http.MethodPost, path: "/api/baselines/BL-20260607-0001/compare", body: "{}", statusCode: http.StatusOK},
		{name: "create release", method: http.MethodPost, path: "/api/releases", body: "{}", statusCode: http.StatusCreated},
		{name: "release detail", method: http.MethodGet, path: "/api/releases/REL-20260607-031", statusCode: http.StatusOK},
		{name: "deploy tasks", method: http.MethodGet, path: "/api/deploy-tasks", statusCode: http.StatusOK},
		{name: "create deploy task", method: http.MethodPost, path: "/api/deploy-tasks", body: "{}", statusCode: http.StatusCreated},
		{name: "deploy detail", method: http.MethodGet, path: "/api/deploy-tasks/DEP-20260607-009", statusCode: http.StatusOK},
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
	router := NewRouter(nil)
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

func TestUnknownRoute(t *testing.T) {
	router := NewRouter(nil)
	request := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
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
