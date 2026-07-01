package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"ops-release-platform/backend/internal/domain"
)

var ErrNotImplemented = errors.New("integration operation is not implemented")

func NewRealSuite(cfg Config, timeout time.Duration) (Suite, error) {
	registries := compactRegistries(cfg.Registries)
	clusters := compactClusters(cfg.Clusters)
	client := &http.Client{Timeout: timeout}
	suite := Suite{
		Jenkins:    RealJenkinsAdapter{client: client},
		Registry:   UnsupportedRegistryAdapter{},
		Kubernetes: UnsupportedKubernetesAdapter{},
	}
	if len(registries) > 0 {
		suite.Registry = RealRegistryAdapter{configs: registries, client: client}
	}
	if len(clusters) > 0 {
		suite.Kubernetes = RealKubernetesAdapter{configs: clusters, timeout: timeout}
	}
	return suite, nil
}

func NewRealJenkinsAdapter(timeout time.Duration) JenkinsAdapter {
	return RealJenkinsAdapter{client: &http.Client{Timeout: timeout}}
}

type RealJenkinsAdapter struct {
	client *http.Client
}

type UnsupportedRegistryAdapter struct{}

func (UnsupportedRegistryAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return IntegrationCheck{}, err
	}
	return IntegrationCheck{}, ErrMissingRealConfig
}

func (UnsupportedRegistryAdapter) ListImageTags(ctx context.Context, environment domain.Environment, repository string) ([]ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ErrMissingRealConfig
}

func (UnsupportedRegistryAdapter) GetImage(ctx context.Context, image string, tag string) (ImageInfo, error) {
	if err := ctx.Err(); err != nil {
		return ImageInfo{}, err
	}
	return ImageInfo{}, ErrMissingRealConfig
}

func (UnsupportedRegistryAdapter) SyncImage(ctx context.Context, req SyncImageRequest) (SyncImageResult, error) {
	if err := ctx.Err(); err != nil {
		return SyncImageResult{}, err
	}
	return SyncImageResult{}, ErrMissingRealConfig
}

type UnsupportedKubernetesAdapter struct{}

func (UnsupportedKubernetesAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	if err := ctx.Err(); err != nil {
		return IntegrationCheck{}, err
	}
	return IntegrationCheck{}, ErrMissingRealConfig
}

func (UnsupportedKubernetesAdapter) ListWorkloads(ctx context.Context, environment domain.Environment) ([]Workload, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ErrMissingRealConfig
}

func (UnsupportedKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req SetImageRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrMissingRealConfig
}

