package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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
	return ImageInfo{}, ErrNotImplemented
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
			cfg.Kubeconfig = strings.TrimSpace(cfg.Kubeconfig)
			output[normalized] = cfg
		}
	}
	return output
}

func registryConfigForEnvironment(configs map[string]RegistryConfig, environment domain.Environment) (RegistryConfig, string, error) {
	return keyedRegistryConfig(configs, environment.RegistryID, environment.Type)
}

func keyedRegistryConfig(configs map[string]RegistryConfig, id string, environmentType string) (RegistryConfig, string, error) {
	key := strings.ToLower(strings.TrimSpace(id))
	if key == "" {
		key = defaultIntegrationKey(environmentType)
	}
	cfg, ok := configs[key]
	if !ok {
		return RegistryConfig{}, key, fmt.Errorf("registry config %q is not configured", key)
	}
	return cfg, key, nil
}

func clusterConfigForEnvironment(configs map[string]ClusterConfig, environment domain.Environment) (ClusterConfig, string, error) {
	key := strings.ToLower(strings.TrimSpace(environment.ClusterID))
	if key == "" {
		key = defaultIntegrationKey(environment.Type)
	}
	cfg, ok := configs[key]
	if !ok {
		return ClusterConfig{}, key, fmt.Errorf("cluster config %q is not configured", key)
	}
	return cfg, key, nil
}

func defaultIntegrationKey(environmentType string) string {
	if strings.EqualFold(environmentType, "LOCAL") {
		return "local"
	}
	return "remote"
}

func appendURLPath(raw string, path string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String(), nil
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
