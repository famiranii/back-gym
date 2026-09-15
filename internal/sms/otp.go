package sms

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateOTP(digits int) (string, error) {
	if digits <= 0 {
		digits = 6
	}

	max := new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(digits)),
		nil,
	)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}

	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}