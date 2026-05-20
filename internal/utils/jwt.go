package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// custom errors
var (
	ErrInvalidToken  = errors.New("invalid or expired token")
	ErrMissingSecret = errors.New("jwt secret not set")
)

// JWT Claims
type JwtClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ============================================
// GENERATE ACCESS TOKEN
// ============================================
func GenerateAccessToken(userID, email, role string) (string, error) {

	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		return "", ErrMissingSecret
	}

	claims := JwtClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // ✅ ensures unique token
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ============================================
// GENERATE REFRESH TOKEN
// ============================================
func GenerateRefreshToken(userID, email, role string) (string, error) {

	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		return "", ErrMissingSecret
	}

	claims := JwtClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // ✅ always unique
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ============================================
// VERIFY ACCESS TOKEN
// ============================================
func VerifyAccessToken(tokenStr string) (*JwtClaims, error) {

	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		return nil, ErrMissingSecret
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&JwtClaims{},
		func(token *jwt.Token) (interface{}, error) {

			// SECURITY: ensure correct signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err // ✅ expose real error for debugging
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ============================================
// VERIFY REFRESH TOKEN
// ============================================
func VerifyRefreshToken(tokenStr string) (*JwtClaims, error) {

	secret := os.Getenv("JWT_REFRESH_SECRET")
	if secret == "" {
		return nil, ErrMissingSecret
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&JwtClaims{},
		func(token *jwt.Token) (interface{}, error) {

			// SECURITY: ensure correct signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err // ✅ debugging-friendly
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
