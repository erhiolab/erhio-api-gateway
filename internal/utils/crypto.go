package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"elake-api-gateway/internal/config"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
)

// GenerateCredentials 生成认证凭证
func GenerateCredentials() (string, string, string, error) {
	// 生成密钥 ID
	secretID, err := generateToken(8)
	if err != nil {
		return "", "", "", err
	}
	// 生成密钥
	secretKey, err := generateToken(32)
	if err != nil {
		return "", "", "", err
	}
	// 加密密钥
	encKey, err := EncryptSecret(secretKey)
	if err != nil {
		return "", "", "", err
	}
	return "elk_" + secretID, secretKey, encKey, nil
}

// EncryptSecret 加密密钥
func EncryptSecret(plain string) (string, error) {
	cfg := config.Get()
	block, err := aes.NewCipher([]byte(cfg.DatabaseConfig.Auth.MasterKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret 解密密钥
func DecryptSecret(enc string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	cfg := config.Get()
	block, err := aes.NewCipher([]byte(cfg.DatabaseConfig.Auth.MasterKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// Verify 验证签名
func Verify(secretKey string, r *http.Request, timestamp, nonce, signature string) bool {
	if secretKey == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	payload, err := buildPayload(r, timestamp, nonce)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal(
		[]byte(strings.ToLower(expected)),
		[]byte(strings.ToLower(signature)),
	)
}

// buildPayload 构建签名 payload
func buildPayload(r *http.Request, timestamp, nonce string) (string, error) {
	// host
	host := r.Header.Get("Host")
	if host == "" {
		host = r.Host
	}
	// 读取 body
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return "", err
	}
	// body hash
	hash := sha256.Sum256(bodyBytes)
	// query(自动排序)
	queryStr := r.URL.Query().Encode()
	// content type
	contentType := r.Header.Get("Content-Type")
	// 拼接 payload
	payload := strings.Join([]string{
		strings.ToUpper(r.Method),
		host,
		r.URL.Path,
		queryStr,
		contentType,
		hex.EncodeToString(hash[:]),
		timestamp,
		nonce,
	}, "\n")
	return payload, nil
}

// readBodyBytes 读取 body 内容
func readBodyBytes(r *http.Request) ([]byte, error) {
	var bodyBytes []byte
	var err error
	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		// 还原 body
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return bodyBytes, nil
}
