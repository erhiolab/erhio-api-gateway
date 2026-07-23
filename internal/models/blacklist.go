package models

// Blacklist 全局黑名单
type Blacklist struct {
	ID          string `json:"id" db:"id"`
	Type        string `json:"type" db:"type"`
	Value       string `json:"value" db:"value"`
	Description string `json:"description" db:"description"`
}
