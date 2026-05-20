package config

import (
	"log"
	"os"
)

// Config structure
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
		AccessSecret  string
		RefreshSecret string
	}

	MinIO struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		Bucket    string
		UseSSL    string
	}
}

// LoadConfig loads all environment variables
func LoadConfig() *Config {

	cfg := &Config{}

	cfg.Port = getEnv("PORT", "4000")

	// API version can be used for future versioning of API endpoints
	cfg.API.Version = getEnv("API_VERSION", "v1")

	// database
	cfg.DB.URL = mustGetEnv("DATABASE_URL")

	// jwt
	cfg.JWT.AccessSecret = mustGetEnv("JWT_ACCESS_SECRET")
	cfg.JWT.RefreshSecret = mustGetEnv("JWT_REFRESH_SECRET")

	// minio
	cfg.MinIO.Endpoint = mustGetEnv("MINIO_ENDPOINT")
	cfg.MinIO.AccessKey = mustGetEnv("MINIO_ACCESS_KEY")
	cfg.MinIO.SecretKey = mustGetEnv("MINIO_SECRET_KEY")
	cfg.MinIO.Bucket = mustGetEnv("MINIO_BUCKET")
	cfg.MinIO.UseSSL = getEnv("MINIO_USE_SSL", "false")
	cfg.RunMigration = getEnv("RUN_MIGRATION", "false") == "true"
	cfg.RunSeed = getEnv("RUN_SEED", "false") == "true"

	return cfg
}

// helpers

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
