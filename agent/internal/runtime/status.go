package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	status := reporter.RuntimeStatus{
		Kubernetes: e.kubernetesStatus(ctx),
		Harbor:     e.harborStatus(ctx),
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
	return newRuntimeComponentStatus("HEALTHY", fmt.Sprintf("已发现 %d 个命名空间", len(items)), compactRuntimeItems(items))
}

func (e *ProbeExecutor) harborStatus(ctx context.Context) reporter.RuntimeComponentStatus {
	if e.cfg.HarborURL == "" || e.cfg.HarborUsername == "" || e.cfg.HarborPassword == "" {
		return newRuntimeComponentStatus("UNKNOWN", "未配置 Harbor 地址或账号", nil)
	}
	endpoint := e.cfg.HarborURL + "/api/v2.0/projects?page_size=100"
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
	return newRuntimeComponentStatus("HEALTHY", fmt.Sprintf("已发现 %d 个镜像项目", len(items)), compactRuntimeItems(items))
}

func (e *ProbeExecutor) logRuntimeStatus(status reporter.RuntimeStatus) {
	if e.statusLogger == nil {
		e.statusLogger = &runtimeStatusLogger{}
	}
	e.statusLogger.logComponent("kubernetes", status.Kubernetes, &e.statusLogger.lastKubernetes)
	e.statusLogger.logComponent("harbor", status.Harbor, &e.statusLogger.lastHarbor)
}

func (l *runtimeStatusLogger) logComponent(name string, status reporter.RuntimeComponentStatus, last *string) {
	current := status.Status + "|" + status.Message + "|" + strings.Join(status.Items, ",")
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
