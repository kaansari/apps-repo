package config

import "os"

type Config struct {
	Env          string
	Port         string
	AgentBaseURL string
	APIBaseURL   string
}

// LoadFromEnv reads simple configuration from environment variables.
func LoadFromEnv() Config {
	return Config{
		Env:          env("CEERAT_ENV", env("APP_ENV", "development")),
		Port:         env("CEERAT_CUSTOMER_UI_PORT", env("PORT", "3005")),
		AgentBaseURL: env("CEERAT_AGENT_BASE_URL", env("AGENT_BASE_URL", "http://localhost:8088")),
		APIBaseURL:   env("CEERAT_API_BASE_URL", env("API_BASE_URL", "http://localhost:8080")),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
