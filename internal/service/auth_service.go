package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/utils"
	"github.com/jackc/pgx/v5"
)

type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
	recoveryCodeRepo *repository.RecoveryCodeRepository
}

// constructor
func NewAuthService(
	userRepo *repository.UserRepository,
	refreshRepo *repository.RefreshTokenRepository,
	recoveryCodeRepo *repository.RecoveryCodeRepository,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshRepo,
		recoveryCodeRepo: recoveryCodeRepo,
	}
}

// ==============================
// REGISTER
// ==============================
// Register creates the account and immediately starts mandatory 2FA setup.
// No access/refresh tokens are issued here — the caller must complete
// LoginVerify with the returned challenge before it can use the API.
func (s *AuthService) Register(
	ctx context.Context,
	email, password, name string,
) (user *models.User, challenge string, qrCode string, err error) {

	// normalize email (important)
	email = strings.TrimSpace(strings.ToLower(email))

	// check if user exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service register: get user: %w", err)
	}

	if existing != nil {
		return nil, "", "", utils.ErrAlreadyExists
	}

	//  hash password
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service register: hash password: %w", err)
	}

	user = &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         "viewer",
		Status:       "active",
	}

	//  insert user
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, "", "", fmt.Errorf("auth service register: create user: %w", err)
	}

	challenge, err = utils.GenerateLoginChallenge(user.ID, user.Name)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service register: challenge: %w", err)
	}

	qrCode, err = s.Enable2FA(ctx, user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service register: enable 2fa: %w", err)
	}

	return user, challenge, qrCode, nil
}

// ==============================
// GET USER BY ID
// ==============================
func (s *AuthService) GetUserByID(
	ctx context.Context,
	userID string,
) (*models.User, error) {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth service get user by id: %w", err)
	}

	if user == nil {
		return nil, utils.ErrNotFound
	}

	return user, nil
}

// ==============================
// LOGIN (step 1 — password check only)
// ==============================
// Login never issues access/refresh tokens directly. On valid credentials it
// returns a short-lived login challenge and a status telling the caller
// whether to prompt for an existing 2FA code ("2fa_required") or to walk the
// user through mandatory setup ("2fa_setup_required" — covers brand-new
// registrants and every pre-existing account that has no verified 2FA yet).
func (s *AuthService) Login(
	ctx context.Context,
	username string,
	password string,
) (status string, challenge string, qrCode string, user *models.User, err error) {

	//  normalize username
	username = strings.TrimSpace(username)

	//  get user by username ONLY
	user, err = s.userRepo.GetByUsername(ctx, username)
	if err != nil || user == nil {
		return "", "", "", nil, utils.ErrInvalidCredentials
	}

	//  check user status
	if user.Status != "active" {
		return "", "", "", nil, utils.ErrUnauthorized
	}

	//  compare password
	if err := utils.ComparePassword(user.PasswordHash, password); err != nil {
		return "", "", "", nil, utils.ErrInvalidCredentials
	}

	challenge, err = utils.GenerateLoginChallenge(user.ID, user.Name)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("auth service login: challenge: %w", err)
	}

	twoFactor, tfErr := s.userRepo.GetTwoFactor(ctx, user.ID)
	if tfErr != nil && tfErr != pgx.ErrNoRows {
		return "", "", "", nil, fmt.Errorf("auth service login: get two factor: %w", tfErr)
	}

	if tfErr == pgx.ErrNoRows || twoFactor == nil || !twoFactor.Verified {
		// No 2FA set up yet, or a previous setup attempt never completed —
		// (re)start setup with a fresh secret.
		qrCode, err = s.Enable2FA(ctx, user.ID)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("auth service login: enable 2fa: %w", err)
		}
		return "2fa_setup_required", challenge, qrCode, user, nil
	}

	return "2fa_required", challenge, "", user, nil
}

