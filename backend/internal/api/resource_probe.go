package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"

	"ops-release-platform/backend/internal/domain"
)

const resourceProbeTimeout = 10 * time.Second

func (h *Handler) TestKubernetesCluster(c *gin.Context) {
	h.probeKubernetesCluster(c, false)
}

func (h *Handler) RefreshKubernetesCluster(c *gin.Context) {
	h.probeKubernetesCluster(c, true)
}

func (h *Handler) TestHarborRegistry(c *gin.Context) {
	h.probeHarborRegistry(c, false)
}

func (h *Handler) RefreshHarborRegistry(c *gin.Context) {
	h.probeHarborRegistry(c, true)
}

func (h *Handler) TestJenkinsInstance(c *gin.Context) {
	h.probeJenkinsInstance(c, false)
}

func (h *Handler) RefreshJenkinsInstance(c *gin.Context) {
	h.probeJenkinsInstance(c, true)
}

func (h *Handler) probeKubernetesCluster(c *gin.Context, refresh bool) {
	id := c.Param("id")
	cluster, ok := h.repo.GetKubernetesCluster(id)
	if !ok {
		NotFound(c, "kubernetes cluster not found")
		return
	}
	namespaces, err := checkKubernetesCluster(c.Request.Context(), cluster, refresh)
	checkedAt := time.Now()
	status, message := probeResult(err, "kubernetes connection ok")
	item, ok, updateErr := h.repo.UpdateKubernetesClusterProbe(id, status, message, namespaces, checkedAt)
	if updateErr != nil {
		BadRequest(c, "update kubernetes probe result failed")
		return
	}
	if !ok {
		NotFound(c, "kubernetes cluster not found")
		return
	}
	if err != nil {
		BadRequest(c, message)
		return
	}
	OK(c, item)
}

func (h *Handler) probeHarborRegistry(c *gin.Context, refresh bool) {
	id := c.Param("id")
	registry, ok := h.repo.GetHarborRegistry(id)
	if !ok {
		NotFound(c, "harbor registry not found")
		return
	}
	projects, registryHost, err := checkHarborRegistry(c.Request.Context(), registry, refresh)
	checkedAt := time.Now()
	status, message := probeResult(err, "harbor connection ok")
	item, ok, updateErr := h.repo.UpdateHarborRegistryProbe(id, status, message, projects, registryHost, checkedAt)
	if updateErr != nil {
		BadRequest(c, "update harbor probe result failed")
		return
	}
	if !ok {
		NotFound(c, "harbor registry not found")
		return
	}
	if err != nil {
		BadRequest(c, message)
		return
	}
	OK(c, item)
}

func (h *Handler) probeJenkinsInstance(c *gin.Context, refresh bool) {
	id := c.Param("id")
	instance, ok := h.repo.GetJenkinsInstance(id)
	if !ok {
		NotFound(c, "jenkins instance not found")
		return
	}
	views, jobs, pipelines, err := checkJenkinsInstance(c.Request.Context(), instance, refresh)
	checkedAt := time.Now()
	status, message := probeResult(err, "jenkins connection ok")
	item, ok, updateErr := h.repo.UpdateJenkinsInstanceProbe(id, status, message, views, jobs, pipelines, checkedAt)
	if updateErr != nil {
		BadRequest(c, "update jenkins probe result failed")
		return
	}
	if !ok {
		NotFound(c, "jenkins instance not found")
		return
	}
	if err != nil {
		BadRequest(c, message)
		return
	}
	OK(c, item)
}

func probeResult(err error, successMessage string) (string, string) {
	if err != nil {
		return "UNHEALTHY", trimProbeMessage(err.Error())
	}
	return "HEALTHY", successMessage
}

func trimProbeMessage(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 480 {
		return message[:480]
	}
	return message
}

func checkKubernetesCluster(ctx context.Context, cluster domain.KubernetesCluster, refresh bool) ([]string, error) {
	client, server, err := kubernetesClientFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	if _, err := getJSON(ctx, client, joinURL(server, "/readyz"), nil); err != nil {
		return nil, err
	}
	if !refresh {
		return nil, nil
	}
	body, err := getJSON(ctx, client, joinURL(server, "/api/v1/namespaces"), nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse kubernetes namespaces failed: %w", err)
	}
	namespaces := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if name := strings.TrimSpace(item.Metadata.Name); name != "" {
			namespaces = append(namespaces, name)
		}
	}
	return compactProbeList(namespaces), nil
}

