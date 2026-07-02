package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/utils"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecoveryCodeRepository struct {
	db *pgxpool.Pool
}

func NewRecoveryCodeRepository(db *pgxpool.Pool) *RecoveryCodeRepository {
	return &RecoveryCodeRepository{db: db}
}

// CreateBatch stores a freshly-generated set of (already-hashed) recovery
// codes for a user. Callers should call DeleteAllForUser first if replacing
// an existing set.
func (r *RecoveryCodeRepository) CreateBatch(ctx context.Context, userID string, hashedCodes []string) error {

	batch := &pgx.Batch{}
	for _, hash := range hashedCodes {
		batch.Queue(`INSERT INTO user_recovery_codes (user_id, code_hash) VALUES ($1, $2)`, userID, hash)
	}

	results := r.db.SendBatch(ctx, batch)
	defer results.Close()

	for range hashedCodes {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("create recovery codes: %w", err)
		}
	}

	return nil
}

// FindUnused scans a user's unused recovery codes and returns the one
// matching the supplied plaintext code, if any. Recovery codes are hashed
// with bcrypt, so this is a linear scan (bounded by RecoveryCodeCount) rather
// than an indexed lookup.
func (r *RecoveryCodeRepository) FindUnused(ctx context.Context, userID string, code string) (*models.UserRecoveryCode, error) {

	rows, err := r.db.Query(ctx, `
    SELECT id, user_id, code_hash, used_at, created_at
    FROM user_recovery_codes
    WHERE user_id = $1 AND used_at IS NULL
  `, userID)
	if err != nil {
		return nil, fmt.Errorf("find recovery codes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rc models.UserRecoveryCode
		if err := rows.Scan(&rc.ID, &rc.UserID, &rc.CodeHash, &rc.UsedAt, &rc.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recovery code: %w", err)
		}
		if utils.ComparePassword(rc.CodeHash, code) == nil {
			return &rc, nil
		}
	}

	return nil, rows.Err()
}

func (r *RecoveryCodeRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
    UPDATE user_recovery_codes SET used_at = $2 WHERE id = $1
  `, id, time.Now())
	if err != nil {
		return fmt.Errorf("mark recovery code used: %w", err)
	}
	return nil
}

func (r *RecoveryCodeRepository) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_recovery_codes WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}
	return nil
}
