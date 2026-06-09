package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL       string
	agentID       string
	environmentID string
	token         string
	httpClient    *http.Client
}

type apiResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type LeaseResponse struct {
	Leased  bool   `json:"leased"`
	LeaseID string `json:"leaseId"`
	Task    *Task  `json:"task"`
}

type Task struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Action        string            `json:"action"`
	Status        string            `json:"status"`
	Step          string            `json:"step"`
	AgentID       string            `json:"agentId"`
	EnvironmentID string            `json:"environmentId"`
	LeaseID       string            `json:"leaseId"`
	Payload       map[string]string `json:"payload"`
}

func NewClient(baseURL string, agentID string, environmentID string, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL:       baseURL,
		agentID:       agentID,
		environmentID: environmentID,
		token:         token,
		httpClient:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Heartbeat(ctx context.Context, version string, capabilities []string) error {
	return c.post(ctx, fmt.Sprintf("/api/agents/%s/heartbeat", c.agentID), map[string]any{
		"version":      version,
		"capabilities": capabilities,
	}, nil)
}

func (c *Client) Lease(ctx context.Context, leaseSeconds int) (LeaseResponse, error) {
	var response apiResponse[LeaseResponse]
	err := c.post(ctx, "/api/agent-tasks/lease", map[string]any{
		"agentId":       c.agentID,
		"environmentId": c.environmentID,
		"maxTasks":      1,
		"leaseSeconds":  leaseSeconds,
	}, &response)
	if err != nil {
		return LeaseResponse{}, err
	}
	if response.Code != "OK" {
		return LeaseResponse{}, fmt.Errorf("lease rejected: %s", response.Message)
	}
	return response.Data, nil
}

func (c *Client) ReportStep(ctx context.Context, taskID string, step string, status string) error {
	return c.post(ctx, fmt.Sprintf("/api/agent-tasks/%s/steps", taskID), map[string]string{
		"step":   step,
		"status": status,
	}, nil)
}

func (c *Client) AppendLog(ctx context.Context, taskID string, line string) error {
	return c.post(ctx, fmt.Sprintf("/api/agent-tasks/%s/logs", taskID), map[string]string{
		"line": line,
	}, nil)
}

func (c *Client) ReportResult(ctx context.Context, taskID string, status string, message string) error {
	return c.post(ctx, fmt.Sprintf("/api/agent-tasks/%s/result", taskID), map[string]string{
		"status":  status,
		"message": message,
	}, nil)
}

func (c *Client) post(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("platform api %s returned %d: %s", path, response.StatusCode, string(content))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(content, out)
}
