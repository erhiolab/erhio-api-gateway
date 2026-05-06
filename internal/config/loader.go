package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

// Load 加载基础配置
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("读取配置文件失败: %v", err)
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("解析配置文件失败: %v", err)
		return nil, err
	}
	return &config, nil
}

// LoadDatabaseConfig 从数据库加载配置
func LoadDatabaseConfig(db *sqlx.DB) (*DatabaseConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 查询所有配置项
	type ConfigItem struct {
		ID          string `db:"id"`
		GroupName   string `db:"group_name"`
		Type        string `db:"type"`
		Value       string `db:"value"`
		Description string `db:"description"`
		Version     int    `db:"version"`
	}
	var configItems []ConfigItem
	query := `SELECT id, group_name, type, value, description, version FROM api_gateway_config`
	err := db.SelectContext(ctx, &configItems, query)
	if err != nil {
		return nil, err
	}
	// 构建配置映射
	configMap := make(map[string]string)
	for _, item := range configItems {
		configMap[item.ID] = item.Value
	}
	// 解析配置
	dbConfig := &DatabaseConfig{}
	// 本地缓存过期时间
	if val, ok := configMap["gateway.local-cache-expire"]; ok {
		if expire, err := strconv.Atoi(val); err == nil {
			dbConfig.LocalCacheExpire = expire
		}
	}
	// Redis缓存过期时间
	if val, ok := configMap["gateway.redis-cache-expire"]; ok {
		if expire, err := strconv.Atoi(val); err == nil {
			dbConfig.RedisCacheExpire = expire
		}
	}
	// 网关API根路由
	if val, ok := configMap["gateway.api-root"]; ok {
		dbConfig.ApiRoot = val
	}
	// 节点超时时间
	if val, ok := configMap["gateway.node-timeout"]; ok {
		if timeout, err := strconv.Atoi(val); err == nil {
			dbConfig.NodeTimeout = timeout
		}
	}
	// 总超时时间
	if val, ok := configMap["gateway.total-timeout"]; ok {
		if timeout, err := strconv.Atoi(val); err == nil {
			dbConfig.TotalTimeout = timeout
		}
	}
	// 邮件配置
	emailConfig := EmailConfig{}
	// 邮件主机
	if val, ok := configMap["email.host"]; ok {
		emailConfig.Host = val
	}
	// 邮件端口
	if val, ok := configMap["email.port"]; ok {
		if port, err := strconv.Atoi(val); err == nil {
			emailConfig.Port = port
		}
	}
	// 邮件用户名
	if val, ok := configMap["email.username"]; ok {
		emailConfig.Username = val
	}
	// 邮件密码
	if val, ok := configMap["email.password"]; ok {
		emailConfig.Password = val
	}
	// 邮件超时时间
	if val, ok := configMap["email.timeout"]; ok {
		if timeout, err := strconv.Atoi(val); err == nil {
			emailConfig.Timeout = timeout
		}
	}
	// 邮件最大重试次数
	if val, ok := configMap["email.max-retry"]; ok {
		if retry, err := strconv.Atoi(val); err == nil {
			emailConfig.MaxRetry = retry
		}
	}
	// 邮件配置
	dbConfig.Email = emailConfig
	// 认证配置
	authConfig := AuthConfig{}
	// 加密密钥
	if val, ok := configMap["auth.master-key"]; ok {
		authConfig.MasterKey = val
	}
	// 时间戳窗口
	if val, ok := configMap["auth.timestamp-window"]; ok {
		if window, err := strconv.ParseInt(val, 10, 64); err == nil {
			authConfig.TimestampWindow = window
		}
	}
	// nonce 唯一性校验窗口(次)
	if val, ok := configMap["auth.nonce-window"]; ok {
		if window, err := strconv.ParseInt(val, 10, 64); err == nil {
			authConfig.NonceWindow = window
		}
	}
	// nonce 唯一性校验窗口(小时)
	if val, ok := configMap["auth.nonce-window-hour"]; ok {
		if window, err := strconv.ParseInt(val, 10, 64); err == nil {
			authConfig.NonceWindowHour = window
		}
	}
	// 每秒最大请求数
	if val, ok := configMap["auth.qps-limit"]; ok {
		if limit, err := strconv.ParseInt(val, 10, 64); err == nil {
			authConfig.QpsLimit = limit
		}
	}
	// 每分钟最大请求数
	if val, ok := configMap["auth.qpm-limit"]; ok {
		if limit, err := strconv.ParseInt(val, 10, 64); err == nil {
			authConfig.QpmLimit = limit
		}
	}
	// 全局IP黑名单
	if val, ok := configMap["auth.ip-black-list"]; ok {
		if val != "" {
			authConfig.IPBlacklist = strings.Split(strings.TrimSpace(val), "\n")
		}
	}
	// 全局国家黑名单
	if val, ok := configMap["auth.country-black-list"]; ok {
		if val != "" {
			authConfig.CountryBlackList = strings.Split(strings.TrimSpace(val), "\n")
		}
	}
	dbConfig.Auth = authConfig
	return dbConfig, nil
}

// MergeConfig 合并配置
func MergeConfig(base *Config, dbConfig *DatabaseConfig) *Config {
	base.DatabaseConfig = *dbConfig
	return base
}
