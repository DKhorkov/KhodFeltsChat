package common

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const (
	OTPMin              uint64 = 100000
	OTPMax              uint64 = 999999
	OTPGenerateAttempts        = 10
)

var ErrOTPCollision = errors.New("failed to generate unique OTP code")

// GenerateOTP returns a cryptographically random 6-digit code in [OTPMin, OTPMax].
func GenerateOTP() (uint64, error) {
	span := new(big.Int).SetUint64(OTPMax - OTPMin + 1)

	n, err := rand.Int(rand.Reader, span)
	if err != nil {
		return 0, err
	}

	return n.Uint64() + OTPMin, nil
}
