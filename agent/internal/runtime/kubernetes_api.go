package runtime

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
)

type kubeconfigFile struct {
	CurrentContext string `yaml:"current-context"`
	Clusters       []struct {
		Name    string      `yaml:"name"`
		Cluster kubeCluster `yaml:"cluster"`
	} `yaml:"clusters"`
	Contexts []struct {
		Name    string      `yaml:"name"`
		Context kubeContext `yaml:"context"`
	} `yaml:"contexts"`
	Users []struct {
		Name string   `yaml:"name"`
		User kubeUser `yaml:"user"`
	} `yaml:"users"`
}

type kubeCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
}

type kubeContext struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

type kubeUser struct {
	Token                 string `yaml:"token"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	ClientCertificate     string `yaml:"client-certificate"`
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKey             string `yaml:"client-key"`
	ClientKeyData         string `yaml:"client-key-data"`
}

type kubernetesClient struct {
	client *http.Client
	server string
}

type kubeNamespaceList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	} `json:"items"`
}

func newKubernetesClient(kubeconfigPath string, timeout time.Duration, timeoutTransport http.RoundTripper) (kubernetesClient, error) {
	path := strings.TrimSpace(kubeconfigPath)
	if path == "" {
		return kubernetesClient{}, errors.New("AGENT_KUBECONFIG is not configured")
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return kubernetesClient{}, fmt.Errorf("read kubeconfig failed: %w", err)
	}
	var parsed kubeconfigFile
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return kubernetesClient{}, fmt.Errorf("parse kubeconfig failed: %w", err)
	}
	cluster, user, err := selectedKubeEntries(parsed)
	if err != nil {
		return kubernetesClient{}, err
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: cluster.InsecureSkipTLSVerify} //nolint:gosec
	if !cluster.InsecureSkipTLSVerify {
		ca, err := kubePEMData(path, cluster.CertificateAuthorityData, cluster.CertificateAuthority)
		if err != nil {
			return kubernetesClient{}, fmt.Errorf("load kubeconfig ca failed: %w", err)
		}
		if len(ca) > 0 {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(ca) {
				return kubernetesClient{}, errors.New("parse kubeconfig ca failed")
			}
			tlsConfig.RootCAs = pool
		}
	}
	certData, err := kubePEMData(path, user.ClientCertificateData, user.ClientCertificate)
	if err != nil {
		return kubernetesClient{}, fmt.Errorf("load kubeconfig client certificate failed: %w", err)
	}
	keyData, err := kubePEMData(path, user.ClientKeyData, user.ClientKey)
	if err != nil {
		return kubernetesClient{}, fmt.Errorf("load kubeconfig client key failed: %w", err)
	}
	if len(certData) > 0 || len(keyData) > 0 {
		if len(certData) == 0 || len(keyData) == 0 {
			return kubernetesClient{}, errors.New("kubeconfig client certificate and key must be configured together")
		}
		cert, err := tls.X509KeyPair(certData, keyData)
		if err != nil {
			return kubernetesClient{}, fmt.Errorf("parse kubeconfig client certificate failed: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	transport, ok := timeoutTransport.(*http.Transport)
	if !ok || transport == nil {
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	transport = transport.Clone()
	transport.TLSClientConfig = tlsConfig

	server := strings.TrimRight(strings.TrimSpace(cluster.Server), "/")
	if server == "" {
		return kubernetesClient{}, errors.New("kubeconfig cluster server is empty")
	}
	return kubernetesClient{
		client: &http.Client{
			Timeout:   timeout,
			Transport: kubeAuthTransport{token: user.Token, username: user.Username, password: user.Password, next: transport},
		},
		server: server,
	}, nil
}

func (c kubernetesClient) listNamespaces(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/api/v1/namespaces")
	if err != nil {
		return nil, err
	}
	var payload kubeNamespaceList
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse kubernetes namespaces failed: %w", err)
	}
	items := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		name := strings.TrimSpace(item.Metadata.Name)
		if name != "" {
			items = append(items, name)
		}
	}
	return compactRuntimeItems(items), nil
}

func (c kubernetesClient) namespaceExists(ctx context.Context, namespace string) error {
	name := strings.TrimSpace(namespace)
	if name == "" {
		return errors.New("namespace is empty")
	}
	_, err := c.get(ctx, "/api/v1/namespaces/"+url.PathEscape(name))
	return err
}

func (c kubernetesClient) get(ctx context.Context, path string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.server+"/"+strings.TrimLeft(path, "/"), nil)
	if err != nil {
		return nil, fmt.Errorf("build kubernetes request failed: %w", err)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("kubernetes api request failed: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read kubernetes response failed: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("kubernetes api returned %d: %s", response.StatusCode, message)
	}
	return body, nil
}

func selectedKubeEntries(config kubeconfigFile) (kubeCluster, kubeUser, error) {
	contextName := strings.TrimSpace(config.CurrentContext)
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
	if strings.TrimSpace(cluster.Server) == "" {
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

func kubePEMData(kubeconfigPath string, encoded string, filePath string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded != "" {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, nil
	}
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(filepath.Dir(kubeconfigPath), filePath)
	}
	return os.ReadFile(filepath.Clean(filePath))
}

type kubeAuthTransport struct {
	token    string
	username string
	password string
	next     http.RoundTripper
}

func (t kubeAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if strings.TrimSpace(t.token) != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	} else if strings.TrimSpace(t.username) != "" || strings.TrimSpace(t.password) != "" {
		req.SetBasicAuth(t.username, t.password)
	}
	return t.next.RoundTrip(req)
}
