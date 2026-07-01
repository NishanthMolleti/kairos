// config/config.go
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	OuraClientID      string
	OuraClientSecret  string
	OuraRedirectURL   string
	JWTSecret         string
	GroqAPIKey        string
	HuggingFaceAPIKey string
	FrontendURL       string
	Port              string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		DatabaseURL:       mustEnv("DATABASE_URL"),
		OuraClientID:      mustEnv("OURA_CLIENT_ID"),
		OuraClientSecret:  mustEnv("OURA_CLIENT_SECRET"),
		OuraRedirectURL:   mustEnv("OURA_REDIRECT_URL"),
		JWTSecret:         mustEnv("JWT_SECRET"),
		GroqAPIKey:        mustEnv("GROQ_API_KEY"),
		HuggingFaceAPIKey: mustEnv("HUGGINGFACE_API_KEY"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		Port:              getEnv("PORT", "8080"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