func (UnsupportedKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (RolloutStatus, error) {
	if err := ctx.Err(); err != nil {
		return RolloutStatus{}, err
	}
	return RolloutStatus{}, ErrMissingRealConfig
}

func (a RealJenkinsAdapter) TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error) {
	client := a.clientWithCookieJar(req.InsecureSkipTLSVerify)
	jobURL, err := jenkinsJobURL(req.JenkinsURL, req.JobName, req.JobURL)
	if err != nil {
		return BuildResult{}, err
	}
	params := compactBuildParameters(req.Parameters)
	endpoint := strings.TrimRight(jobURL, "/") + "/build"
	endpointType := "build"
	var body io.Reader
	if req.Parameterized || len(params) > 0 {
		values := url.Values{}
		for key, value := range params {
			values.Set(key, value)
		}
		endpoint = strings.TrimRight(jobURL, "/") + "/buildWithParameters"
		endpointType = "buildWithParameters"
		body = strings.NewReader(values.Encode())
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return BuildResult{}, err
	}
	if req.Parameterized || len(params) > 0 {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	applyJenkinsAuth(httpReq, req.Username, req.Token)
	crumbApplied := false
	if crumb, ok := a.jenkinsCrumb(ctx, client, req, jobURL); ok {
		httpReq.Header.Set(crumb.Field, crumb.Value)
		for _, cookie := range crumb.Cookies {
			httpReq.AddCookie(cookie)
		}
		crumbApplied = true
	}
	httpReq.Header.Set("Referer", jobURL)
	resp, err := client.Do(httpReq)
	if err != nil {
		return BuildResult{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return BuildResult{}, jenkinsTriggerStatusError(resp.StatusCode, endpointType, endpoint, firstNonEmpty(strings.TrimSpace(req.JobName), jobURL), crumbApplied, sortedMapKeys(params), respBody)
	}
	location := strings.TrimSpace(resp.Header.Get("Location"))
	result := BuildResult{Status: "QUEUED", URL: location}
	if location == "" {
		result.BuildID = "queued"
		result.URL = jobURL
		return result, nil
	}
	if queueID := jenkinsQueueID(location); queueID != "" {
		result.BuildID = "queue:" + queueID
	}
	if executable := a.pollJenkinsQueue(ctx, client, req.Username, req.Token, location); executable.Number != "" {
		result.BuildID = executable.Number
		result.Status = "BUILDING"
		result.URL = executable.URL
	}
	if result.BuildID == "" {
		result.BuildID = location
	}
	return result, nil
}

func (a RealJenkinsAdapter) GetJobParameters(ctx context.Context, req JobParametersRequest) ([]domain.JenkinsPipelineParameter, error) {
	client := a.clientWithCookieJar(req.InsecureSkipTLSVerify)
	jobURL, err := jenkinsJobURL(req.JenkinsURL, req.JobName, req.JobURL)
	if err != nil {
		return nil, err
	}
	tree := "property[parameterDefinitions[name,type,description,defaultParameterValue[value]]]"
	endpoint := strings.TrimRight(jobURL, "/") + "/api/json?tree=" + url.QueryEscape(tree)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	applyJenkinsAuth(httpReq, req.Username, req.Token)
	if crumb, ok := a.jenkinsCrumb(ctx, client, BuildRequest{
		JenkinsURL:            req.JenkinsURL,
		Username:              req.Username,
		Token:                 req.Token,
		InsecureSkipTLSVerify: req.InsecureSkipTLSVerify,
	}, jobURL); ok {
		httpReq.Header.Set(crumb.Field, crumb.Value)
		for _, cookie := range crumb.Cookies {
			httpReq.AddCookie(cookie)
		}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jenkins job parameters returned %d: %s", resp.StatusCode, compactHTTPErrorBody(body))
	}
	var payload struct {
		Property []jenkinsJobPropertyResponse `json:"property"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return jenkinsParametersFromProperties(payload.Property), nil
}

func (a RealJenkinsAdapter) GetBuildStatus(ctx context.Context, req BuildStatusRequest) (BuildStatus, error) {
	client := a.clientFor(req.InsecureSkipTLSVerify)
	if queueURL := jenkinsQueueURL(req); queueURL != "" {
		status, ready, err := a.jenkinsQueueStatus(ctx, client, req.Username, req.Token, queueURL, req.LogLineLimit)
		if err != nil {
			return BuildStatus{}, err
		}
		if !ready {
			return status, nil
		}
		req.BuildID = status.BuildID
		req.BuildURL = status.URL
	}
	buildURL, err := jenkinsBuildURL(req)
	if err != nil {
		return BuildStatus{}, err
	}
	apiURL := strings.TrimRight(buildURL, "/") + "/api/json"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return BuildStatus{}, err
	}
	httpReq.Header.Set("Accept", "application/json")
	applyJenkinsAuth(httpReq, req.Username, req.Token)
	resp, err := client.Do(httpReq)
	if err != nil {
		return BuildStatus{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return BuildStatus{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return BuildStatus{}, fmt.Errorf("jenkins build status returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		ID        string `json:"id"`
		Number    int    `json:"number"`
		Building  bool   `json:"building"`
		Result    string `json:"result"`
		Timestamp int64  `json:"timestamp"`
		Duration  int64  `json:"duration"`
		URL       string `json:"url"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return BuildStatus{}, err
	}
	status := strings.TrimSpace(payload.Result)
	if status == "" && payload.Building {
		status = "BUILDING"
	}
	if status == "" {
		status = "QUEUED"
	}
	buildID := strings.TrimSpace(payload.ID)
	if buildID == "" && payload.Number > 0 {
		buildID = fmt.Sprint(payload.Number)
	}
	if buildID == "" {
		buildID = strings.TrimSpace(req.BuildID)
	}
	finalURL := firstNonEmpty(strings.TrimSpace(payload.URL), buildURL)
	logURL := strings.TrimRight(finalURL, "/") + "/console"
	logs := a.jenkinsConsoleLogs(ctx, client, req.Username, req.Token, finalURL, req.LogLineLimit)
	return BuildStatus{
		BuildID:    buildID,
		Status:     status,
		StartedAt:  formatJenkinsMillis(payload.Timestamp),
		FinishedAt: formatJenkinsFinishedAt(payload.Timestamp, payload.Duration, payload.Building),
		LogURL:     logURL,
		URL:        finalURL,
		Logs:       logs,
	}, nil
}

func compactHTTPErrorBody(body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty response"
	}
	message = strings.ReplaceAll(message, "\r\n", "\n")
	lines := strings.Split(message, "\n")
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		compact = append(compact, trimmed)
		if len(compact) >= 3 {
			break
		}
	}
	message = strings.Join(compact, " ")
	if len(message) > 500 {
		return message[:500] + "..."
	}
	return message
}

