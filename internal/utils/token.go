package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// HashToken hashes refresh tokens before storing
func HashToken(token string) (string, error) {

	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:]), nil
}

// GenerateRandomToken generates a secure random token of given byte length
func GenerateRandomToken(length int) (string, error) {

	bytes := make([]byte, length)

	// fill with secure random data
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// encode to URL-safe base64 string
	token := base64.URLEncoding.EncodeToString(bytes)

	return token, nil
}
