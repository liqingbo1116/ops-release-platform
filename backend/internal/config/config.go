package config

import "os"

type Config struct {
	Port        string
	DatabaseDSN string
	RedisAddr   string
}

func Load() Config {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:        port,
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
	}
}
