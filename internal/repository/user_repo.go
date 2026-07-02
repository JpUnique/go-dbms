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
    SELECT id, email, password_hash, name, role, department, status, created_at, updated_at
    FROM users
    WHERE id = $1
    `

	user := &models.User{}

	err := r.db.QueryRow(ctx, query, id).
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
		return fmt.Errorf("no user updated")
	}

	return nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]models.User, error) {

	query := `
    SELECT u.id, u.email, u.name, u.role, u.department, u.status,
           u.last_login, u.created_at, u.updated_at,
           COALESCE(tf.enabled AND tf.verified, false) AS two_factor_enabled
    FROM users u
    LEFT JOIN user_two_factor tf ON tf.user_id = u.id
    ORDER BY u.created_at DESC
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
			&user.LastLogin,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.TwoFactorEnabled,
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

func (r *UserRepository) GetAdmins(ctx context.Context) ([]models.User, error) {
	query := `SELECT id, email, name, role, department, status, created_at, updated_at FROM users WHERE role = 'admin' AND status = 'active'`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get admins: %w", err)
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Department, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) GetByEmailOrUsername(
	ctx context.Context,
	identifier string,
) (*models.User, error) {

	query := `
        SELECT id, email, name, password_hash, status, role
        FROM users
        WHERE LOWER(email) = LOWER($1)
           OR LOWER(name) = LOWER($1)
    `

	row := r.db.QueryRow(ctx, query, identifier)

	var user models.User

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByUsername(
	ctx context.Context,
	username string,
) (*models.User, error) {

	query := `
    SELECT id, email, name, password_hash, status, role
    FROM users
    WHERE name = $1
    `

	row := r.db.QueryRow(ctx, query, username)

	var user models.User

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetTwoFactor(
	ctx context.Context,
	userID string,
) (*models.UserTwoFactor, error) {

	query := `
    SELECT secret, enabled, verified, created_at, updated_at
    FROM user_two_factor
    WHERE user_id = $1
    `

	var tf models.UserTwoFactor
	tf.UserID = userID

	err := r.db.QueryRow(ctx, query, userID).
		Scan(
			&tf.Secret,
			&tf.Enabled,
			&tf.Verified,
			&tf.CreatedAt,
			&tf.UpdatedAt,
		)

	if err != nil {
		return nil, err
	}

	return &tf, nil
}

func (r *UserRepository) SetTwoFactorSecret(
	ctx context.Context,
	userID string,
	secret string,
) error {

	query := `
    INSERT INTO user_two_factor (user_id, secret, enabled, verified)
    VALUES ($1, $2, true, false)
    ON CONFLICT (user_id)
    DO UPDATE SET
        secret = EXCLUDED.secret,
        enabled = true,
        verified = false,
        updated_at = NOW()
    `

	_, err := r.db.Exec(ctx, query, userID, secret)
	return err
}

func (r *UserRepository) VerifyTwoFactor(
	ctx context.Context,
	userID string,
) error {

	query := `
    UPDATE user_two_factor
    SET verified = true,
        updated_at = NOW()
    WHERE user_id = $1
    `

	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) GetPreferences(
	ctx context.Context,
	userID string,
) (*models.UserPreferences, error) {

	// include timestamps so caller can inspect when prefs were last changed
	query := `
	SELECT dark_mode, email_notifications, created_at, updated_at
	FROM user_preferences
	WHERE user_id = $1
	`

	// sensible default values (match table defaults)
	prefs := &models.UserPreferences{
		DarkMode:           false,
		EmailNotifications: true,
	}

	err := r.db.QueryRow(ctx, query, userID).
		Scan(&prefs.DarkMode, &prefs.EmailNotifications, &prefs.CreatedAt, &prefs.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// no preferences row yet — return defaults rather than an error
			return prefs, nil
		}
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	return prefs, nil
}

func (r *UserRepository) UpsertPreferences(
	ctx context.Context,
	userID string,
	darkMode bool,
	notifications bool,
) error {

	query := `
    INSERT INTO user_preferences (user_id, dark_mode, email_notifications)
    VALUES ($1, $2, $3)
    ON CONFLICT (user_id)
    DO UPDATE SET
        dark_mode = EXCLUDED.dark_mode,
        email_notifications = EXCLUDED.email_notifications,
        updated_at = NOW()
    `

	_, err := r.db.Exec(ctx, query, userID, darkMode, notifications)
	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}

	return nil
}

type DepartmentStat struct {
	Department string `json:"department"`
	UserCount  int    `json:"user_count"`
}

func (r *UserRepository) GetDepartmentStats(ctx context.Context) ([]DepartmentStat, error) {
	query := `
		SELECT department, COUNT(*) AS user_count
		FROM users
		WHERE department IS NOT NULL AND department != ''
		GROUP BY department
		ORDER BY user_count DESC, department ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get department stats: %w", err)
	}
	defer rows.Close()

	var stats []DepartmentStat
	for rows.Next() {
		var s DepartmentStat
		if err := rows.Scan(&s.Department, &s.UserCount); err != nil {
			return nil, fmt.Errorf("scan department stat: %w", err)
		}
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []DepartmentStat{}
	}
	return stats, nil
}

// AdminUpdateUser updates a user's editable fields (name, email, role, department).
func (r *UserRepository) AdminUpdateUser(
	ctx context.Context,
	userID, name, email, role string,
	department *string,
) (*models.User, error) {
	query := `
	UPDATE users
	SET name = $1, email = $2, role = $3, department = $4, updated_at = NOW()
	WHERE id = $5
	RETURNING id, email, name, role, department, status, created_at, updated_at
	`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, name, email, role, department, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role,
		&user.Department, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("admin update user: %w", err)
	}
	return user, nil
}

// UpdateStatus sets a user's status to "active" or "inactive".
func (r *UserRepository) UpdateStatus(ctx context.Context, userID, status string) (*models.User, error) {
	query := `
	UPDATE users
	SET status = $1, updated_at = NOW()
	WHERE id = $2
	RETURNING id, email, name, role, department, status, created_at, updated_at
	`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, status, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.Role,
		&user.Department, &user.Status, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}
	return user, nil
}

// DeleteUser hard-deletes a user by ID.
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
