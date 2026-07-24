package models

// Blacklist 全局黑名单
type Blacklist struct {
	ID          int64  `json:"id" db:"id"`
	Type        string `json:"type" db:"type"`
	Value       string `json:"value" db:"value"`
	Description string `json:"description" db:"description"`
}
