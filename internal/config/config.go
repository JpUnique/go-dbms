package config

import (
	"log"
	"os"
)

// ======================================
// CONFIG STRUCT
// ======================================
type Config struct {
	Port string

	RunMigration bool
	RunSeed      bool

	API struct {
		Version string
	}

	DB struct {
		URL string
	}

	JWT struct {
		AccessSecret    string
		RefreshSecret   string
		ChallengeSecret string
	}

	MinIO struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		UseSSL    string
	}
}

// ======================================
// LOAD CONFIG
// ======================================
func LoadConfig() *Config {

	cfg := &Config{}

	// ======================================
	// SERVER
	// ======================================
	cfg.Port = getEnv("PORT", "4000")

	// ======================================
	// API VERSION
	// ======================================
	cfg.API.Version = getEnv("API_VERSION", "v1")

	// ======================================
	// DATABASE
	// ======================================
	cfg.DB.URL = mustGetEnv("DATABASE_URL")

	// ======================================
	// JWT
	// ======================================
	cfg.JWT.AccessSecret = mustGetEnv("JWT_ACCESS_SECRET")
	cfg.JWT.RefreshSecret = mustGetEnv("JWT_REFRESH_SECRET")
	cfg.JWT.ChallengeSecret = mustGetEnv("JWT_2FA_CHALLENGE_SECRET")

	// ======================================
	// MINIO ( GLOBAL BUCKET)
	// ======================================
	cfg.MinIO.Endpoint = mustGetEnv("MINIO_ENDPOINT")
	cfg.MinIO.AccessKey = mustGetEnv("MINIO_ACCESS_KEY")
	cfg.MinIO.SecretKey = mustGetEnv("MINIO_SECRET_KEY")
	cfg.MinIO.UseSSL = getEnv("MINIO_USE_SSL", "false")

	// ======================================
	// FLAGS
	// ======================================
	cfg.RunMigration = getEnv("RUN_MIGRATION", "false") == "true"
	cfg.RunSeed = getEnv("RUN_SEED", "false") == "true"

	return cfg
}

// ======================================
// HELPERS
// ======================================
func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing required env variable: %s", key)
	}
	return val
}
