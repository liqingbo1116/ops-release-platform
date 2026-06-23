package integration

import (
	"context"
	"testing"

	"ops-release-platform/backend/internal/domain"
)

func TestNewSuiteDefaultsToMock(t *testing.T) {
	suite, err := NewSuite(Config{})
	if err != nil {
		t.Fatalf("new suite: %v", err)
	}
	if suite.Jenkins == nil || suite.Registry == nil || suite.Kubernetes == nil {
		t.Fatal("expected all mock adapters to be configured")
	}
}

func TestNewSuiteRejectsUnsupportedMode(t *testing.T) {
	_, err := NewSuite(Config{Mode: "bad"})
	if err != ErrUnsupportedMode {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}

func TestNewSuiteRealRequiresConfig(t *testing.T) {
	_, err := NewSuite(Config{Mode: "real"})
	if err != ErrMissingRealConfig {
		t.Fatalf("expected missing real config error, got %v", err)
	}
}

func TestNewSuiteRealInitializesWithConfig(t *testing.T) {
	suite, err := NewSuite(Config{
		Mode: "real",
		Registries: map[string]RegistryConfig{
			"local": {URL: "http://registry.example.test"},
		},
		Clusters: map[string]ClusterConfig{
			"local": {Kubeconfig: "local.yaml"},
		},
	})
	if err != nil {
		t.Fatalf("new real suite: %v", err)
	}
	if suite.Jenkins == nil || suite.Registry == nil || suite.Kubernetes == nil {
		t.Fatal("expected all real adapters to be configured")
	}
}

func TestMockAdapters(t *testing.T) {
	ctx := context.Background()
	suite := NewMockSuite()

	build, err := suite.Jenkins.TriggerBuild(ctx, BuildRequest{JobName: "build-user-service", Branch: "main"})
	if err != nil {
		t.Fatalf("trigger build: %v", err)
	}
	if build.Status != "QUEUED" || build.BuildID == "" {
		t.Fatalf("unexpected build result: %+v", build)
	}

	image, err := suite.Registry.GetImage(ctx, "harbor.local/project-x/user-service", "20260607-a1b2c3")
	if err != nil {
		t.Fatalf("get image: %v", err)
	}
	if !image.Exists || image.Digest == "" {
		t.Fatalf("unexpected image info: %+v", image)
	}

	workloads, err := suite.Kubernetes.ListWorkloads(ctx, domain.Environment{ID: "env-project-x-prod", Namespace: "project-x-prod"})
	if err != nil {
		t.Fatalf("list workloads: %v", err)
	}
	if len(workloads) == 0 || workloads[0].ReadyReplicas == 0 {
		t.Fatalf("unexpected workloads: %+v", workloads)
	}
}