type jenkinsCrumbResponse struct {
	Field   string `json:"crumbRequestField"`
	Value   string `json:"crumb"`
	Cookies []*http.Cookie
}

type jenkinsExecutable struct {
	Number string
	URL    string
}

func (a RealJenkinsAdapter) clientFor(insecure bool) *http.Client {
	if !insecure && a.client != nil {
		return a.client
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}
}

func (a RealJenkinsAdapter) clientWithCookieJar(insecure bool) *http.Client {
	base := a.clientFor(insecure)
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout:   base.Timeout,
		Transport: base.Transport,
		Jar:       jar,
	}
}

func (a RealJenkinsAdapter) jenkinsCrumb(ctx context.Context, client *http.Client, req BuildRequest, jobURL string) (jenkinsCrumbResponse, bool) {
	endpoints := []string{}
	if endpoint, err := jenkinsRootEndpoint(jobURL, "/crumbIssuer/api/json"); err == nil {
		endpoints = append(endpoints, endpoint)
	}
	if endpoint, err := appendURLPath(req.JenkinsURL, "/crumbIssuer/api/json"); err == nil {
		endpoints = append(endpoints, endpoint)
	}
	for _, endpoint := range compactStringList(endpoints) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		httpReq.Header.Set("Accept", "application/json")
		applyJenkinsAuth(httpReq, req.Username, req.Token)
		resp, err := client.Do(httpReq)
		if err != nil {
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
			continue
		}
		var crumb jenkinsCrumbResponse
		err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&crumb)
		crumb.Cookies = resp.Cookies()
		_ = resp.Body.Close()
		if err != nil {
			continue
		}
		if strings.TrimSpace(crumb.Field) != "" && strings.TrimSpace(crumb.Value) != "" {
			return crumb, true
		}
	}
	return jenkinsCrumbResponse{}, false
}

func (a RealJenkinsAdapter) pollJenkinsQueue(ctx context.Context, client *http.Client, username string, token string, queueURL string) jenkinsExecutable {
	endpoint := strings.TrimRight(queueURL, "/") + "/api/json"
	for attempt := 0; attempt < 8; attempt++ {
		select {
		case <-ctx.Done():
			return jenkinsExecutable{}
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return jenkinsExecutable{}
		}
		httpReq.Header.Set("Accept", "application/json")
		applyJenkinsAuth(httpReq, username, token)
		resp, err := client.Do(httpReq)
		if err != nil {
			continue
		}
		var payload struct {
			Executable struct {
				Number int    `json:"number"`
				URL    string `json:"url"`
			} `json:"executable"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || decodeErr != nil {
			continue
		}
		if payload.Executable.Number > 0 && strings.TrimSpace(payload.Executable.URL) != "" {
			return jenkinsExecutable{Number: fmt.Sprint(payload.Executable.Number), URL: strings.TrimSpace(payload.Executable.URL)}
		}
	}
	return jenkinsExecutable{}
}

func (a RealJenkinsAdapter) jenkinsQueueStatus(ctx context.Context, client *http.Client, username string, token string, queueURL string, limit int) (BuildStatus, bool, error) {
	endpoint := strings.TrimRight(queueURL, "/") + "/api/json"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return BuildStatus{}, false, err
	}
	httpReq.Header.Set("Accept", "application/json")
	applyJenkinsAuth(httpReq, username, token)
	resp, err := client.Do(httpReq)
	if err != nil {
		return BuildStatus{}, false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return BuildStatus{}, false, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return BuildStatus{}, false, fmt.Errorf("jenkins queue status returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		ID         int    `json:"id"`
		Why        string `json:"why"`
		Executable struct {
			Number int    `json:"number"`
			URL    string `json:"url"`
		} `json:"executable"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return BuildStatus{}, false, err
	}
	if payload.Executable.Number > 0 && strings.TrimSpace(payload.Executable.URL) != "" {
		return BuildStatus{
			BuildID: fmt.Sprint(payload.Executable.Number),
			Status:  "BUILDING",
			URL:     strings.TrimSpace(payload.Executable.URL),
			LogURL:  strings.TrimRight(strings.TrimSpace(payload.Executable.URL), "/") + "/console",
			Logs:    a.jenkinsConsoleLogs(ctx, client, username, token, payload.Executable.URL, limit),
		}, true, nil
	}
	buildID := "queued"
	if payload.ID > 0 {
		buildID = "queue:" + fmt.Sprint(payload.ID)
	}
	message := strings.TrimSpace(payload.Why)
	if message == "" {
		message = "Jenkins 构建仍在队列中"
	}
	return BuildStatus{
		BuildID: buildID,
		Status:  "QUEUED",
		URL:     strings.TrimRight(queueURL, "/"),
		Logs:    []string{message},
	}, false, nil
}

func (a RealJenkinsAdapter) jenkinsConsoleLogs(ctx context.Context, client *http.Client, username string, token string, buildURL string, limit int) []string {
	endpoint := strings.TrimRight(buildURL, "/") + "/consoleText"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return []string{}
	}
	applyJenkinsAuth(httpReq, username, token)
	resp, err := client.Do(httpReq)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []string{}
	}
	return tailLines(string(body), limit)
}

