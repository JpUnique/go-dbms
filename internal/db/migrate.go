package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func RunMigration(ctx context.Context) error {

	schemaPath := "migrations/schema.sql"

	sqlBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	sql := string(sqlBytes)

	err = WithTransaction(ctx, func(tx pgx.Tx) error {

		_, err := tx.Exec(ctx, sql)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Migration completed successfully")

	return nil
}
