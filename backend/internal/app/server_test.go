package app

import (
	"testing"

	"ops-release-platform/backend/internal/config"
)

func TestValidateRuntimeConfigRequiresDatabaseDSN(t *testing.T) {
	err := validateRuntimeConfig(config.Config{RedisAddr: "100.120.3.230:6379"})
	if err == nil || err.Error() != "DATABASE_DSN is required for V1 mainline runtime" {
		t.Fatalf("expected DATABASE_DSN requirement, got %v", err)
	}
}

func TestValidateRuntimeConfigRequiresRedisAddr(t *testing.T) {
	err := validateRuntimeConfig(config.Config{DatabaseDSN: "postgres://user:pass@100.120.3.230:5432/app"})
	if err == nil || err.Error() != "REDIS_ADDR is required for V1 mainline runtime" {
		t.Fatalf("expected REDIS_ADDR requirement, got %v", err)
	}
}

func TestValidateRuntimeConfigAcceptsRealDataRuntime(t *testing.T) {
	err := validateRuntimeConfig(config.Config{
		DatabaseDSN: "postgres://user:pass@100.120.3.230:5432/app",
		RedisAddr:   "100.120.3.230:6379",
	})
	if err != nil {
		t.Fatalf("expected valid runtime config, got %v", err)
	}
}