func applyJenkinsAuth(req *http.Request, username string, token string) {
	if strings.TrimSpace(username) != "" || strings.TrimSpace(token) != "" {
		req.SetBasicAuth(username, token)
	}
}

func compactBuildParameters(values map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = value
	}
	return result
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func compactStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func isJenkinsBranchParameter(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "branch", "branch_name", "branchname", "git_branch", "gitbranch", "git_branch_name", "gitbranchname", "ref", "git_ref":
		return true
	default:
		return false
	}
}

func jenkinsTriggerStatusError(status int, endpointType string, endpoint string, job string, crumbApplied bool, parameterNames []string, body []byte) error {
	response := compactHTTPErrorBody(body)
	endpointURL := sanitizeJenkinsErrorURL(endpoint)
	params := strings.Join(parameterNames, ",")
	if params == "" {
		params = "none"
	}
	if status == http.StatusForbidden {
		return fmt.Errorf(
			"Jenkins 返回 403：请检查 Jenkins 用户/API Token 是否有该 Pipeline 的 Build 权限、是否允许远程触发构建，或 Jenkins CSRF Crumb 配置；endpoint=%s url=%s job=%s crumb=%t params=%s response=%s",
			endpointType,
			endpointURL,
			job,
			crumbApplied,
			params,
			response,
		)
	}
	return fmt.Errorf(
		"jenkins trigger returned status %d: endpoint=%s url=%s job=%s crumb=%t params=%s response=%s",
		status,
		endpointType,
		endpointURL,
		job,
		crumbApplied,
		params,
		response,
	)
}

func jenkinsRootEndpoint(raw string, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("jenkins url is invalid")
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func sanitizeJenkinsErrorURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimSpace(raw)
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func jenkinsJobURL(baseURL string, jobName string, jobURL string) (string, error) {
	if trimmed := strings.TrimSpace(jobURL); trimmed != "" {
		return strings.TrimRight(trimmed, "/"), nil
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	job := strings.TrimSpace(jobName)
	if base == "" || job == "" {
		return "", fmt.Errorf("jenkins url and job name are required")
	}
	parts := strings.Split(job, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			escaped = append(escaped, "job", url.PathEscape(trimmed))
		}
	}
	if len(escaped) == 0 {
		return "", fmt.Errorf("jenkins job name is required")
	}
	return base + "/" + strings.Join(escaped, "/"), nil
}

func jenkinsBuildURL(req BuildStatusRequest) (string, error) {
	if trimmed := strings.TrimSpace(req.BuildURL); trimmed != "" && !strings.HasPrefix(trimmed, "queue:") {
		if strings.Contains(trimmed, "/queue/item/") {
			return "", fmt.Errorf("jenkins executable build is not ready")
		}
		return strings.TrimRight(trimmed, "/"), nil
	}
	buildID := strings.TrimSpace(req.BuildID)
	if strings.HasPrefix(buildID, "queue:") || buildID == "" || buildID == "queued" {
		return "", fmt.Errorf("jenkins executable build is not ready")
	}
	jobURL, err := jenkinsJobURL(req.JenkinsURL, req.JobName, req.JobURL)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(jobURL, "/") + "/" + url.PathEscape(buildID), nil
}

func jenkinsQueueURL(req BuildStatusRequest) string {
	if trimmed := strings.TrimRight(strings.TrimSpace(req.BuildURL), "/"); strings.Contains(trimmed, "/queue/item/") {
		return trimmed
	}
	if buildID := strings.TrimSpace(req.BuildID); strings.HasPrefix(buildID, "queue:") {
		queueID := strings.TrimSpace(strings.TrimPrefix(buildID, "queue:"))
		if queueID == "" {
			return ""
		}
		base := strings.TrimRight(strings.TrimSpace(req.JenkinsURL), "/")
		if base == "" {
			if jobURL := strings.TrimSpace(req.JobURL); jobURL != "" {
				parsed, err := url.Parse(jobURL)
				if err == nil && parsed.Scheme != "" && parsed.Host != "" {
					base = parsed.Scheme + "://" + parsed.Host
				}
			}
		}
		if base == "" {
			return ""
		}
		return base + "/queue/item/" + url.PathEscape(queueID)
	}
	return ""
}

func jenkinsQueueID(location string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(location), "/")
	index := strings.LastIndex(trimmed, "/")
	if index < 0 || index == len(trimmed)-1 {
		return ""
	}
	value := trimmed[index+1:]
	if _, err := strconv.Atoi(value); err != nil {
		return ""
	}
	return value
}

