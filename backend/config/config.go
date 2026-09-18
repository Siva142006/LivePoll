package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDatabase  string
	RedisURL       string
	JWTSecret      string
	JWTExpiration  time.Duration
	Environment    string
	FrontendOrigin string
}

func Load() *Config {
	_ = godotenv.Load()

	jwtExp := time.Hour * 24
	if value := os.Getenv("JWT_EXPIRATION"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			jwtExp = parsed
		}
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDatabase:  getEnv("MONGODB_DATABASE", "livepoll"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiration:  jwtExp,
		Environment:    getEnv("APP_ENV", "development"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func MustInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
