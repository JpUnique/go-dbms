package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/JpUnique/go-dbms/internal/models"
	"github.com/JpUnique/go-dbms/internal/repository"
	"github.com/JpUnique/go-dbms/internal/utils"
)

type AuthService struct {
	userRepo         *repository.UserRepository
	refreshTokenRepo *repository.RefreshTokenRepository
}

// constructor
func NewAuthService(
	userRepo *repository.UserRepository,
	refreshRepo *repository.RefreshTokenRepository,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshRepo,
	}
}

// ==============================
// REGISTER
// ==============================
func (s *AuthService) Register(
	ctx context.Context,
	email, password, name string,
) (*models.User, error) {

	// normalize email (important)
	email = strings.TrimSpace(strings.ToLower(email))

	// check if user exists
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth service register: get user: %w", err)
	}

	if existing != nil {
		return nil, utils.ErrAlreadyExists
	}

	//  hash password
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("auth service register: hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         "viewer",
		Status:       "active",
	}

	//  insert user
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("auth service register: create user: %w", err)
	}

	return user, nil
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
// LOGIN
// ==============================
func (s *AuthService) Login(
	ctx context.Context,
	email, password string,
) (*models.User, string, string, error) {

	// normalize email
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service login: get user: %w", err)
	}

	if user == nil {
		return nil, "", "", utils.ErrInvalidCredentials
	}

	if user.Status != "active" {
		return nil, "", "", utils.ErrUnauthorized
	}

	// compare password
	if err := utils.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, "", "", utils.ErrInvalidCredentials
	}

	// generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service login: generate access token: %w", err)
	}

	//generate refresh token
	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service login: generate refresh token: %w", err)
	}

	// hash refresh token
	tokenHash, err := utils.HashToken(refreshToken)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth service login: hash refresh token: %w", err)
	}

	// store refresh token
	if err := s.refreshTokenRepo.Create(ctx, user.ID, tokenHash); err != nil {
		return nil, "", "", fmt.Errorf("auth service login: store refresh token: %w", err)
	}

	// update last login (non-critical)
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// do NOT fail login, just log for debugging
		fmt.Println("auth service login warning: update last login failed:", err)
	}

	return user, accessToken, refreshToken, nil
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

func (s *AuthService) GetAllUsers(ctx context.Context) ([]models.User, error) {

	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth service get users: %w", err)
	}

	return users, nil
}