func formatJenkinsMillis(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.UnixMilli(value).Format(time.RFC3339)
}

func formatJenkinsFinishedAt(start int64, duration int64, building bool) string {
	if start <= 0 || duration <= 0 || building {
		return ""
	}
	return time.UnixMilli(start + duration).Format(time.RFC3339)
}

func tailLines(text string, limit int) []string {
	if limit <= 0 {
		limit = 200
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines
}

type RealRegistryAdapter struct {
	configs map[string]RegistryConfig
	client  *http.Client
}

func (a RealRegistryAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	cfg, key, err := registryConfigForEnvironment(a.configs, environment)
	if err != nil {
		return IntegrationCheck{}, err
	}
	endpoint, err := appendURLPath(cfg.URL, "/api/v2.0/systeminfo")
	if err != nil {
		return IntegrationCheck{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return IntegrationCheck{}, err
	}
	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return IntegrationCheck{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return IntegrationCheck{}, fmt.Errorf("harbor %s returned status %d", key, resp.StatusCode)
	}
	return IntegrationCheck{
		Component: "harbor",
		Status:    "HEALTHY",
		Message:   "registry " + key + " is reachable",
		CheckedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (a RealRegistryAdapter) GetImage(ctx context.Context, image string, tag string) (ImageInfo, error) {
	environment := domain.Environment{RegistryID: "local", Type: "LOCAL"}
	tags, err := a.ListImageTags(ctx, environment, image)
	if err != nil {
		return ImageInfo{}, err
	}
	for _, item := range tags {
		if item.Tag == tag {
			item.Exists = true
			return item, nil
		}
	}
	return ImageInfo{Image: image, Tag: tag, Exists: false}, nil
}

func (a RealRegistryAdapter) ListImageTags(ctx context.Context, environment domain.Environment, repository string) ([]ImageInfo, error) {
	cfg, _, err := registryConfigForEnvironment(a.configs, environment)
	if err != nil {
		return nil, err
	}
	project, repoName, err := splitHarborRepository(repository)
	if err != nil {
		return nil, err
	}
	endpoint, err := harborArtifactsURL(cfg.URL, project, repoName)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("harbor returned status %d", resp.StatusCode)
	}
	var artifacts []harborArtifact
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&artifacts); err != nil {
		return nil, err
	}
	items := make([]ImageInfo, 0)
	for _, artifact := range artifacts {
		for _, tag := range artifact.Tags {
			if strings.TrimSpace(tag.Name) == "" {
				continue
			}
			items = append(items, ImageInfo{
				Image:     repository,
				Tag:       tag.Name,
				Digest:    artifact.Digest,
				Exists:    true,
				UpdatedAt: firstNonEmpty(tag.PushTime, artifact.PushTime),
			})
		}
	}
	return items, nil
}

func (a RealRegistryAdapter) SyncImage(ctx context.Context, req SyncImageRequest) (SyncImageResult, error) {
	return SyncImageResult{}, ErrNotImplemented
}

type RealKubernetesAdapter struct {
	configs map[string]ClusterConfig
	timeout time.Duration
}

func ListWorkloadsWithKubeconfig(ctx context.Context, kubeconfig string, apiServer string, namespaces []string, timeout time.Duration) ([]Workload, error) {
	adapter := RealKubernetesAdapter{
		configs: map[string]ClusterConfig{
			"cluster": {Kubeconfig: kubeconfig},
		},
		timeout: timeout,
	}
	return adapter.ListWorkloads(ctx, domain.Environment{
		ClusterID:         "cluster",
		ClusterAPIServer:  apiServer,
		ClusterKubeconfig: kubeconfig,
		Namespace:         strings.Join(namespaces, ","),
	})
}

func (a RealKubernetesAdapter) CheckConnection(ctx context.Context, environment domain.Environment) (IntegrationCheck, error) {
	cfg, key, err := clusterConfigForEnvironment(a.configs, environment)
	if err != nil {
		return IntegrationCheck{}, err
	}
	client, server, err := kubernetesHTTPClient(cfg, a.timeout)
	if err != nil {
		return IntegrationCheck{}, err
	}
	if configuredServer := strings.TrimSpace(environment.ClusterAPIServer); configuredServer != "" {
		server = configuredServer
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(server, "/")+"/readyz", nil)
	if err != nil {
		return IntegrationCheck{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return IntegrationCheck{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return IntegrationCheck{}, fmt.Errorf("kubernetes %s returned status %d", key, resp.StatusCode)
	}
	return IntegrationCheck{
		Component: "kubernetes",
		Status:    "HEALTHY",
		Message:   "cluster " + key + " is reachable",
		CheckedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (a RealKubernetesAdapter) ListWorkloads(ctx context.Context, environment domain.Environment) ([]Workload, error) {
	cfg, _, err := clusterConfigForEnvironment(a.configs, environment)
	if err != nil {
		return nil, err
	}
	client, server, err := kubernetesHTTPClient(cfg, a.timeout)
	if err != nil {
		return nil, err
	}
	if configuredServer := strings.TrimSpace(environment.ClusterAPIServer); configuredServer != "" {
		server = configuredServer
	}
	namespaces := workloadNamespaces(environment)
	if len(namespaces) == 0 {
		return nil, errors.New("kubernetes namespace is required")
	}
	items := make([]Workload, 0)
	for _, namespace := range namespaces {
		deployments, err := a.listWorkloadsByType(ctx, client, server, namespace, "Deployment", "/apis/apps/v1/namespaces/"+url.PathEscape(namespace)+"/deployments")
		if err != nil {
			return nil, err
		}
		items = append(items, deployments...)
		statefulSets, err := a.listWorkloadsByType(ctx, client, server, namespace, "StatefulSet", "/apis/apps/v1/namespaces/"+url.PathEscape(namespace)+"/statefulsets")
		if err != nil {
			return nil, err
		}
		items = append(items, statefulSets...)
		daemonSets, err := a.listWorkloadsByType(ctx, client, server, namespace, "DaemonSet", "/apis/apps/v1/namespaces/"+url.PathEscape(namespace)+"/daemonsets")
		if err != nil {
			return nil, err
		}
		items = append(items, daemonSets...)
	}
	return items, nil
}

func (a RealKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req SetImageRequest) error {
	return ErrNotImplemented
}

func (a RealKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (RolloutStatus, error) {
	return RolloutStatus{}, ErrNotImplemented
}

func (a RealKubernetesAdapter) listWorkloadsByType(ctx context.Context, client *http.Client, server string, namespace string, workloadType string, path string) ([]Workload, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(server, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("kubernetes %s %s returned status %d", namespace, strings.ToLower(workloadType), resp.StatusCode)
	}
	var payload kubeWorkloadList
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8*1024*1024)).Decode(&payload); err != nil {
		return nil, err
	}
	items := make([]Workload, 0, len(payload.Items))
	for _, item := range payload.Items {
		containers := make([]WorkloadContainer, 0, len(item.Spec.Template.Spec.InitContainers)+len(item.Spec.Template.Spec.Containers))
		for _, container := range item.Spec.Template.Spec.InitContainers {
			if strings.TrimSpace(container.Name) == "" || strings.TrimSpace(container.Image) == "" {
				continue
			}
			containers = append(containers, WorkloadContainer{Name: container.Name, Type: "INIT", Image: container.Image})
		}
		for _, container := range item.Spec.Template.Spec.Containers {
			if strings.TrimSpace(container.Name) == "" || strings.TrimSpace(container.Image) == "" {
				continue
			}
			containers = append(containers, WorkloadContainer{Name: container.Name, Type: "APP", Image: container.Image})
		}
		items = append(items, Workload{
			Namespace:     firstNonEmpty(item.Metadata.Namespace, namespace),
			Name:          item.Metadata.Name,
			Type:          workloadType,
			Replicas:      item.Spec.Replicas,
			ReadyReplicas: item.Status.ReadyReplicas,
			Containers:    containers,
		})
	}
	return items, nil
}

func workloadNamespaces(environment domain.Environment) []string {
	values := make([]string, 0)
	for _, binding := range environment.Bindings {
		if binding.ResourceType != "K8S" || binding.BindingRole != "BUILD_SOURCE" || binding.ScopeValue == "" {
			continue
		}
		values = appendUniqueTrimmed(values, binding.ScopeValue)
	}
	values = appendUniqueTrimmed(values, environment.Namespace)
	return values
}

func appendUniqueTrimmed(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}

func compactRegistries(input map[string]RegistryConfig) map[string]RegistryConfig {
	output := map[string]RegistryConfig{}
	for key, cfg := range input {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized != "" && strings.TrimSpace(cfg.URL) != "" {
			cfg.URL = strings.TrimSpace(cfg.URL)
			output[normalized] = cfg
		}
	}
	return output
}

func compactClusters(input map[string]ClusterConfig) map[string]ClusterConfig {
	output := map[string]ClusterConfig{}
	for key, cfg := range input {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized != "" && strings.TrimSpace(cfg.Kubeconfig) != "" {
			cfg.Kubeconfig = resolveExistingPath(strings.TrimSpace(cfg.Kubeconfig))
			output[normalized] = cfg
		}
	}
	return output
}

func resolveExistingPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return path
	}
	for {
		candidate := filepath.Join(currentDir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}
	return path
}

func registryConfigForEnvironment(configs map[string]RegistryConfig, environment domain.Environment) (RegistryConfig, string, error) {
	resourceID := strings.TrimSpace(environment.RegistryID)
	if resourceID == "" {
		return RegistryConfig{}, "", errors.New("registry resource is not selected")
	}
	key := strings.ToLower(firstNonEmpty(environment.RegistryCredentialRef, resourceID))
	cfg, ok := configs[key]
	if !ok && strings.TrimSpace(environment.RegistryCredentialRef) != "" {
		key = strings.ToLower(resourceID)
		cfg, ok = configs[key]
	}
	if strings.TrimSpace(environment.RegistryURL) == "" {
		return RegistryConfig{}, resourceID, fmt.Errorf("registry resource %q has no url", resourceID)
	}
	cfg.URL = environment.RegistryURL
	return cfg, resourceID, nil
}

func clusterConfigForEnvironment(configs map[string]ClusterConfig, environment domain.Environment) (ClusterConfig, string, error) {
	resourceID := strings.TrimSpace(environment.ClusterID)
	if resourceID == "" {
		return ClusterConfig{}, "", errors.New("kubernetes cluster resource is not selected")
	}
	if strings.TrimSpace(environment.ClusterKubeconfig) != "" {
		return ClusterConfig{Kubeconfig: environment.ClusterKubeconfig}, resourceID, nil
	}
	key := strings.ToLower(firstNonEmpty(environment.ClusterCredentialRef, resourceID))
	cfg, ok := configs[key]
	if !ok && strings.TrimSpace(environment.ClusterCredentialRef) != "" {
		key = strings.ToLower(resourceID)
		cfg, ok = configs[key]
	}
	if !ok {
		return ClusterConfig{}, resourceID, fmt.Errorf("cluster credential %q is not configured", key)
	}
	return cfg, resourceID, nil
}

func appendURLPath(raw string, path string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String(), nil
}

func harborArtifactsURL(raw string, project string, repository string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	escapedRepo := url.PathEscape(repository)
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v2.0/projects/" + url.PathEscape(project) + "/repositories/" + escapedRepo + "/artifacts"
	query := parsed.Query()
	query.Set("with_tag", "true")
	query.Set("page_size", "50")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func splitHarborRepository(repository string) (string, string, error) {
	value := strings.TrimSpace(repository)
	if value == "" {
		return "", "", errors.New("image repository is empty")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = strings.TrimPrefix(parsed.Path, "/")
	} else {
		parts := strings.Split(value, "/")
		if len(parts) >= 3 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost") {
			value = strings.Join(parts[1:], "/")
		}
	}
	parts := strings.SplitN(strings.Trim(value, "/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("image repository %q must include harbor project and repository", repository)
	}
	return parts[0], parts[1], nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func jenkinsParametersFromProperties(properties []jenkinsJobPropertyResponse) []domain.JenkinsPipelineParameter {
	parameters := make([]domain.JenkinsPipelineParameter, 0)
	for _, property := range properties {
		for _, definition := range property.ParameterDefinitions {
			name := strings.TrimSpace(definition.Name)
			if name == "" {
				continue
			}
			defaultValue := ""
			if definition.DefaultParameterValue != nil {
				defaultValue = jenkinsDefaultParameterValue(definition.DefaultParameterValue.Value)
			}
			parameters = append(parameters, domain.JenkinsPipelineParameter{
				Name:         name,
				Type:         strings.TrimSpace(definition.Type),
				DefaultValue: defaultValue,
				Description:  strings.TrimSpace(definition.Description),
				Required:     definition.DefaultParameterValue == nil,
			})
		}
	}
	return parameters
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

type harborArtifact struct {
	Digest   string      `json:"digest"`
	PushTime string      `json:"push_time"`
	Tags     []harborTag `json:"tags"`
}

type harborTag struct {
	Name     string `json:"name"`
	PushTime string `json:"push_time"`
}

type kubeWorkloadList struct {
	Items []kubeWorkload `json:"items"`
}

type kubeWorkload struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Replicas int `json:"replicas"`
		Template struct {
			Spec struct {
				InitContainers []kubeContainer `json:"initContainers"`
				Containers     []kubeContainer `json:"containers"`
			} `json:"spec"`
		} `json:"template"`
	} `json:"spec"`
	Status struct {
		ReadyReplicas int `json:"readyReplicas"`
	} `json:"status"`
}

type kubeContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type kubeconfigFile struct {
	Clusters       []namedCluster `yaml:"clusters"`
	Users          []namedUser    `yaml:"users"`
	Contexts       []namedContext `yaml:"contexts"`
	CurrentContext string         `yaml:"current-context"`
}

type namedCluster struct {
	Name    string      `yaml:"name"`
	Cluster kubeCluster `yaml:"cluster"`
}

type kubeCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
}

type namedUser struct {
	Name string   `yaml:"name"`
	User kubeUser `yaml:"user"`
}

type kubeUser struct {
	Token                 string `yaml:"token"`
	ClientCertificate     string `yaml:"client-certificate"`
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKey             string `yaml:"client-key"`
	ClientKeyData         string `yaml:"client-key-data"`
}

type namedContext struct {
	Name    string      `yaml:"name"`
	Context kubeContext `yaml:"context"`
}

type kubeContext struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

func kubernetesHTTPClient(cfg ClusterConfig, timeout time.Duration) (*http.Client, string, error) {
	content, baseDir, err := kubeconfigContent(cfg.Kubeconfig)
	if err != nil {
		return nil, "", err
	}
	var parsed kubeconfigFile
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return nil, "", err
	}
	cluster, user, err := selectedKubeEntries(parsed)
	if err != nil {
		return nil, "", err
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cluster.InsecureSkipTLSVerify}}
	if !cluster.InsecureSkipTLSVerify {
		pool, err := certPoolFromKubeconfig(baseDir, cluster)
		if err != nil {
			return nil, "", err
		}
		if pool != nil {
			transport.TLSClientConfig.RootCAs = pool
		}
	}
	cert, err := clientCertFromKubeconfig(baseDir, user)
	if err != nil {
		return nil, "", err
	}
	if cert != nil {
		transport.TLSClientConfig.Certificates = []tls.Certificate{*cert}
	}
	return &http.Client{Timeout: timeout, Transport: bearerTokenTransport{token: user.Token, next: transport}}, cluster.Server, nil
}

func kubeconfigContent(value string) ([]byte, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, "", errors.New("kubeconfig is required")
	}
	if content, err := os.ReadFile(value); err == nil {
		return content, filepath.Dir(value), nil
	}
	return []byte(value), "", nil
}

func selectedKubeEntries(config kubeconfigFile) (kubeCluster, kubeUser, error) {
	contextName := config.CurrentContext
	if contextName == "" && len(config.Contexts) > 0 {
		contextName = config.Contexts[0].Name
	}
	var selectedContext kubeContext
	for _, item := range config.Contexts {
		if item.Name == contextName {
			selectedContext = item.Context
			break
		}
	}
	if selectedContext.Cluster == "" && len(config.Clusters) > 0 {
		selectedContext.Cluster = config.Clusters[0].Name
	}
	var cluster kubeCluster
	for _, item := range config.Clusters {
		if item.Name == selectedContext.Cluster {
			cluster = item.Cluster
			break
		}
	}
	if cluster.Server == "" {
		return kubeCluster{}, kubeUser{}, errors.New("kubeconfig cluster server is empty")
	}
	var user kubeUser
	for _, item := range config.Users {
		if item.Name == selectedContext.User {
			user = item.User
			break
		}
	}
	return cluster, user, nil
}

func certPoolFromKubeconfig(baseDir string, cluster kubeCluster) (*x509.CertPool, error) {
	caData, err := decodeOrRead(baseDir, cluster.CertificateAuthorityData, cluster.CertificateAuthority)
	if err != nil || len(caData) == 0 {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, errors.New("failed to parse kubeconfig certificate authority")
	}
	return pool, nil
}

func clientCertFromKubeconfig(baseDir string, user kubeUser) (*tls.Certificate, error) {
	certData, err := decodeOrRead(baseDir, user.ClientCertificateData, user.ClientCertificate)
	if err != nil || len(certData) == 0 {
		return nil, err
	}
	keyData, err := decodeOrRead(baseDir, user.ClientKeyData, user.ClientKey)
	if err != nil || len(keyData) == 0 {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func decodeOrRead(baseDir string, inline string, path string) ([]byte, error) {
	if inline != "" {
		return base64.StdEncoding.DecodeString(inline)
	}
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return os.ReadFile(path)
}

type bearerTokenTransport struct {
	token string
	next  http.RoundTripper
}

func (t bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.next.RoundTrip(req)
}
