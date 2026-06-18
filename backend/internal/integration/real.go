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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"ops-release-platform/backend/internal/domain"
)

var ErrNotImplemented = errors.New("integration operation is not implemented")

func NewRealSuite(cfg Config, timeout time.Duration) (Suite, error) {
	registries := compactRegistries(cfg.Registries)
	clusters := compactClusters(cfg.Clusters)
	if len(registries) == 0 || len(clusters) == 0 {
		return Suite{}, ErrMissingRealConfig
	}
	client := &http.Client{Timeout: timeout}
	return Suite{
		Jenkins:    UnsupportedJenkinsAdapter{},
		Registry:   RealRegistryAdapter{configs: registries, client: client},
		Kubernetes: RealKubernetesAdapter{configs: clusters, timeout: timeout},
	}, nil
}

type UnsupportedJenkinsAdapter struct{}

func (UnsupportedJenkinsAdapter) TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error) {
	return BuildResult{}, ErrNotImplemented
}

func (UnsupportedJenkinsAdapter) GetBuildStatus(ctx context.Context, buildID string) (BuildStatus, error) {
	return BuildStatus{}, ErrNotImplemented
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

func (a RealKubernetesAdapter) ListWorkloads(ctx context.Context, environmentID string) ([]Workload, error) {
	return nil, ErrNotImplemented
}

func (a RealKubernetesAdapter) SetImage(ctx context.Context, environmentID string, req SetImageRequest) error {
	return ErrNotImplemented
}

func (a RealKubernetesAdapter) GetRolloutStatus(ctx context.Context, environmentID string, workload string) (RolloutStatus, error) {
	return RolloutStatus{}, ErrNotImplemented
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

type harborArtifact struct {
	Digest   string      `json:"digest"`
	PushTime string      `json:"push_time"`
	Tags     []harborTag `json:"tags"`
}

type harborTag struct {
	Name     string `json:"name"`
	PushTime string `json:"push_time"`
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
	content, err := os.ReadFile(cfg.Kubeconfig)
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
		pool, err := certPoolFromKubeconfig(filepath.Dir(cfg.Kubeconfig), cluster)
		if err != nil {
			return nil, "", err
		}
		if pool != nil {
			transport.TLSClientConfig.RootCAs = pool
		}
	}
	cert, err := clientCertFromKubeconfig(filepath.Dir(cfg.Kubeconfig), user)
	if err != nil {
		return nil, "", err
	}
	if cert != nil {
		transport.TLSClientConfig.Certificates = []tls.Certificate{*cert}
	}
	return &http.Client{Timeout: timeout, Transport: bearerTokenTransport{token: user.Token, next: transport}}, cluster.Server, nil
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
