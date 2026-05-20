package db

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// RunSeed initializes default system data
func RunSeed(ctx context.Context) error {

	fmt.Println("[SEED] starting database seed")

	adminEmail := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminName := os.Getenv("ADMIN_NAME")

	if adminEmail == "" || adminPassword == "" {
		return fmt.Errorf("admin credentials not set")
	}

	// hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 10)
	if err != nil {
		return err
	}

	err = WithTransaction(ctx, func(tx pgx.Tx) error {

		// ------------------------
		// create admin user
		// ------------------------
		var adminID string

		err := tx.QueryRow(ctx, `
            INSERT INTO users (email, password_hash, name, role, department, status)
            VALUES ($1, $2, $3, 'admin', $4, 'active')
            ON CONFLICT (LOWER(email)) DO NOTHING
            RETURNING id
        `,
			adminEmail,
			string(passwordHash),
			adminName,
			"IT",
		).Scan(&adminID)

		if err != nil {
			// since DO NOTHING was used, id may not return
			err = tx.QueryRow(ctx, `
                SELECT id FROM users
                WHERE LOWER(email) = LOWER($1)
            `, adminEmail).Scan(&adminID)

			if err != nil {
				return err
			}
		}

		fmt.Println("[SEED] admin user ready:", adminEmail)

		// ------------------------
		// insert default tags
		// ------------------------
		tags := []struct {
			Name  string
			Color string
		}{
			{"Important", "#EF4444"},
			{"Urgent", "#F59E0B"},
			{"Confidential", "#8B5CF6"},
			{"Draft", "#6B7280"},
			{"Approved", "#10B981"},
		}

		for _, tag := range tags {
			_, err := tx.Exec(ctx, `
                INSERT INTO tags (name, color)
                VALUES ($1, $2)
                ON CONFLICT DO NOTHING
            `, strings.ToLower(tag.Name), tag.Color)

			if err != nil {
				return err
			}
		}

		fmt.Println("[SEED] default tags ready")

		// ------------------------
		// create root folder
		// ------------------------
		_, err = tx.Exec(ctx, `
            INSERT INTO folders (name, parent_id, owner_id, department)
            VALUES ('Root', NULL, $1, $2)
            ON CONFLICT DO NOTHING
        `,
			adminID,
			"IT",
		)

		if err != nil {
			return err
		}

		fmt.Println("[SEED] root folder created")

		return nil
	})

	if err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	fmt.Println("[SEED] completed successfully")

	return nil
}
