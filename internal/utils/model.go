package utils

// UserAgent 客户端 User-Agent 解析结果
type UserAgent struct {
	UserAgent string `json:"user_agent"`
	Device    string `json:"device"`
}
