package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// ======================================
// BUILD DATABASE URL (FLEXIBLE)
// ======================================
func getDatabaseURL() string {

	// use DATABASE_URL if provided
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// fallback to individual env vars
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		host,
		port,
		name,
		sslmode,
	)
}

// ======================================
// CONNECT DATABASE
// ======================================
func ConnectDB() *pgxpool.Pool {

	dbURL := getDatabaseURL()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("Invalid database config:", err)
	}

	// pool tuning
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Unable to connect to DB:", err)
	}

	// verify connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	log.Println("✅ Connected to PostgreSQL")

	Pool = pool
	return pool
}

// ======================================
// QUERY HELPER
// ======================================
func Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {

	start := time.Now()

	rows, err := Pool.Query(ctx, sql, args...)
	duration := time.Since(start)

	if err != nil {
		log.Println("[DB ERROR]", err)
		return nil, err
	}

	log.Printf("[DB QUERY] duration=%s\n", duration)

	return rows, nil
}

// ======================================
// TRANSACTION HELPER
// ======================================
func WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// ======================================
// CLOSE DB (GRACEFUL SHUTDOWN)
// ======================================
func CloseDB() {
	if Pool != nil {
		log.Println("Closing database pool")
		Pool.Close()
	}
}
