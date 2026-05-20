package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/utils"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

// constructor
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// ==============================
// CREATE USER
// ==============================
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {

	query := `
    INSERT INTO users (email, password_hash, name, role, department, status)
    VALUES ($1, $2, $3, $4, $5, 'active')
    RETURNING id, created_at, updated_at
    `

	err := r.db.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.Department,
	).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {

		// handle unique constraint (email already exists)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return utils.ErrAlreadyExists
			}
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// ==============================
// GET USER BY EMAIL
// ==============================
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {

	query := `
    SELECT id, email, password_hash, name, role, department, status, created_at, updated_at
    FROM users
    WHERE LOWER(email) = LOWER($1)
    `

	user := &models.User{}

	err := r.db.QueryRow(ctx, query, email).
		Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.Role,
			&user.Department,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err != nil {

		// user not found is NOT an error
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

// ==============================
// GET USER BY ID
// ==============================
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {

	query := `
    SELECT id, email, name, role, department, status, created_at, updated_at
    FROM users
    WHERE id = $1
    `

	user := &models.User{}

	err := r.db.QueryRow(ctx, query, id).
		Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Role,
			&user.Department,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

// ==============================
// UPDATE LAST LOGIN
// ==============================
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {

	query := `
    UPDATE users
    SET last_login = NOW()
    WHERE id = $1
    `

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}

	return nil
}

func (r *UserRepository) UpdateProfile(
	ctx context.Context,
	userID string,
	name string,
	department *string,
) (*models.User, error) {

	query := `
    UPDATE users
    SET name = $1,
        department = $2,
        updated_at = NOW()
    WHERE id = $3
    RETURNING id, email, name, role, department, status, created_at, updated_at
    `

	user := &models.User{}

	err := r.db.QueryRow(ctx, query,
		name,
		department,
		userID,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Department,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return user, nil
}

func (r *UserRepository) UpdatePassword(
	ctx context.Context,
	userID string,
	hashedPassword string,
) error {

	query := `
    UPDATE users
    SET password_hash = $1,
        updated_at = NOW()
    WHERE id = $2
    `

	cmdTag, err := r.db.Exec(ctx, query, hashedPassword, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return nil
	}

	return nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]models.User, error) {

	query := `
    SELECT id, email, name, role, department, status, created_at, updated_at
    FROM users
    ORDER BY created_at DESC
    `

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all users: %w", err)
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.Role,
			&user.Department,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}