// ==============================
// LOGIN VERIFY (step 2 — code check, issues tokens)
// ==============================
// code may be a fresh 6-digit TOTP code or a one-time recovery code. On the
// first successful verification (verified flips false -> true) a fresh batch
// of recovery codes is generated and returned — shown to the caller exactly
// once. Routine logins (already verified) return no recovery codes.
func (s *AuthService) LoginVerify(
	ctx context.Context,
	challenge string,
	code string,
) (user *models.User, accessToken string, refreshToken string, recoveryCodes []string, err error) {

	claims, err := utils.VerifyLoginChallenge(challenge)
	if err != nil {
		return nil, "", "", nil, utils.ErrUnauthorized
	}

	user, err = s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("auth service login verify: get user: %w", err)
	}
	if user == nil {
		return nil, "", "", nil, utils.ErrInvalidCredentials
	}

	twoFactor, err := s.userRepo.GetTwoFactor(ctx, user.ID)
	if err != nil || twoFactor == nil || twoFactor.Secret == "" {
		return nil, "", "", nil, utils.ErrInvalidCredentials
	}

	validCode := utils.VerifyOTP(twoFactor.Secret, code)

	var usedRecovery *models.UserRecoveryCode
	if !validCode {
		usedRecovery, _ = s.recoveryCodeRepo.FindUnused(ctx, user.ID, utils.NormalizeRecoveryCode(code))
		validCode = usedRecovery != nil
	}

	if !validCode {
		return nil, "", "", nil, utils.ErrInvalidCredentials
	}

	if usedRecovery != nil {
		if err := s.recoveryCodeRepo.MarkUsed(ctx, usedRecovery.ID); err != nil {
			return nil, "", "", nil, fmt.Errorf("auth service login verify: mark recovery code used: %w", err)
		}
	}

	if !twoFactor.Verified {
		if err := s.userRepo.VerifyTwoFactor(ctx, user.ID); err != nil {
			return nil, "", "", nil, fmt.Errorf("auth service login verify: %w", err)
		}
		recoveryCodes, err = s.regenerateRecoveryCodes(ctx, user.ID)
		if err != nil {
			return nil, "", "", nil, fmt.Errorf("auth service login verify: recovery codes: %w", err)
		}
	}

	accessToken, err = utils.GenerateAccessToken(user.ID, user.Email, user.Role, user.Name)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err = utils.GenerateRefreshToken(user.ID, user.Email, user.Role, user.Name)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("generate refresh token: %w", err)
	}

	tokenHash, err := utils.HashToken(refreshToken)
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("hash refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, user.ID, tokenHash); err != nil {
		return nil, "", "", nil, fmt.Errorf("store refresh token: %w", err)
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		fmt.Println("warning: update last login failed:", err)
	}

	return user, accessToken, refreshToken, recoveryCodes, nil
}

// ==============================
// REFRESH TOKEN
// ==============================
func (s *AuthService) Refresh(
	ctx context.Context,
	refreshToken string,
) (string, error) {

	claims, err := utils.VerifyRefreshToken(refreshToken)
	if err != nil {
		return "", utils.ErrUnauthorized
	}

	tokenHash, err := utils.HashToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("auth service refresh: hash token: %w", err)
	}

	valid, err := s.refreshTokenRepo.FindValid(ctx, tokenHash)
	if err != nil {
		return "", fmt.Errorf("auth service refresh: check token: %w", err)
	}

	if !valid {
		return "", utils.ErrUnauthorized
	}

	newAccessToken, err := utils.GenerateAccessToken(
		claims.UserID,
		claims.Email,
		claims.Role,
		claims.Username,
	)
	if err != nil {
		return "", fmt.Errorf("auth service refresh: generate token: %w", err)
	}

	return newAccessToken, nil
}

// ==============================
// LOGOUT
// ==============================
func (s *AuthService) Logout(
	ctx context.Context,
	userID string,
) error {

	if err := s.refreshTokenRepo.RevokeAll(ctx, userID); err != nil {
		return fmt.Errorf("auth service logout: %w", err)
	}

	return nil
}

func (s *AuthService) UpdateProfile(
	ctx context.Context,
	userID string,
	name string,
	department *string,
) (*models.User, error) {

	user, err := s.userRepo.UpdateProfile(ctx, userID, name, department)
	if err != nil {
		return nil, fmt.Errorf("auth service update profile: %w", err)
	}

	if user == nil {
		return nil, utils.ErrNotFound
	}

	return user, nil
}

func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID string,
	oldPassword string,
	newPassword string,
) error {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("auth service change password: get user: %w", err)
	}

	if user == nil {
		return utils.ErrNotFound
	}

	if err := utils.ComparePassword(user.PasswordHash, oldPassword); err != nil {
		return utils.ErrInvalidCredentials
	}

	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth service change password: hash: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("auth service change password: update: %w", err)
	}

	return nil
}

func (s *AuthService) GetAdmins(ctx context.Context) ([]models.User, error) {
	return s.userRepo.GetAdmins(ctx)
}

func (s *AuthService) GetAllUsers(ctx context.Context) ([]models.User, error) {

	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth service get users: %w", err)
	}

	return users, nil
}

// ==============================
// RESET PASSWORD (TOTP / recovery code — no email involved)
// ==============================
// Any failure (unknown username, no verified 2FA, wrong code) returns the
// same generic error so the caller never learns which case triggered it.
func (s *AuthService) ResetPasswordViaTwoFactor(
	ctx context.Context,
	username string,
	code string,
	newPassword string,
) error {

	username = strings.TrimSpace(username)

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil || user == nil || user.Status != "active" {
		return utils.ErrInvalidCredentials
	}

	twoFactor, err := s.userRepo.GetTwoFactor(ctx, user.ID)
	if err != nil || twoFactor == nil || !twoFactor.Verified {
		return utils.ErrInvalidCredentials
	}

	validCode := utils.VerifyOTP(twoFactor.Secret, code)

	var usedRecovery *models.UserRecoveryCode
	if !validCode {
		usedRecovery, _ = s.recoveryCodeRepo.FindUnused(ctx, user.ID, utils.NormalizeRecoveryCode(code))
		validCode = usedRecovery != nil
	}

	if !validCode {
		return utils.ErrInvalidCredentials
	}

	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth service reset password: hash: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID, hash); err != nil {
		return fmt.Errorf("auth service reset password: update: %w", err)
	}

	if usedRecovery != nil {
		if err := s.recoveryCodeRepo.MarkUsed(ctx, usedRecovery.ID); err != nil {
			return fmt.Errorf("auth service reset password: mark recovery code used: %w", err)
		}
	}

	// A password reset invalidates any existing sessions.
	if err := s.refreshTokenRepo.RevokeAll(ctx, user.ID); err != nil {
		return fmt.Errorf("auth service reset password: revoke tokens: %w", err)
	}

	return nil
}

