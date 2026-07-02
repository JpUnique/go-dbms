package utils

import (
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func VerifyOTP(secret string, code string) bool {

	valid, err := totp.ValidateCustom(
		code,
		secret,
		time.Now(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
	if err != nil {
		return false
	}
	return valid
}

// Generate2FASecret creates a new TOTP secret for a user
func Generate2FASecret(email string) (*otp.Key, error) {

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "DBMS App", // your app name
		AccountName: email,      // user email
	})

	if err != nil {
		return nil, err
	}

	return key, nil
}
