package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"ops-release-platform/agent/internal/reporter"
)

type harborProject struct {
	Name string `json:"name"`
}

type runtimeStatusLogger struct {
	lastKubernetes string
	lastHarbor     string
}

func (e *ProbeExecutor) RuntimeStatus(ctx context.Context) reporter.RuntimeStatus {
	kubernetes := e.kubernetesStatus(ctx)
	status := reporter.RuntimeStatus{
		Kubernetes: kubernetes,
		Harbor:     e.harborStatus(ctx, kubernetes.Workloads),
	}
	e.logRuntimeStatus(status)
	return status
}

func (e *ProbeExecutor) kubernetesStatus(ctx context.Context) reporter.RuntimeComponentStatus {
	if strings.TrimSpace(e.cfg.Kubeconfig) == "" {
		return newRuntimeComponentStatus("UNKNOWN", "未配置 AGENT_KUBECONFIG", nil)
	}
	client, err := newKubernetesClient(e.cfg.Kubeconfig, e.cfg.HTTPTimeout, e.httpClient.Transport)
	if err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "K8s 配置无效："+err.Error(), nil)
	}
	items, err := client.listNamespaces(ctx)
	if err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "K8s API 访问失败："+err.Error(), nil)
	}
	workloads, err := client.listWorkloads(ctx, items)
	if err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "K8s 工作负载访问失败："+err.Error(), compactRuntimeItems(items))
	}
	status := newRuntimeComponentStatus("HEALTHY", fmt.Sprintf("已发现 %d 个命名空间，%d 个工作负载", len(items), len(workloads)), compactRuntimeItems(items))
	status.Workloads = workloads
	return status
}

func (e *ProbeExecutor) harborStatus(ctx context.Context, workloads []reporter.RuntimeWorkload) reporter.RuntimeComponentStatus {
	if e.cfg.HarborURL == "" || e.cfg.HarborUsername == "" || e.cfg.HarborPassword == "" {
		return newRuntimeComponentStatus("UNKNOWN", "未配置 Harbor 地址或账号", nil)
	}
	registryHost, registryErr := e.harborRegistryHost(ctx)
	endpoint := strings.TrimRight(e.cfg.HarborURL, "/") + "/api/v2.0/projects?page_size=100"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "构造 Harbor 请求失败："+err.Error(), nil)
	}
	request.SetBasicAuth(e.cfg.HarborUsername, e.cfg.HarborPassword)
	response, err := e.httpClient.Do(request)
	if err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "Harbor 访问失败："+err.Error(), nil)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return newRuntimeComponentStatus("UNHEALTHY", fmt.Sprintf("Harbor 返回状态码 %d", response.StatusCode), nil)
	}
	var payload []harborProject
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return newRuntimeComponentStatus("UNHEALTHY", "Harbor 返回数据无法解析："+err.Error(), nil)
	}
	items := make([]string, 0, len(payload))
	for _, item := range payload {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			items = append(items, name)
		}
	}
	if registryHost == "" {
		registryHost = inferRegistryHostFromWorkloads(workloads, items)
	}
	message := fmt.Sprintf("已发现 %d 个镜像项目", len(items))
	if registryHost == "" && registryErr != nil {
		message += "，未能自动识别 registry：" + registryErr.Error()
	}
	status := newRuntimeComponentStatus("HEALTHY", message, compactRuntimeItems(items))
	status.Endpoint = e.cfg.HarborURL
	status.RegistryHost = registryHost
	return status
}

func (e *ProbeExecutor) harborRegistryHost(ctx context.Context) (string, error) {
	endpoint := strings.TrimRight(e.cfg.HarborURL, "/") + "/api/v2.0/systeminfo"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.SetBasicAuth(e.cfg.HarborUsername, e.cfg.HarborPassword)
	response, err := e.httpClient.Do(request)
	if err != nil {
		if registryHost := e.harborRegistryHostFromConfigurations(ctx); registryHost != "" {
			return registryHost, nil
		}
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if registryHost := e.harborRegistryHostFromConfigurations(ctx); registryHost != "" {
			return registryHost, nil
		}
		return "", fmt.Errorf("Harbor 返回状态码 %d", response.StatusCode)
	}
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", err
	}
	if registryHost := registryHostFromPayload(payload); registryHost != "" {
		return registryHost, nil
	}
	return e.harborRegistryHostFromConfigurations(ctx), nil
}

