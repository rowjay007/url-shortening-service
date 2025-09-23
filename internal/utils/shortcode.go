package utils

import (
	"crypto/rand"
	"math/big"
)

const (
	// Base62 characters (0-9, a-z, A-Z)
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// GenerateShortCode generates a random base62 string of the specified length
func GenerateShortCode(length int) (string, error) {
	result := make([]byte, length)
	
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[num.Int64()]
	}
	
	return string(result), nil
}

// IsValidShortCode validates if a short code contains only base62 characters
func IsValidShortCode(shortCode string) bool {
	if len(shortCode) == 0 {
		return false
	}
	
	for _, char := range shortCode {
		valid := false
		for _, validChar := range base62Chars {
			if char == validChar {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}
	
	return true
}
