package config

import "os"

type Config struct {
	Port                     string
	DatabaseDSN              string
	RedisAddr                string
	IntegrationMode          string
	LocalHarborURL           string
	LocalHarborUsername      string
	LocalHarborPassword      string
	LocalKubeconfig          string
	RemoteHarborURL          string
	RemoteHarborUsername     string
	RemoteHarborPassword     string
	RemoteKubeconfig         string
	IntegrationHTTPTimeoutMS string
}

func Load() Config {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:                     port,
		DatabaseDSN:              os.Getenv("DATABASE_DSN"),
		RedisAddr:                os.Getenv("REDIS_ADDR"),
		IntegrationMode:          envWithDefault("INTEGRATION_MODE", "mock"),
		LocalHarborURL:           os.Getenv("LOCAL_HARBOR_URL"),
		LocalHarborUsername:      os.Getenv("LOCAL_HARBOR_USERNAME"),
		LocalHarborPassword:      os.Getenv("LOCAL_HARBOR_PASSWORD"),
		LocalKubeconfig:          os.Getenv("LOCAL_K8S_KUBECONFIG"),
		RemoteHarborURL:          os.Getenv("REMOTE_HARBOR_URL"),
		RemoteHarborUsername:     os.Getenv("REMOTE_HARBOR_USERNAME"),
		RemoteHarborPassword:     os.Getenv("REMOTE_HARBOR_PASSWORD"),
		RemoteKubeconfig:         os.Getenv("REMOTE_K8S_KUBECONFIG"),
		IntegrationHTTPTimeoutMS: envWithDefault("INTEGRATION_HTTP_TIMEOUT_MS", "10000"),
	}
}

func envWithDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