func (e *ProbeExecutor) harborRegistryHostFromConfigurations(ctx context.Context) string {
	endpoint := strings.TrimRight(e.cfg.HarborURL, "/") + "/api/v2.0/configurations"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	request.SetBasicAuth(e.cfg.HarborUsername, e.cfg.HarborPassword)
	response, err := e.httpClient.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ""
	}
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return ""
	}
	return registryHostFromPayload(payload)
}

func registryHostFromPayload(payload any) string {
	values := map[string]string{}
	collectRegistryFields(payload, values)
	for _, key := range []string{"registry_url", "registryHost", "registry_host", "external_url", "externalURL"} {
		if host := normalizedImageRegistry(values[key]); host != "" {
			return host
		}
	}
	return ""
}

func collectRegistryFields(value any, output map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if stringValue, ok := item.(string); ok {
				output[key] = stringValue
				continue
			}
			if nested, ok := item.(map[string]any); ok {
				if stringValue, ok := nested["value"].(string); ok {
					output[key] = stringValue
				}
			}
			collectRegistryFields(item, output)
		}
	case []any:
		for _, item := range typed {
			collectRegistryFields(item, output)
		}
	}
}

func inferRegistryHostFromWorkloads(workloads []reporter.RuntimeWorkload, projects []string) string {
	projectSet := map[string]bool{}
	for _, project := range projects {
		if name := strings.TrimSpace(project); name != "" {
			projectSet[name] = true
		}
	}
	if len(projectSet) == 0 {
		return ""
	}
	candidates := map[string]bool{}
	for _, workload := range workloads {
		for _, container := range workload.Containers {
			image := parseRuntimeImage(container.Image)
			if image.Registry == "" || image.Registry == "docker.io" {
				continue
			}
			if projectSet[image.Project] {
				candidates[image.Registry] = true
			}
		}
	}
	if len(candidates) != 1 {
		return ""
	}
	for candidate := range candidates {
		return candidate
	}
	return ""
}

type runtimeImage struct {
	Registry string
	Project  string
}

func parseRuntimeImage(image string) runtimeImage {
	image = strings.TrimSpace(image)
	name := image
	if at := strings.Index(name, "@"); at >= 0 {
		name = name[:at]
	}
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		if colon := strings.LastIndex(name, ":"); colon > slash {
			name = name[:colon]
		}
	} else if colon := strings.LastIndex(name, ":"); colon >= 0 {
		name = name[:colon]
	}
	parts := strings.Split(name, "/")
	registry := "docker.io"
	repositoryParts := parts
	if len(parts) > 1 && looksLikeRegistry(parts[0]) {
		registry = normalizedImageRegistry(parts[0])
		repositoryParts = parts[1:]
	}
	project := "library"
	if len(repositoryParts) > 1 {
		project = repositoryParts[0]
	}
	return runtimeImage{Registry: registry, Project: project}
}

func looksLikeRegistry(value string) bool {
	return strings.Contains(value, ".") || strings.Contains(value, ":") || value == "localhost"
}

func normalizedImageRegistry(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(value), "http://"), "https://")
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	return strings.TrimRight(strings.ToLower(host), "/")
}

func (e *ProbeExecutor) logRuntimeStatus(status reporter.RuntimeStatus) {
	if e.statusLogger == nil {
		e.statusLogger = &runtimeStatusLogger{}
	}
	e.statusLogger.logComponent("kubernetes", status.Kubernetes, &e.statusLogger.lastKubernetes)
	e.statusLogger.logComponent("harbor", status.Harbor, &e.statusLogger.lastHarbor)
}

func (l *runtimeStatusLogger) logComponent(name string, status reporter.RuntimeComponentStatus, last *string) {
	current := status.Status + "|" + status.Message + "|" + strings.Join(status.Items, ",") + fmt.Sprintf("|workloads=%d", len(status.Workloads))
	if current == *last {
		return
	}
	*last = current

	if len(status.Items) == 0 {
		log.Printf("%s status=%s message=%s", name, status.Status, status.Message)
		return
	}
	log.Printf("%s status=%s message=%s items=%s", name, status.Status, status.Message, strings.Join(status.Items, ","))
}

func newRuntimeComponentStatus(status string, message string, items []string) reporter.RuntimeComponentStatus {
	return reporter.RuntimeComponentStatus{
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Items:     items,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactRuntimeItems(input []string) []string {
	seen := map[string]bool{}
	items := make([]string, 0, len(input))
	for _, value := range input {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}
