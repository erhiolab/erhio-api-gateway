package models

// AuthRequirement 认证要求
type AuthRequirement struct {
	SecretID  string
	Timestamp string
	Nonce     string
	Signature string
}
