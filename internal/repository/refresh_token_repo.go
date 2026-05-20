package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// save token
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, tokenHash string) error {

	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
         VALUES ($1, $2, NOW() + INTERVAL '7 days')`,
		userID,
		tokenHash,
	)

	return err
}

// find valid token
func (r *RefreshTokenRepository) FindValid(ctx context.Context, tokenHash string) (bool, error) {

	var exists bool

	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
            SELECT 1 FROM refresh_tokens
            WHERE token_hash = $1
            AND revoked = false
            AND expires_at > NOW()
        )`,
		tokenHash,
	).Scan(&exists)

	return exists, err
}

// revoke all user tokens
func (r *RefreshTokenRepository) RevokeAll(ctx context.Context, userID string) error {

	_, err := r.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`,
		userID,
	)

	return err
}
