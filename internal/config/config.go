package config

import "os"

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	GoogleClientIDs string
}

func Load() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "bingoals.db"),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Port:            getEnv("PORT", "8080"),
		GoogleClientIDs: getEnv("GOOGLE_CLIENT_IDS", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
