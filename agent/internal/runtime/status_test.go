package runtime

import (
	"testing"

	"ops-release-platform/agent/internal/reporter"
)

func TestRegistryHostFromPayloadReadsHarborFields(t *testing.T) {
	payload := map[string]any{
		"external_url": "https://reg.example.org:5000",
	}
	if got := registryHostFromPayload(payload); got != "reg.example.org:5000" {
		t.Fatalf("expected registry host from external_url, got %q", got)
	}
}

func TestRegistryHostFromPayloadReadsConfigurationValue(t *testing.T) {
	payload := map[string]any{
		"external_url": map[string]any{
			"value": "https://reg.example.org",
		},
	}
	if got := registryHostFromPayload(payload); got != "reg.example.org" {
		t.Fatalf("expected registry host from configuration value, got %q", got)
	}
}

func TestInferRegistryHostFromWorkloads(t *testing.T) {
	workloads := []reporter.RuntimeWorkload{
		{
			Namespace: "default",
			Name:      "app",
			Containers: []reporter.RuntimeContainer{
				{Name: "app", Image: "reg.example.org:5000/project-x/app:v1"},
				{Name: "sidecar", Image: "docker.io/library/busybox:latest"},
			},
		},
	}
	if got := inferRegistryHostFromWorkloads(workloads, []string{"project-x"}); got != "reg.example.org:5000" {
		t.Fatalf("expected inferred registry host, got %q", got)
	}
}