func checkHarborRegistry(ctx context.Context, registry domain.HarborRegistry, refresh bool) ([]string, string, error) {
	client := basicAuthClient(registry.InsecureSkipTLSVerify)
	headers := basicAuthHeaders(registry.Username, registry.Password)
	baseURL := normalizeProbeURL(registry.URL, registry.Scheme)
	systemInfoBody, err := getJSON(ctx, client, joinURL(baseURL, "/api/v2.0/systeminfo"), headers)
	if err != nil {
		return nil, "", err
	}
	registryHost := harborRegistryHostFromPayload(systemInfoBody)
	if registryHost == "" {
		registryHost = harborRegistryHostFromConfigurations(ctx, client, baseURL, headers)
	}
	if !refresh {
		return nil, registryHost, nil
	}
	body, err := getJSON(ctx, client, joinURL(baseURL, "/api/v2.0/projects?page_size=100"), headers)
	if err != nil {
		return nil, registryHost, err
	}
	var response []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, registryHost, fmt.Errorf("parse harbor projects failed: %w", err)
	}
	projects := make([]string, 0, len(response))
	for _, item := range response {
		if name := strings.TrimSpace(item.Name); name != "" {
			projects = append(projects, name)
		}
	}
	return compactProbeList(projects), registryHost, nil
}

func harborRegistryHostFromConfigurations(ctx context.Context, client *http.Client, baseURL string, headers map[string]string) string {
	body, err := getJSON(ctx, client, joinURL(baseURL, "/api/v2.0/configurations"), headers)
	if err != nil {
		return ""
	}
	return harborRegistryHostFromPayload(body)
}

func harborRegistryHostFromPayload(body []byte) string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	values := map[string]string{}
	collectHarborRegistryFields(payload, values)
	for _, key := range []string{"registry_url", "registryHost", "registry_host", "external_url", "externalURL"} {
		if host := normalizedImageRegistry(values[key]); host != "" {
			return host
		}
	}
	return ""
}

func collectHarborRegistryFields(value any, output map[string]string) {
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
			collectHarborRegistryFields(item, output)
		}
	case []any:
		for _, item := range typed {
			collectHarborRegistryFields(item, output)
		}
	}
}

func checkJenkinsInstance(ctx context.Context, instance domain.JenkinsInstance, refresh bool) ([]string, []string, []domain.JenkinsPipeline, error) {
	client := basicAuthClient(instance.InsecureSkipTLSVerify)
	headers := basicAuthHeaders(instance.Username, instance.Token)
	tree := "views[name,url],jobs[name,url,property[parameterDefinitions[name,type,description,defaultParameterValue[value]]]]"
	if !refresh {
		tree = "mode"
	}
	body, err := getJSON(ctx, client, joinURL(instance.URL, "/api/json?tree="+url.QueryEscape(tree)), headers)
	if err != nil {
		return nil, nil, nil, err
	}
	if !refresh {
		return nil, nil, nil, nil
	}
	var response jenkinsRootResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil, nil, fmt.Errorf("parse jenkins views/jobs failed: %w", err)
	}
	views := make([]string, 0, len(response.Views))
	for _, item := range response.Views {
		if name := strings.TrimSpace(item.Name); name != "" {
			views = append(views, name)
		}
	}
	jobs := make([]string, 0, len(response.Jobs))
	for _, item := range response.Jobs {
		if name := strings.TrimSpace(item.Name); name != "" {
			jobs = append(jobs, name)
		}
	}
	pipelines := make([]domain.JenkinsPipeline, 0, len(response.Jobs))
	for _, item := range response.Jobs {
		if pipeline := jenkinsPipelineFromJob(item, "", ""); pipeline.Name != "" {
			pipelines = append(pipelines, pipeline)
		}
	}
	for _, view := range response.Views {
		viewName := strings.TrimSpace(view.Name)
		if viewName == "" {
			continue
		}
		viewJobs, err := fetchJenkinsViewJobs(ctx, client, headers, instance.URL, view)
		if err != nil {
			log.Printf("jenkins instance %s view %s pipeline probe failed: %v", instance.ID, viewName, err)
			continue
		}
		for _, item := range viewJobs {
			if name := strings.TrimSpace(item.Name); name != "" {
				jobs = append(jobs, name)
			}
			if pipeline := jenkinsPipelineFromJob(item, viewName, view.URL); pipeline.Name != "" {
				pipelines = append(pipelines, pipeline)
			}
		}
	}
	return compactProbeList(views), compactProbeList(jobs), compactProbePipelines(pipelines), nil
}

type jenkinsRootResponse struct {
	Views []jenkinsViewResponse `json:"views"`
	Jobs  []jenkinsJobResponse  `json:"jobs"`
}

type jenkinsViewResponse struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type jenkinsJobResponse struct {
	Name     string                       `json:"name"`
	URL      string                       `json:"url"`
	Property []jenkinsJobPropertyResponse `json:"property"`
}

