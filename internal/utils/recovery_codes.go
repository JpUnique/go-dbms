package utils

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// crockfordAlphabet excludes visually ambiguous characters (0/O, 1/I/L, U)
// so recovery codes are easy to read and type by hand.
const crockfordAlphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"

const (
	RecoveryCodeCount  = 10
	recoveryCodeLength = 10
)

// GenerateRecoveryCodes returns a batch of plaintext, human-formatted
// recovery codes (e.g. "ABCDE-FGHJK"). Callers must hash them (HashPassword)
// before storing — these are shown to the user exactly once.
func GenerateRecoveryCodes() ([]string, error) {

	codes := make([]string, RecoveryCodeCount)

	for i := range codes {
		raw, err := randomCrockfordString(recoveryCodeLength)
		if err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%s-%s", raw[:5], raw[5:])
	}

	return codes, nil
}

// NormalizeRecoveryCode strips formatting so a user-entered code (with or
// without the dash, any case) compares consistently against a stored hash.
func NormalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

func randomCrockfordString(length int) (string, error) {

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	out := make([]byte, length)
	for i, b := range bytes {
		out[i] = crockfordAlphabet[int(b)%len(crockfordAlphabet)]
	}

	return string(out), nil
}
