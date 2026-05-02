package config

import "os"

type Config struct {
	Port       string
	APIBaseURL string
	Env        string
}

func Load() Config {
	return Config{
		Port:       env("CEERAT_WEB_UI_PORT", "3000"),
		APIBaseURL: env("CEERAT_API_BASE_URL", "http://localhost:8080"),
		Env:        env("CEERAT_ENV", env("APP_ENV", "development")),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
