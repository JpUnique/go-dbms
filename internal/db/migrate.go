package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5"
)

// ======================================
// RUN MIGRATION
// ======================================
func RunMigration(ctx context.Context) error {

	// control via env
	if os.Getenv("RUN_MIGRATION") != "true" {
		fmt.Println("[MIGRATION] skipped")
		return nil
	}

	fmt.Println("[MIGRATION] starting...")

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("unable to determine migration source path")
	}

	schemaPath := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "schema.sql")
	schemaPath = filepath.Clean(schemaPath)

	sqlBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema file failed: %w", err)
	}

	sql := string(sqlBytes)

	// guard against empty migration
	if len(sql) == 0 {
		return fmt.Errorf("schema.sql is empty")
	}

	fmt.Println("[MIGRATION] executing schema.sql")

	err = WithTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, sql)
		return err
	})

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("[MIGRATION] completed successfully ✅")
	return nil
}
