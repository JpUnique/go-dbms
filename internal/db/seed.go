package db

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ======================================
// RUN SEED
// ======================================
func RunSeed(ctx context.Context) error {
	if os.Getenv("RUN_SEED") != "true" {
		fmt.Println("[SEED] skipped")
		return nil
	}

	fmt.Println("[SEED] starting database seed")

	creds, err := loadAdminCredentials()
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 10)
	if err != nil {
		return err
	}

	err = WithTransaction(ctx, func(tx pgx.Tx) error {
		return seedDatabase(ctx, tx, creds, string(passwordHash))
	})

	if err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	fmt.Println("[SEED] completed successfully ✅")
	logAdminBucket(creds.Name)
	return nil
}

// ======================================
// HELPER: LOAD ADMIN CREDENTIALS
// ======================================
type AdminCredentials struct {
	Email    string
	Password string
	Name     string
}

func loadAdminCredentials() (*AdminCredentials, error) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := os.Getenv("ADMIN_PASSWORD")
	name := strings.TrimSpace(os.Getenv("ADMIN_NAME"))

	if email == "" || password == "" || name == "" {
		return nil, fmt.Errorf("admin credentials not set properly")
	}

	return &AdminCredentials{
		Email:    email,
		Password: password,
		Name:     name,
	}, nil
}

// ======================================
// HELPER: SEED DATABASE
// ======================================
func seedDatabase(ctx context.Context, tx pgx.Tx, creds *AdminCredentials, passwordHash string) error {
	adminID, err := createAdminUser(ctx, tx, creds, passwordHash)
	if err != nil {
		return err
	}

	if err := seedDefaultTags(ctx, tx); err != nil {
		return err
	}

	if err := createRootFolder(ctx, tx, adminID); err != nil {
		return err
	}

	return nil
}

// ======================================
// HELPER: CREATE ADMIN USER
// ======================================
func createAdminUser(ctx context.Context, tx pgx.Tx, creds *AdminCredentials, passwordHash string) (string, error) {
	var adminID string

	// Try inserting new admin
	err := tx.QueryRow(ctx, `
        INSERT INTO users (email, password_hash, name, role, department, status)
        VALUES ($1, $2, $3, 'admin', $4, 'active')
        ON CONFLICT DO NOTHING
        RETURNING id
    `,
		creds.Email,
		passwordHash,
		creds.Name,
		"IT",
	).Scan(&adminID)

	// If insert didn't return, admin already exists
	if err != nil {
		err = tx.QueryRow(ctx, `
            SELECT id FROM users
            WHERE LOWER(email) = LOWER($1)
        `, creds.Email).Scan(&adminID)

		if err != nil {
			return "", err
		}
	}

	fmt.Println("[SEED] admin user ready:", creds.Name)
	return adminID, nil
}

// ======================================
// HELPER: SEED DEFAULT TAGS
// ======================================
func seedDefaultTags(ctx context.Context, tx pgx.Tx) error {
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
	return nil
}

// ======================================
// HELPER: CREATE ROOT FOLDER
// ======================================
func createRootFolder(ctx context.Context, tx pgx.Tx, adminID string) error {
	_, err := tx.Exec(ctx, `
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
}

// ======================================
// HELPER: LOG ADMIN BUCKET
// ======================================
func logAdminBucket(adminName string) {
	bucketName := "user-" + strings.ToLower(adminName)
	fmt.Println("[SEED] admin bucket would be:", bucketName)
}
