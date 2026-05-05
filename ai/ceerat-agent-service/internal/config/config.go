package config

import "os"

type Config struct {
	Port               string
	UserServiceAddress string
	OpenAIAPIKey       string
	OpenAIModel        string
}

func Load() Config {
	return Config{
		Port:               getenv("PORT", "8088"),
		UserServiceAddress: getenv("CEERAT_USER_SERVICE_ADDR", getenv("USER_SERVICE_ADDR", "localhost:50051")),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:        getenv("OPENAI_MODEL", "gpt-4.1-mini"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
