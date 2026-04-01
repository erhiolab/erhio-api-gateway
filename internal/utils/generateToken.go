package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateToken 生成随机 Token
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
