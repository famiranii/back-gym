package util

import (
	"crypto/rand"
	"math/big"
)

const discountChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateDiscountCode(length int) (string, error) {
	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(discountChars))),
		)
		if err != nil {
			return "", err
		}

		result[i] = discountChars[n.Int64()]
	}

	return string(result), nil
}
