package config

import "os"

type Config struct {
	DatabaseURL       string
	JWTSecret         string
	Port              string
	GoogleClientIDs   string
	FCMServiceAccount string
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretKey       string
	R2Bucket          string
	R2PublicURL       string
}

func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "bingoals.db"),
		JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		Port:              getEnv("PORT", "8080"),
		GoogleClientIDs:   getEnv("GOOGLE_CLIENT_IDS", ""),
		FCMServiceAccount: getEnv("FCM_SERVICE_ACCOUNT", ""),
		R2AccountID:       getEnv("CF_R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("CF_R2_ACCESS_KEY_ID", ""),
		R2SecretKey:       getEnv("CF_R2_SECRET_KEY", ""),
		R2Bucket:          getEnv("CF_R2_BUCKET", ""),
		R2PublicURL:       getEnv("CF_R2_PUBLIC_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
