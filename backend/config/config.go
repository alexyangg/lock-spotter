package config

import (
	"os"
)

// Config holds all environmental variables for the microservice
type Config struct {
	Port        string
	PostgresDSN string
	RedisAddr   string
}

// LoadConfig reads values from the environment or falls back to secure local defaults
func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://lockspotter_admin:secure_password_123@localhost:5432/lockspotter_db?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}