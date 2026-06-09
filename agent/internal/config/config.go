package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AgentID           string
	EnvironmentID     string
	PlatformURL       string
	Token             string
	Mode              string
	HealthPort        string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	HTTPTimeout       time.Duration
	Capabilities      []string
}

func Load() (Config, error) {
	cfg := Config{
		AgentID:           strings.TrimSpace(os.Getenv("AGENT_ID")),
		EnvironmentID:     strings.TrimSpace(os.Getenv("AGENT_ENVIRONMENT_ID")),
		PlatformURL:       strings.TrimRight(strings.TrimSpace(os.Getenv("PLATFORM_URL")), "/"),
		Token:             strings.TrimSpace(os.Getenv("AGENT_TOKEN")),
		Mode:              envWithDefault("AGENT_MODE", "mock"),
		HealthPort:        envWithDefault("AGENT_HEALTH_PORT", "18080"),
		PollInterval:      secondsEnv("AGENT_POLL_INTERVAL_SECONDS", 5),
		HeartbeatInterval: secondsEnv("AGENT_HEARTBEAT_INTERVAL_SECONDS", 15),
		HTTPTimeout:       secondsEnv("AGENT_HTTP_TIMEOUT_SECONDS", 10),
		Capabilities:      splitCSV(envWithDefault("AGENT_CAPABILITIES", "mock-executor,image-sync,kubectl,http-check")),
	}
	if cfg.AgentID == "" {
		return Config{}, errors.New("AGENT_ID is required")
	}
	if cfg.EnvironmentID == "" {
		return Config{}, errors.New("AGENT_ENVIRONMENT_ID is required")
	}
	if cfg.PlatformURL == "" {
		return Config{}, errors.New("PLATFORM_URL is required")
	}
	if cfg.Mode != "mock" {
		return Config{}, errors.New("only AGENT_MODE=mock is supported before Jenkins/Harbor/K8s integration")
	}
	return cfg, nil
}

func envWithDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func secondsEnv(key string, fallback int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(value) * time.Second
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
