package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AgentID           string
	EnvironmentID     string
	PlatformURL       string
	Token             string
	RegisterToken     string
	Mode              string
	HealthPort        string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	HTTPTimeout       time.Duration
	MaxTasks          int
	Capabilities      []string
	Kubeconfig        string
	HarborURL         string
	HarborUsername    string
	HarborPassword    string
	HarborInsecureTLS bool
}

func Load(configFile string) (Config, error) {
	if err := loadEnvFile(configFile); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AgentID:           strings.TrimSpace(os.Getenv("AGENT_ID")),
		EnvironmentID:     strings.TrimSpace(os.Getenv("AGENT_ENVIRONMENT_ID")),
		PlatformURL:       strings.TrimRight(strings.TrimSpace(os.Getenv("PLATFORM_URL")), "/"),
		Token:             strings.TrimSpace(os.Getenv("AGENT_TOKEN")),
		RegisterToken:     strings.TrimSpace(os.Getenv("AGENT_REGISTER_TOKEN")),
		Mode:              envWithDefault("AGENT_MODE", "remote-probe"),
		HealthPort:        envWithDefault("AGENT_HEALTH_PORT", "18080"),
		PollInterval:      secondsEnv("AGENT_POLL_INTERVAL_SECONDS", 5),
		HeartbeatInterval: secondsEnv("AGENT_HEARTBEAT_INTERVAL_SECONDS", 15),
		HTTPTimeout:       secondsEnv("AGENT_HTTP_TIMEOUT_SECONDS", 10),
		MaxTasks:          intEnv("AGENT_MAX_TASKS", 1),
		Capabilities:      splitCSV(envWithDefault("AGENT_CAPABILITIES", "remote-probe,k8s-api,http-check")),
		Kubeconfig:        strings.TrimSpace(os.Getenv("AGENT_KUBECONFIG")),
		HarborURL:         strings.TrimRight(strings.TrimSpace(os.Getenv("AGENT_HARBOR_URL")), "/"),
		HarborUsername:    strings.TrimSpace(os.Getenv("AGENT_HARBOR_USERNAME")),
		HarborPassword:    strings.TrimSpace(os.Getenv("AGENT_HARBOR_PASSWORD")),
		HarborInsecureTLS: boolEnv("AGENT_HARBOR_INSECURE_SKIP_TLS_VERIFY", false),
	}
	if cfg.AgentID == "" {
		hostname, _ := os.Hostname()
		cfg.AgentID = "agent-" + normalizeID(hostname)
	}
	if cfg.AgentID == "agent-" {
		return Config{}, errors.New("AGENT_ID is required when hostname cannot be resolved")
	}
	if cfg.PlatformURL == "" {
		return Config{}, errors.New("PLATFORM_URL is required")
	}
	if cfg.Token == "" && cfg.RegisterToken == "" {
		return Config{}, errors.New("AGENT_TOKEN or AGENT_REGISTER_TOKEN is required")
	}
	if cfg.Mode != "remote-probe" {
		return Config{}, errors.New("AGENT_MODE only supports remote-probe")
	}
	if cfg.MaxTasks != 1 {
		return Config{}, errors.New("only AGENT_MAX_TASKS=1 is supported in v1")
	}
	if cfg.HarborURL != "" {
		if _, err := url.ParseRequestURI(cfg.HarborURL); err != nil {
			return Config{}, fmt.Errorf("AGENT_HARBOR_URL is invalid: %w", err)
		}
	}
	return cfg, nil
}

func normalizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteByte('-')
	}
	return strings.Trim(builder.String(), "-")
}

func loadEnvFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("open config file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("parse config file %s line %d: missing '='", path, lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("parse config file %s line %d: empty key", path, lineNo)
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s from %s: %w", key, path, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}

	return nil
}

func PersistRuntimeToken(configFile string, token string) error {
	configFile = strings.TrimSpace(configFile)
	token = strings.TrimSpace(token)
	if configFile == "" {
		return errors.New("config file path is required")
	}
	if token == "" {
		return errors.New("agent token is required")
	}

	cleanPath := filepath.Clean(configFile)
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", configFile, err)
	}

	lines := strings.Split(string(content), "\n")
	hasTrailingNewline := strings.HasSuffix(string(content), "\n")
	tokenUpdated := false
	registerTokenUpdated := false
	for index, line := range lines {
		key, _, ok := parseEnvAssignment(line)
		if !ok {
			continue
		}
		switch key {
		case "AGENT_TOKEN":
			lines[index] = "AGENT_TOKEN=" + token
			tokenUpdated = true
		case "AGENT_REGISTER_TOKEN":
			lines[index] = "AGENT_REGISTER_TOKEN="
			registerTokenUpdated = true
		}
	}

	if !tokenUpdated {
		lines = append(lines, "AGENT_TOKEN="+token)
	}
	if !registerTokenUpdated {
		lines = append(lines, "AGENT_REGISTER_TOKEN=")
	}

	output := strings.Join(lines, "\n")
	if hasTrailingNewline && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("stat config file %s: %w", configFile, err)
	}
	if err := os.WriteFile(cleanPath, []byte(output), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write config file %s: %w", configFile, err)
	}
	return nil
}

func parseEnvAssignment(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	key, value, ok := strings.Cut(trimmed, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(value), true
}

func envWithDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func secondsEnv(key string, fallback int) time.Duration {
	return time.Duration(intEnv(key, fallback)) * time.Second
}

func intEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "y"
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
