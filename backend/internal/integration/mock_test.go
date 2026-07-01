package integration

import "testing"

func TestNewSuiteDefaultsToReal(t *testing.T) {
	suite, err := NewSuite(Config{})
	if err != nil {
		t.Fatalf("new suite: %v", err)
	}
	if suite.Jenkins == nil || suite.Registry == nil || suite.Kubernetes == nil {
		t.Fatal("expected all adapters to be configured")
	}
	if _, ok := suite.Jenkins.(RealJenkinsAdapter); !ok {
		t.Fatalf("expected Jenkins to use real adapter, got %T", suite.Jenkins)
	}
}

func TestNewSuiteRealInitializesJenkinsWithoutResourceConfig(t *testing.T) {
	suite, err := NewSuite(Config{})
	if err != nil {
		t.Fatalf("new real suite: %v", err)
	}
	if suite.Jenkins == nil || suite.Registry == nil || suite.Kubernetes == nil {
		t.Fatal("expected real suite adapters to be configured")
	}
	if _, ok := suite.Registry.(UnsupportedRegistryAdapter); !ok {
		t.Fatalf("expected registry adapter to be unsupported without config, got %T", suite.Registry)
	}
	if _, ok := suite.Kubernetes.(UnsupportedKubernetesAdapter); !ok {
		t.Fatalf("expected kubernetes adapter to be unsupported without config, got %T", suite.Kubernetes)
	}
}

func TestNewSuiteRealInitializesWithConfig(t *testing.T) {
	suite, err := NewSuite(Config{
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
	if _, ok := suite.Registry.(RealRegistryAdapter); !ok {
		t.Fatalf("expected real registry adapter, got %T", suite.Registry)
	}
	if _, ok := suite.Kubernetes.(RealKubernetesAdapter); !ok {
		t.Fatalf("expected real kubernetes adapter, got %T", suite.Kubernetes)
	}
}
