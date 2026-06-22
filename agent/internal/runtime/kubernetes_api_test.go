package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKubernetesClientUsesAPIFromKubeconfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected bearer token auth, got %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/namespaces":
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"xjzt"}},{"metadata":{"name":"default"}}]}`))
		case "/api/v1/namespaces/xjzt":
			_, _ = w.Write([]byte(`{"metadata":{"name":"xjzt"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	kubeconfig := strings.Join([]string{
		"apiVersion: v1",
		"current-context: test",
		"clusters:",
		"- name: test-cluster",
		"  cluster:",
		"    server: " + server.URL,
		"    insecure-skip-tls-verify: true",
		"contexts:",
		"- name: test",
		"  context:",
		"    cluster: test-cluster",
		"    user: test-user",
		"users:",
		"- name: test-user",
		"  user:",
		"    token: test-token",
	}, "\n")
	path := filepath.Join(t.TempDir(), "kubeconfig.yaml")
	if err := os.WriteFile(path, []byte(kubeconfig), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	client, err := newKubernetesClient(path, time.Second, http.DefaultTransport)
	if err != nil {
		t.Fatalf("new kubernetes client: %v", err)
	}
	namespaces, err := client.listNamespaces(context.Background())
	if err != nil {
		t.Fatalf("list namespaces: %v", err)
	}
	if strings.Join(namespaces, ",") != "default,xjzt" {
		t.Fatalf("unexpected namespaces: %v", namespaces)
	}
	if err := client.namespaceExists(context.Background(), "xjzt"); err != nil {
		t.Fatalf("namespace exists: %v", err)
	}
}

func TestKubernetesClientLoadsRelativeCertificateAuthorityFile(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caPath, []byte("invalid-ca"), 0o600); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	kubeconfig := strings.Join([]string{
		"apiVersion: v1",
		"clusters:",
		"- name: test-cluster",
		"  cluster:",
		"    server: https://127.0.0.1:6443",
		"    certificate-authority: ca.crt",
		"contexts:",
		"- name: test",
		"  context:",
		"    cluster: test-cluster",
	}, "\n")
	path := filepath.Join(dir, "kubeconfig.yaml")
	if err := os.WriteFile(path, []byte(kubeconfig), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	_, err := newKubernetesClient(path, time.Second, http.DefaultTransport)
	if err == nil || !strings.Contains(err.Error(), "parse kubeconfig ca failed") {
		t.Fatalf("expected ca parse error, got %v", err)
	}
}
