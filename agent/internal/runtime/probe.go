package runtime

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"ops-release-platform/agent/internal/config"
	"ops-release-platform/agent/internal/reporter"
)

type ProbeExecutor struct {
	cfg        config.Config
	client     *reporter.Client
	httpClient *http.Client
}

type probeResult struct {
	Status  string       `json:"status"`
	Checks []probeCheck `json:"checks"`
}

type probeCheck struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CheckedAt string `json:"checkedAt"`
}

func NewProbeExecutor(cfg config.Config, client *reporter.Client) *ProbeExecutor {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.HarborInsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &ProbeExecutor{
		cfg:        cfg,
		client:     client,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout, Transport: transport},
	}
}

func (e *ProbeExecutor) Execute(ctx context.Context, task reporter.Task) error {
	if task.Type != "probe" || task.Action != "remote_resource_probe" {
		return fmt.Errorf("unsupported task for probe executor: type=%s action=%s", task.Type, task.Action)
	}
	if err := e.client.ReportStep(ctx, task.ID, "remote-probe", "RUNNING"); err != nil {
		return err
	}
	if err := e.client.AppendLog(ctx, task.ID, "开始远程探测环境资源"); err != nil {
		return err
	}

	result := probeResult{Checks: []probeCheck{}}
	for _, namespace := range csvValues(task.Payload["k8sNamespaces"]) {
		result.Checks = append(result.Checks, e.checkNamespace(ctx, namespace))
	}
	for _, project := range csvValues(task.Payload["harborProjects"]) {
		result.Checks = append(result.Checks, e.checkHarborProject(ctx, project))
	}
	result.Status = probeStatus(result.Checks)

	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	taskStatus := "SUCCESS"
	if result.Status == "UNHEALTHY" {
		taskStatus = "FAILED"
	}
	if err := e.client.AppendLog(ctx, task.ID, "远程探测完成，环境状态："+result.Status); err != nil {
		return err
	}
	return e.client.ReportResult(ctx, task.ID, taskStatus, string(body))
}

func (e *ProbeExecutor) checkNamespace(ctx context.Context, namespace string) probeCheck {
	if strings.TrimSpace(e.cfg.Kubeconfig) == "" {
		return newProbeCheck("K8s 命名空间", "DEGRADED", "未配置 AGENT_KUBECONFIG，无法验证 "+namespace)
	}
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", e.cfg.Kubeconfig, "get", "namespace", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return newProbeCheck("K8s 命名空间", "UNHEALTHY", namespace+" 不存在或无法访问："+message)
	}
	return newProbeCheck("K8s 命名空间", "HEALTHY", namespace+" 存在")
}

func (e *ProbeExecutor) checkHarborProject(ctx context.Context, project string) probeCheck {
	if e.cfg.HarborURL == "" || e.cfg.HarborUsername == "" || e.cfg.HarborPassword == "" {
		return newProbeCheck("Harbor 镜像项目", "DEGRADED", "未配置 Harbor 地址或账号，无法验证 "+project)
	}
	endpoint := e.cfg.HarborURL + "/api/v2.0/projects/" + url.PathEscape(project)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return newProbeCheck("Harbor 镜像项目", "UNHEALTHY", "构造 Harbor 探测请求失败："+err.Error())
	}
	request.SetBasicAuth(e.cfg.HarborUsername, e.cfg.HarborPassword)
	response, err := e.httpClient.Do(request)
	if err != nil {
		return newProbeCheck("Harbor 镜像项目", "UNHEALTHY", "Harbor 访问失败："+err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return newProbeCheck("Harbor 镜像项目", "HEALTHY", project+" 存在")
	}
	if response.StatusCode == http.StatusNotFound {
		return newProbeCheck("Harbor 镜像项目", "UNHEALTHY", project+" 不存在")
	}
	return newProbeCheck("Harbor 镜像项目", "UNHEALTHY", fmt.Sprintf("Harbor 返回状态码 %d，无法确认 %s", response.StatusCode, project))
}

func newProbeCheck(component string, status string, message string) probeCheck {
	return probeCheck{
		Component: component,
		Status:    status,
		Message:   message,
		CheckedAt: time.Now().Format(time.RFC3339),
	}
}

func probeStatus(checks []probeCheck) string {
	if len(checks) == 0 {
		return "UNKNOWN"
	}
	for _, check := range checks {
		if check.Status == "UNHEALTHY" {
			return "UNHEALTHY"
		}
	}
	for _, check := range checks {
		if check.Status != "HEALTHY" {
			return "DEGRADED"
		}
	}
	return "HEALTHY"
}

func csvValues(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
