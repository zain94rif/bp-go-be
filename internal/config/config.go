package config

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

var databaseSchemaPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Config struct {
	AppEnv            string
	AppPort           string
	DatabaseURL       string
	DatabaseSchema    string
	FrontendURL       string
	StorageDriver     string
	StoragePath       string
	JWTSecret         string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	CAPTCHARequired   bool
	CAPTCHAVerifyURL  string
	CAPTCHASEcret     string
	CAPTCHAMode       string
	SeedAdminEmail    string
	SeedAdminPassword string
	MaxUploadBytes    int64
}

func Load() Config {
	return Config{
		AppEnv:            getenv("APP_ENV", "development"),
		AppPort:           getenv("APP_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DatabaseSchema:    getenv("DATABASE_SCHEMA", "public"),
		FrontendURL:       os.Getenv("FRONTEND_URL"),
		StorageDriver:     getenv("STORAGE_DRIVER", "local"),
		StoragePath:       getenv("STORAGE_PATH", "./storage"),
		JWTSecret:         getenv("JWT_SECRET", "development-only-change-me"),
		AccessTTL:         durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:        durationEnv("REFRESH_TOKEN_TTL", 720*time.Hour),
		CAPTCHARequired:   os.Getenv("CAPTCHA_REQUIRED") != "false",
		CAPTCHAVerifyURL:  os.Getenv("CAPTCHA_VERIFY_URL"),
		CAPTCHASEcret:     os.Getenv("CAPTCHA_SECRET"),
		CAPTCHAMode:       getenv("CAPTCHA_MODE", "provider"),
		SeedAdminEmail:    os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword: os.Getenv("SEED_ADMIN_PASSWORD"),
		MaxUploadBytes:    int64Env("MAX_UPLOAD_BYTES", 25*1024*1024),
	}
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func int64Env(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		var parsed int64
		if _, err := fmt.Sscan(value, &parsed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func ValidDatabaseSchema(schema string) bool {
	return databaseSchemaPattern.MatchString(schema)
}
