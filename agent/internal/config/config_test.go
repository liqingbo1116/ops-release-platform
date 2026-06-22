package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistRuntimeTokenUpdatesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.conf")
	input := strings.Join([]string{
		"# comment",
		"AGENT_ID=agent-test",
		"AGENT_TOKEN=",
		"AGENT_REGISTER_TOKEN=agtr_once",
		"AGENT_KUBECONFIG=/etc/kubeconfig",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := PersistRuntimeToken(path, "agt_runtime"); err != nil {
		t.Fatalf("persist runtime token: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	output := string(content)
	if !strings.Contains(output, "AGENT_TOKEN=agt_runtime") {
		t.Fatalf("expected runtime token to be persisted, got:\n%s", output)
	}
	if !strings.Contains(output, "AGENT_REGISTER_TOKEN=\n") {
		t.Fatalf("expected register token to be cleared, got:\n%s", output)
	}
	if !strings.Contains(output, "AGENT_KUBECONFIG=/etc/kubeconfig") {
		t.Fatalf("expected unrelated config to be preserved, got:\n%s", output)
	}
}

func TestPersistRuntimeTokenAppendsMissingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.conf")
	if err := os.WriteFile(path, []byte("AGENT_ID=agent-test\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := PersistRuntimeToken(path, "agt_runtime"); err != nil {
		t.Fatalf("persist runtime token: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	output := string(content)
	if !strings.Contains(output, "AGENT_TOKEN=agt_runtime") {
		t.Fatalf("expected runtime token to be appended, got:\n%s", output)
	}
	if !strings.Contains(output, "AGENT_REGISTER_TOKEN=") {
		t.Fatalf("expected register token key to be appended, got:\n%s", output)
	}
}