type jenkinsJobPropertyResponse struct {
	ParameterDefinitions []jenkinsParameterDefinitionResponse `json:"parameterDefinitions"`
}

type jenkinsParameterDefinitionResponse struct {
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	Description           string `json:"description"`
	DefaultParameterValue *struct {
		Value any `json:"value"`
	} `json:"defaultParameterValue"`
}

func fetchJenkinsViewJobs(ctx context.Context, client *http.Client, headers map[string]string, baseURL string, view jenkinsViewResponse) ([]jenkinsJobResponse, error) {
	tree := "jobs[name,url,property[parameterDefinitions[name,type,description,defaultParameterValue[value]]]]"
	endpoints := jenkinsViewEndpoints(baseURL, view)
	var lastErr error
	for _, endpoint := range endpoints {
		body, err := getJSON(ctx, client, joinURL(endpoint, "/api/json?tree="+url.QueryEscape(tree)), headers)
		if err != nil {
			lastErr = err
			continue
		}
		var response struct {
			Jobs []jenkinsJobResponse `json:"jobs"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("parse jenkins view jobs failed: %w", err)
		}
		return response.Jobs, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("jenkins view endpoint is empty")
}

func jenkinsViewEndpoints(baseURL string, view jenkinsViewResponse) []string {
	endpoints := make([]string, 0, 2)
	if endpoint := strings.TrimSpace(view.URL); endpoint != "" {
		endpoints = append(endpoints, endpoint)
	}
	viewName := strings.TrimSpace(view.Name)
	if viewName != "" {
		endpoints = append(endpoints, joinURL(baseURL, "/view/"+jenkinsViewPathEscape(viewName)+"/"))
	}
	return compactProbeList(endpoints)
}

func jenkinsViewPathEscape(viewName string) string {
	parts := strings.Split(viewName, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return strings.Join(parts, "/view/")
}

func jenkinsPipelineFromJob(job jenkinsJobResponse, view string, viewURL string) domain.JenkinsPipeline {
	name := strings.TrimSpace(job.Name)
	if name == "" {
		return domain.JenkinsPipeline{}
	}
	parameters := make([]domain.JenkinsPipelineParameter, 0)
	for _, property := range job.Property {
		for _, definition := range property.ParameterDefinitions {
			parameterName := strings.TrimSpace(definition.Name)
			if parameterName == "" {
				continue
			}
			defaultValue := ""
			if definition.DefaultParameterValue != nil {
				defaultValue = jenkinsDefaultParameterValue(definition.DefaultParameterValue.Value)
			}
			parameters = append(parameters, domain.JenkinsPipelineParameter{
				Name:         parameterName,
				Type:         strings.TrimSpace(definition.Type),
				DefaultValue: defaultValue,
				Description:  strings.TrimSpace(definition.Description),
				Required:     definition.DefaultParameterValue == nil,
			})
		}
	}
	return domain.JenkinsPipeline{
		Name:       name,
		View:       strings.TrimSpace(view),
		ViewURL:    strings.TrimSpace(viewURL),
		URL:        strings.TrimSpace(job.URL),
		Parameters: parameters,
	}
}

func jenkinsDefaultParameterValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", typed), "0"), ".")
	default:
		return fmt.Sprint(typed)
	}
}

func compactProbePipelines(items []domain.JenkinsPipeline) []domain.JenkinsPipeline {
	seen := map[string]struct{}{}
	result := make([]domain.JenkinsPipeline, 0, len(items))
	for _, item := range items {
		item.Name = strings.TrimSpace(item.Name)
		item.View = strings.TrimSpace(item.View)
		item.ViewURL = strings.TrimSpace(item.ViewURL)
		item.URL = strings.TrimSpace(item.URL)
		if item.Name == "" {
			continue
		}
		key := item.View + "\x00" + item.ViewURL + "\x00" + item.Name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].View == result[j].View {
			return result[i].Name < result[j].Name
		}
		return result[i].View < result[j].View
	})
	return result
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request %s returned %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func basicAuthClient(insecureSkipTLSVerify bool) *http.Client {
	return &http.Client{
		Timeout: resourceProbeTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipTLSVerify},
		},
	}
}

func basicAuthHeaders(username string, password string) map[string]string {
	if strings.TrimSpace(username) == "" && strings.TrimSpace(password) == "" {
		return nil
	}
	token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return map[string]string{"Authorization": "Basic " + token}
}

type resourceKubeconfig struct {
	Clusters       []resourceNamedCluster `yaml:"clusters"`
	Users          []resourceNamedUser    `yaml:"users"`
	Contexts       []resourceNamedContext `yaml:"contexts"`
	CurrentContext string                 `yaml:"current-context"`
}

type resourceNamedCluster struct {
	Name    string              `yaml:"name"`
	Cluster resourceKubeCluster `yaml:"cluster"`
}

type resourceKubeCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
}

type resourceNamedUser struct {
	Name string           `yaml:"name"`
	User resourceKubeUser `yaml:"user"`
}

type resourceKubeUser struct {
	Token                 string `yaml:"token"`
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
}

type resourceNamedContext struct {
	Name    string              `yaml:"name"`
	Context resourceKubeContext `yaml:"context"`
}

type resourceKubeContext struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

func kubernetesClientFromCluster(input domain.KubernetesCluster) (*http.Client, string, error) {
	if strings.TrimSpace(input.Kubeconfig) == "" {
		if strings.TrimSpace(input.APIServer) == "" {
			return nil, "", errors.New("kubernetes api server or kubeconfig is required")
		}
		return basicAuthClient(true), input.APIServer, nil
	}
	var parsed resourceKubeconfig
	if err := yaml.Unmarshal([]byte(input.Kubeconfig), &parsed); err != nil {
		return nil, "", fmt.Errorf("parse kubeconfig failed: %w", err)
	}
	cluster, user, err := selectResourceKubeEntries(parsed, input.Context)
	if err != nil {
		return nil, "", err
	}
	server := strings.TrimSpace(input.APIServer)
	if server == "" {
		server = strings.TrimSpace(cluster.Server)
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: cluster.InsecureSkipTLSVerify}
	if !cluster.InsecureSkipTLSVerify && strings.TrimSpace(cluster.CertificateAuthorityData) != "" {
		ca, err := base64.StdEncoding.DecodeString(cluster.CertificateAuthorityData)
		if err != nil {
			return nil, "", fmt.Errorf("decode kubeconfig ca failed: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, "", errors.New("parse kubeconfig ca failed")
		}
		tlsConfig.RootCAs = pool
	}
	if strings.TrimSpace(user.ClientCertificateData) != "" && strings.TrimSpace(user.ClientKeyData) != "" {
		certData, err := base64.StdEncoding.DecodeString(user.ClientCertificateData)
		if err != nil {
			return nil, "", fmt.Errorf("decode kubeconfig client certificate failed: %w", err)
		}
		keyData, err := base64.StdEncoding.DecodeString(user.ClientKeyData)
		if err != nil {
			return nil, "", fmt.Errorf("decode kubeconfig client key failed: %w", err)
		}
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return nil, "", fmt.Errorf("parse kubeconfig client certificate failed: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{
		Timeout:   resourceProbeTimeout,
		Transport: bearerTokenTransport{token: user.Token, next: transport},
	}
	if strings.TrimSpace(server) == "" {
		return nil, "", errors.New("kubeconfig cluster server is empty")
	}
	return client, server, nil
}

func selectResourceKubeEntries(config resourceKubeconfig, requestedContext string) (resourceKubeCluster, resourceKubeUser, error) {
	contextName := strings.TrimSpace(requestedContext)
	if contextName == "" {
		contextName = strings.TrimSpace(config.CurrentContext)
	}
	if contextName == "" && len(config.Contexts) > 0 {
		contextName = config.Contexts[0].Name
	}
	var selectedContext resourceKubeContext
	for _, item := range config.Contexts {
		if item.Name == contextName {
			selectedContext = item.Context
			break
		}
	}
	if selectedContext.Cluster == "" && len(config.Clusters) > 0 {
		selectedContext.Cluster = config.Clusters[0].Name
	}
	var cluster resourceKubeCluster
	for _, item := range config.Clusters {
		if item.Name == selectedContext.Cluster {
			cluster = item.Cluster
			break
		}
	}
	if cluster.Server == "" {
		return resourceKubeCluster{}, resourceKubeUser{}, errors.New("kubeconfig cluster server is empty")
	}
	var user resourceKubeUser
	for _, item := range config.Users {
		if item.Name == selectedContext.User {
			user = item.User
			break
		}
	}
	return cluster, user, nil
}

type bearerTokenTransport struct {
	token string
	next  http.RoundTripper
}

func (t bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.TrimSpace(t.token) != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.next.RoundTrip(req)
}

func joinURL(base string, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.TrimSpace(base) == "" {
		return path
	}
	if strings.HasPrefix(path, "?") {
		return base + path
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

func normalizeProbeURL(rawURL string, scheme string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return strings.TrimRight(value, "/")
	}
	scheme = strings.TrimSpace(strings.ToLower(scheme))
	if scheme != "https" {
		scheme = "http"
	}
	return scheme + "://" + strings.TrimRight(value, "/")
}

func compactProbeList(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}
