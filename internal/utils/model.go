package utils

// UserAgent 客户端 User-Agent 解析结果
type UserAgent struct {
	UserAgent string `json:"user_agent"`
	Device    string `json:"device"`
}

// IPLocation IP 地址位置信息
type IPLocation struct {
	IP           string  `json:"ip"`
	CountryShort string  `json:"country_short"`
	CountryLong  string  `json:"country_long"`
	Region       string  `json:"region"`
	City         string  `json:"city"`
	Latitude     float32 `json:"latitude"`
	Longitude    float32 `json:"longitude"`
	Zipcode      string  `json:"zipcode"`
	Timezone     string  `json:"timezone"`
}