// ==============================
// 2FA
// ==============================

func (s *AuthService) Enable2FA(
	ctx context.Context,
	userID string,
) (string, error) {

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	key, err := utils.Generate2FASecret(user.Email)
	if err != nil {
		return "", err
	}

	secret := key.Secret()

	err = s.userRepo.SetTwoFactorSecret(ctx, userID, secret)
	if err != nil {
		return "", err
	}

	// return QR URL (NOT secret)
	return key.URL(), nil
}

// Verify2FA confirms a setup code and (re)generates recovery codes. Used
// both for first-time self-service setup and for "Reset 2FA" (which calls
// Enable2FA again first, then this) from Settings.
func (s *AuthService) Verify2FA(
	ctx context.Context,
	userID string,
	code string,
) ([]string, error) {
	twoFactor, err := s.userRepo.GetTwoFactor(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, utils.ErrTwoFactorNotEnabled
		}
		return nil, err
	}

	if !twoFactor.Enabled || twoFactor.Secret == "" {
		return nil, utils.ErrTwoFactorNotEnabled
	}

	if !utils.VerifyOTP(twoFactor.Secret, code) {
		return nil, utils.ErrInvalidCredentials
	}

	if err := s.userRepo.VerifyTwoFactor(ctx, userID); err != nil {
		return nil, fmt.Errorf("auth service verify 2fa: %w", err)
	}

	return s.regenerateRecoveryCodes(ctx, userID)
}

// regenerateRecoveryCodes wipes any existing codes and issues a fresh batch,
// returning the plaintext codes for one-time display.
func (s *AuthService) regenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {

	codes, err := utils.GenerateRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	hashed := make([]string, len(codes))
	for i, code := range codes {
		hash, err := utils.HashPassword(utils.NormalizeRecoveryCode(code))
		if err != nil {
			return nil, fmt.Errorf("hash recovery code: %w", err)
		}
		hashed[i] = hash
	}

	if err := s.recoveryCodeRepo.DeleteAllForUser(ctx, userID); err != nil {
		return nil, err
	}

	if err := s.recoveryCodeRepo.CreateBatch(ctx, userID, hashed); err != nil {
		return nil, err
	}

	return codes, nil
}

func (s *AuthService) GetPreferences(
	ctx context.Context,
	userID string,
) (*models.UserPreferences, error) {

	return s.userRepo.GetPreferences(ctx, userID)
}
func (s *AuthService) UpdatePreferences(
	ctx context.Context,
	userID string,
	darkMode bool,
	emailNotifications bool,
) error {

	return s.userRepo.UpsertPreferences(
		ctx,
		userID,
		darkMode,
		emailNotifications,
	)
}

func (s *AuthService) GetDepartmentStats(ctx context.Context) ([]repository.DepartmentStat, error) {
	return s.userRepo.GetDepartmentStats(ctx)
}

// AdminCreateUser creates a user with a specified role and optional department.
func (s *AuthService) AdminCreateUser(
	ctx context.Context,
	email, password, name, role string,
	department *string,
) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("admin create user: %w", err)
	}
	if existing != nil {
		return nil, utils.ErrAlreadyExists
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("admin create user: hash: %w", err)
	}

	if role == "" {
		role = "viewer"
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         role,
		Status:       "active",
		Department:   department,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("admin create user: %w", err)
	}
	return user, nil
}

// AdminUpdateUser updates name, email, role, and department for any user.
func (s *AuthService) AdminUpdateUser(
	ctx context.Context,
	userID, name, email, role string,
	department *string,
) (*models.User, error) {
	return s.userRepo.AdminUpdateUser(ctx, userID, name, email, role, department)
}

// ToggleUserStatus activates or deactivates a user account.
func (s *AuthService) ToggleUserStatus(ctx context.Context, userID, status string) (*models.User, error) {
	return s.userRepo.UpdateStatus(ctx, userID, status)
}

// AdminResetPassword sets a new password for any user.
func (s *AuthService) AdminResetPassword(ctx context.Context, userID, newPassword string) error {
	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.userRepo.UpdatePassword(ctx, userID, hash)
}

// AdminDeleteUser permanently removes a user.
func (s *AuthService) AdminDeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.DeleteUser(ctx, userID)
}
