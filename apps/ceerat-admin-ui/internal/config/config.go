package config

import "os"

type Config struct {
	Port         string
	APIBaseURL   string
	AdminBaseURL string
	Env          string
}

func Load() Config {
	return Config{
		Port:         env("CEERAT_ADMIN_UI_PORT", "3010"),
		APIBaseURL:   env("CEERAT_API_BASE_URL", "localhost:50051"),
		AdminBaseURL: env("CEERAT_ADMIN_API_BASE_URL", "http://localhost:8081"),
		Env:          env("CEERAT_ENV", env("APP_ENV", "development")),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
