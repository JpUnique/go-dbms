package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// Initialize DB connection
func ConnectDB() *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("Invalid database config:", err)
	}

	// Pool configuration (similar to pool.ts)
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal("Unable to connect to DB:", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	log.Println("Connected to PostgreSQL")

	Pool = pool
	return pool
}

// Query helper (similar to pool.query)
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

// Transaction helper
func WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

// Graceful shutdown
func CloseDB() {
	if Pool != nil {
		log.Println("Closing database pool")
		Pool.Close()
	}
}
