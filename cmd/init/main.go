package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type GatewayConfig struct {
	ID                   string   `yaml:"id"`
	IP                   string   `yaml:"ip"`
	Port                 int      `yaml:"port"`
	EmailUsername        []string `yaml:"email-username"`
	DataPath             string   `yaml:"data-path"`
	TempPath             string   `yaml:"temp-path"`
	MaxConcurrencyPerCPU int      `yaml:"max-concurrency-per-cpu"`
	RecalculateInterval  int      `yaml:"recalculate-interval"`
	MaxIdleConns         int      `yaml:"max-idle-conns"`
	MaxIdleConnsPerHost  int      `yaml:"max-idle-conns-per-host"`
	IdleConnTimeout      int      `yaml:"idle-conn-timeout"`
	UUIDPoolSize         int      `yaml:"uuid-pool-size"`
}

type LoggerConfig struct {
	Output         string `yaml:"output"`
	Level          string `yaml:"level"`
	LogPath        string `yaml:"log-path"`
	RequestLogPath string `yaml:"request-log-path"`
	MaxSize        int    `yaml:"max-size"`
	MaxBackups     int    `yaml:"max-backups"`
	MaxAge         int    `yaml:"max-age"`
	Compress       bool   `yaml:"compress"`
}

type HealthConfig struct {
	DBHealthCheckInterval         int   `yaml:"db-health-check-interval"`
	DBHealthCheckFailThreshold    int32 `yaml:"db-health-check-fail-threshold"`
	DBHealthCheckOKThreshold      int32 `yaml:"db-health-check-ok-threshold"`
	RedisHealthCheckFailThreshold int32 `yaml:"redis-health-check-fail-threshold"`
	RedisHealthCheckOKThreshold   int32 `yaml:"redis-health-check-ok-threshold"`
	RedisHealthCheckInterval      int   `yaml:"redis-health-check-interval"`
	IPDBHealthCheckFailThreshold  int32 `yaml:"ipdb-health-check-fail-threshold"`
	IPDBHealthCheckOKThreshold    int32 `yaml:"ipdb-health-check-ok-threshold"`
	IPDBHealthCheckInterval       int   `yaml:"ipdb-health-check-interval"`
}

type DBConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	Name               string `yaml:"name"`
	MaxOpenConnections int    `yaml:"max-open-connections"`
	MaxIdleConnections int    `yaml:"max-idle-connections"`
	ConnMaxLifetime    int    `yaml:"conn-max-lifetime"`
	ConnMaxIdleTime    int    `yaml:"conn-max-idle-time"`
	ReadTimeout        int    `yaml:"read-timeout"`
	WriteTimeout       int    `yaml:"write-timeout"`
}

type RedisConfig struct {
	Host                  string `yaml:"host"`
	Port                  int    `yaml:"port"`
	Password              string `yaml:"password"`
	DB                    int    `yaml:"db"`
	PoolSize              int    `yaml:"pool-size"`
	ProjectPrefix         string `yaml:"project-prefix"`
	MinIdleConnections    int    `yaml:"min-idle-connections"`
	ConnectionMaxIdleTime int    `yaml:"connection-max-idle-time"`
	ConnMaxLifetime       int    `yaml:"conn-max-lifetime"`
	DialTimeout           int    `yaml:"dial-timeout"`
	ReadTimeout           int    `yaml:"read-timeout"`
	WriteTimeout          int    `yaml:"write-timeout"`
	PoolTimeout           int    `yaml:"pool-timeout"`
}

type IPDBConfig struct {
	DownloadURL         string `yaml:"download-url"`
	FileName            string `yaml:"file-name"`
	ZipEntryName        string `yaml:"zip-entry-name"`
	Token               string `yaml:"token"`
	MaxDownloadAttempts int    `yaml:"max-download-attempts"`
}

type Config struct {
	Gateway GatewayConfig `yaml:"gateway"`
	Logger  LoggerConfig  `yaml:"logger"`
	Health  HealthConfig  `yaml:"health"`
	DB      DBConfig      `yaml:"db"`
	Redis   RedisConfig   `yaml:"redis"`
	IPDB    IPDBConfig    `yaml:"ipdb"`
}

var scanner = bufio.NewScanner(os.Stdin)

func main() {
	fmt.Println("========================================")
	fmt.Println("   Elake API Gateway 初始化程序")
	fmt.Println("========================================")

	if err := step1CreateFolders(); err != nil {
		fmt.Printf("步骤1 失败: %v\n", err)
		return
	}

	config, err := step2CreateConfigFile()
	if err != nil {
		fmt.Printf("步骤2 失败: %v\n", err)
		return
	}

	if err := step3TestDBConnection(config); err != nil {
		fmt.Printf("步骤3 失败: %v\n", err)
		return
	}

	if err := step4TestRedisConnection(config); err != nil {
		fmt.Printf("步骤4 失败: %v\n", err)
		return
	}

	if err := step5InitDatabase(config); err != nil {
		fmt.Printf("步骤5 失败: %v\n", err)
		return
	}

	if err := step6InitGatewayConfig(config); err != nil {
		fmt.Printf("步骤6 失败: %v\n", err)
		return
	}

	fmt.Println("")
	fmt.Println("========================================")
	fmt.Println("     初始化完成！")
	fmt.Println("========================================")
}

func step1CreateFolders() error {
	fmt.Println("")
	fmt.Println("【步骤1】创建目录结构")

	folders := []string{
		"elake-api-gateway",
		"elake-api-gateway/configs",
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %v", folder, err)
		}
		fmt.Printf("  ✓ 创建目录: %s\n", folder)
	}

	return nil
}

func step2CreateConfigFile() (*Config, error) {
	fmt.Println("")
	fmt.Println("【步骤2】创建配置文件")
	fmt.Println("请选择配置文件创建方式:")
	fmt.Println("  1. 关键配置模式（仅询问关键配置，其他保持默认）")
	fmt.Println("  2. 全自定义模式（所有配置项都询问）")

	choice := readIntInput("请输入选择 (1/2): ", 1, 2, 1)

	var config *Config
	var err error

	if choice == 1 {
		config, err = createKeyConfig()
	} else {
		config, err = createFullConfig()
	}

	if err != nil {
		return nil, err
	}

	configPath := "elake-api-gateway/configs/config.yaml"
	file, err := os.Create(configPath)
	if err != nil {
		return nil, fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		return nil, fmt.Errorf("写入配置文件失败: %v", err)
	}

	fmt.Printf("  ✓ 配置文件已生成: %s\n", configPath)
	return config, nil
}

func createKeyConfig() (*Config, error) {
	fmt.Println("")
	fmt.Println("--- 关键配置模式 ---")

	config := getDefaultConfig()

	fmt.Println("请输入网关配置:")
	config.Gateway.ID = readStringInput("网关ID (默认: 1): ", config.Gateway.ID)
	config.Gateway.IP = readStringInput("网关IP (默认: 127.0.0.1): ", config.Gateway.IP)
	config.Gateway.Port = readIntInput("监听端口 (默认: 8080): ", 1, 65535, 8080)

	emailCount := readIntInput("邮件通知用户数量 (默认: 0): ", 0, 10, 0)
	if emailCount > 0 {
		config.Gateway.EmailUsername = make([]string, emailCount)
		for i := 0; i < emailCount; i++ {
			config.Gateway.EmailUsername[i] = readStringInput(fmt.Sprintf("  邮件地址 %d: ", i+1), "")
		}
	}

	fmt.Println("")
	fmt.Println("请输入数据库配置:")
	config.DB.Host = readStringInput("数据库主机 (默认: localhost): ", config.DB.Host)
	config.DB.Port = readIntInput("数据库端口 (默认: 3306): ", 1, 65535, 3306)
	config.DB.User = readStringInput("数据库用户名 (默认: elake_api_gateway): ", config.DB.User)
	config.DB.Password = readPasswordInput("数据库密码: ")
	config.DB.Name = readStringInput("数据库名称 (默认: elake_api_gateway): ", config.DB.Name)

	fmt.Println("")
	fmt.Println("请输入Redis配置:")
	config.Redis.Host = readStringInput("Redis主机 (默认: localhost): ", config.Redis.Host)
	config.Redis.Port = readIntInput("Redis端口 (默认: 6379): ", 1, 65535, 6379)
	config.Redis.Password = readPasswordInput("Redis密码 (默认为空): ")

	fmt.Println("")
	fmt.Println("请输入IPDB配置:")
	config.IPDB.Token = readStringInput("IPDB令牌: ", config.IPDB.Token)

	return config, nil
}

func createFullConfig() (*Config, error) {
	fmt.Println("")
	fmt.Println("--- 全自定义模式 ---")

	config := &Config{}

	fmt.Println("【网关配置】")
	config.Gateway.ID = readStringInput("网关ID: ", "1")
	config.Gateway.IP = readStringInput("网关IP: ", "127.0.0.1")
	config.Gateway.Port = readIntInput("监听端口: ", 1, 65535, 8080)

	emailCount := readIntInput("邮件通知用户数量: ", 0, 10, 0)
	if emailCount > 0 {
		config.Gateway.EmailUsername = make([]string, emailCount)
		for i := 0; i < emailCount; i++ {
			config.Gateway.EmailUsername[i] = readStringInput(fmt.Sprintf("  邮件地址 %d: ", i+1), "")
		}
	}
	config.Gateway.DataPath = readStringInput("数据路径: ", "data")
	config.Gateway.TempPath = readStringInput("临时路径: ", "tmp")
	config.Gateway.MaxConcurrencyPerCPU = readIntInput("每个CPU核心的并发上限: ", 1, 1000, 100)
	config.Gateway.RecalculateInterval = readIntInput("重新计算并发上限的间隔(秒): ", 1, 3600, 10)
	config.Gateway.MaxIdleConns = readIntInput("最大空闲连接数: ", 1, 10000, 512)
	config.Gateway.MaxIdleConnsPerHost = readIntInput("每个主机最大空闲连接数: ", 1, 10000, 512)
	config.Gateway.IdleConnTimeout = readIntInput("空闲连接超时时间(秒): ", 1, 3600, 90)
	config.Gateway.UUIDPoolSize = readIntInput("UUID池大小: ", 100, 100000, 5000)

	fmt.Println("")
	fmt.Println("【日志配置】")
	config.Logger.Output = readStringInput("日志输出方式(both/file/stdout): ", "both")
	config.Logger.Level = readStringInput("日志级别(debug/info/warn/error): ", "info")
	config.Logger.LogPath = readStringInput("系统日志文件路径: ", "logs/api-gateway.log")
	config.Logger.RequestLogPath = readStringInput("请求日志文件路径: ", "logs/request.log")
	config.Logger.MaxSize = readIntInput("日志文件最大大小(MB): ", 1, 1000, 100)
	config.Logger.MaxBackups = readIntInput("日志文件最大备份数量: ", 1, 100, 10)
	config.Logger.MaxAge = readIntInput("日志文件最大年龄(天): ", 1, 365, 30)
	config.Logger.Compress = readBoolInput("是否压缩日志文件(true/false): ")

	fmt.Println("")
	fmt.Println("【健康检查配置】")
	config.Health.DBHealthCheckInterval = readIntInput("数据库健康检查间隔(秒): ", 1, 3600, 10)
	config.Health.DBHealthCheckFailThreshold = int32(readIntInput("数据库健康检查失败阈值: ", 1, 100, 3))
	config.Health.DBHealthCheckOKThreshold = int32(readIntInput("数据库健康检查成功阈值: ", 1, 100, 1))
	config.Health.RedisHealthCheckInterval = readIntInput("Redis健康检查间隔(秒): ", 1, 3600, 10)
	config.Health.RedisHealthCheckFailThreshold = int32(readIntInput("Redis健康检查失败阈值: ", 1, 100, 3))
	config.Health.RedisHealthCheckOKThreshold = int32(readIntInput("Redis健康检查成功阈值: ", 1, 100, 1))
	config.Health.IPDBHealthCheckInterval = readIntInput("IP数据库健康检查间隔(秒): ", 1, 3600, 20)
	config.Health.IPDBHealthCheckFailThreshold = int32(readIntInput("IP数据库健康检查失败阈值: ", 1, 100, 3))
	config.Health.IPDBHealthCheckOKThreshold = int32(readIntInput("IP数据库健康检查成功阈值: ", 1, 100, 1))

	fmt.Println("")
	fmt.Println("【数据库配置】")
	config.DB.Host = readStringInput("数据库主机: ", "localhost")
	config.DB.Port = readIntInput("数据库端口: ", 1, 65535, 3306)
	config.DB.User = readStringInput("数据库用户名: ", "elake_api_gateway")
	config.DB.Password = readPasswordInput("数据库密码: ")
	config.DB.Name = readStringInput("数据库名称: ", "elake_api_gateway")
	config.DB.MaxOpenConnections = readIntInput("最大连接数: ", 1, 1000, 50)
	config.DB.MaxIdleConnections = readIntInput("最大空闲连接数: ", 1, 1000, 10)
	config.DB.ConnMaxLifetime = readIntInput("单连接最大存活时间(秒): ", 1, 86400, 3600)
	config.DB.ConnMaxIdleTime = readIntInput("空闲最大时间(秒): ", 1, 86400, 300)
	config.DB.ReadTimeout = readIntInput("读取超时时间(秒): ", 1, 300, 5)
	config.DB.WriteTimeout = readIntInput("写入超时时间(秒): ", 1, 300, 5)

	fmt.Println("")
	fmt.Println("【Redis配置】")
	config.Redis.Host = readStringInput("Redis主机: ", "localhost")
	config.Redis.Port = readIntInput("Redis端口: ", 1, 65535, 6379)
	config.Redis.Password = readPasswordInput("Redis密码: ")
	config.Redis.DB = readIntInput("库索引: ", 0, 15, 0)
	config.Redis.ProjectPrefix = readStringInput("项目前缀: ", "elake-api-gateway")
	config.Redis.PoolSize = readIntInput("连接池大小: ", 1, 1000, 100)
	config.Redis.MinIdleConnections = readIntInput("最小空闲连接数: ", 0, 100, 10)
	config.Redis.ConnectionMaxIdleTime = readIntInput("连接最大空闲时间(分钟): ", 1, 60, 15)
	config.Redis.ConnMaxLifetime = readIntInput("连接最大存活时间(分钟): ", 1, 120, 30)
	config.Redis.DialTimeout = readIntInput("连接超时时间(秒): ", 1, 60, 5)
	config.Redis.ReadTimeout = readIntInput("读取超时时间(秒): ", 1, 60, 5)
	config.Redis.WriteTimeout = readIntInput("写入超时时间(秒): ", 1, 60, 5)
	config.Redis.PoolTimeout = readIntInput("连接池超时时间(秒): ", 1, 60, 3)

	fmt.Println("")
	fmt.Println("【IPDB配置】")
	config.IPDB.DownloadURL = readStringInput("IPDB下载地址: ", "https://www.ip2location.com/download/?token={token}&file={file}")
	config.IPDB.FileName = readStringInput("IPDB文件名: ", "DB11LITEBINIPV6")
	config.IPDB.ZipEntryName = readStringInput("ZIP内BIN文件名: ", "IP2LOCATION-LITE-DB11.IPV6.BIN")
	config.IPDB.Token = readStringInput("IPDB令牌: ", "")
	config.IPDB.MaxDownloadAttempts = readIntInput("最大下载尝试次数: ", 1, 10, 5)

	return config, nil
}

func getDefaultConfig() *Config {
	return &Config{
		Gateway: GatewayConfig{
			ID:                   "1",
			IP:                   "127.0.0.1",
			Port:                 8080,
			EmailUsername:        []string{},
			DataPath:             "data",
			TempPath:             "tmp",
			MaxConcurrencyPerCPU: 100,
			RecalculateInterval:  10,
			MaxIdleConns:         512,
			MaxIdleConnsPerHost:  512,
			IdleConnTimeout:      90,
			UUIDPoolSize:         5000,
		},
		Logger: LoggerConfig{
			Output:         "both",
			Level:          "info",
			LogPath:        "logs/api-gateway.log",
			RequestLogPath: "logs/request.log",
			MaxSize:        100,
			MaxBackups:     10,
			MaxAge:         30,
			Compress:       true,
		},
		Health: HealthConfig{
			DBHealthCheckInterval:         10,
			DBHealthCheckFailThreshold:    3,
			DBHealthCheckOKThreshold:      1,
			RedisHealthCheckInterval:      10,
			RedisHealthCheckFailThreshold: 3,
			RedisHealthCheckOKThreshold:   1,
			IPDBHealthCheckInterval:       20,
			IPDBHealthCheckFailThreshold:  3,
			IPDBHealthCheckOKThreshold:    1,
		},
		DB: DBConfig{
			Host:               "localhost",
			Port:               3306,
			User:               "elake_api_gateway",
			Password:           "",
			Name:               "elake_api_gateway",
			MaxOpenConnections: 50,
			MaxIdleConnections: 10,
			ConnMaxLifetime:    3600,
			ConnMaxIdleTime:    300,
			ReadTimeout:        5,
			WriteTimeout:       5,
		},
		Redis: RedisConfig{
			Host:                  "localhost",
			Port:                  6379,
			Password:              "",
			DB:                    0,
			ProjectPrefix:         "elake-api-gateway",
			PoolSize:              100,
			MinIdleConnections:    10,
			ConnectionMaxIdleTime: 15,
			ConnMaxLifetime:       30,
			DialTimeout:           5,
			ReadTimeout:           5,
			WriteTimeout:          5,
			PoolTimeout:           3,
		},
		IPDB: IPDBConfig{
			DownloadURL:         "https://www.ip2location.com/download/?token={token}&file={file}",
			FileName:            "DB11LITEBINIPV6",
			ZipEntryName:        "IP2LOCATION-LITE-DB11.IPV6.BIN",
			Token:               "",
			MaxDownloadAttempts: 5,
		},
	}
}

func step3TestDBConnection(config *Config) error {
	fmt.Println("")
	fmt.Println("【步骤3】测试数据库连接")

	cfg := mysql.Config{
		User:                 config.DB.User,
		Passwd:               config.DB.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.DB.Host, config.DB.Port),
		DBName:               config.DB.Name,
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %v", err)
	}
	defer db.Close()

	fmt.Print("  正在连接数据库... ")
	if err := db.Ping(); err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}

	fmt.Println("✓ 成功")
	return nil
}

func step4TestRedisConnection(config *Config) error {
	fmt.Println("")
	fmt.Println("【步骤4】测试Redis连接")

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	fmt.Print("  正在连接Redis... ")
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}

	fmt.Println("✓ 成功")
	return nil
}

func step5InitDatabase(config *Config) error {
	fmt.Println("")
	fmt.Println("【步骤5】初始化数据库")

	cfg := mysql.Config{
		User:                 config.DB.User,
		Passwd:               config.DB.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.DB.Host, config.DB.Port),
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %v", err)
	}
	defer db.Close()

	fmt.Print("  正在创建数据库... ")
	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", config.DB.Name)
	if _, err := db.Exec(createDB); err != nil {
		return fmt.Errorf("创建数据库失败: %v", err)
	}
	fmt.Println("✓ 成功")

	cfg.DBName = config.DB.Name
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Print("  正在创建数据库表... ")
	sqlContent, err := readSQLFile("mariaDB.sql")
	if err != nil {
		return fmt.Errorf("读取SQL文件失败: %v", err)
	}

	statements := splitSQLStatements(sqlContent)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if strings.Contains(strings.ToUpper(stmt), "DELIMITER") {
			continue
		}
		if strings.Contains(strings.ToUpper(stmt), "BEGIN") && strings.Contains(strings.ToUpper(stmt), "END") {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("执行SQL语句失败 [%s]: %v", truncate(stmt, 50), err)
		}
	}
	fmt.Println("✓ 成功")

	return nil
}

func step6InitGatewayConfig(config *Config) error {
	fmt.Println("")
	fmt.Println("【步骤6】初始化网关配置")
	fmt.Println("请选择网关配置创建方式:")
	fmt.Println("  1. 关键配置模式（仅配置email和auth.master-key）")
	fmt.Println("  2. 全自定义模式（所有配置项都询问）")

	choice := readIntInput("请输入选择 (1/2): ", 1, 2, 1)

	cfg := mysql.Config{
		User:                 config.DB.User,
		Passwd:               config.DB.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", config.DB.Host, config.DB.Port),
		DBName:               config.DB.Name,
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("创建数据库连接失败: %v", err)
	}
	defer db.Close()

	if choice == 1 {
		fmt.Println("")
		fmt.Println("--- 关键配置模式 ---")

		fmt.Println("请输入Email配置:")
		emailHost := readStringInput("SMTP主机地址 (默认: smtp.qq.com): ", "smtp.qq.com")
		emailPort := readIntInput("SMTP端口 (默认: 587): ", 1, 65535, 587)
		emailUsername := readStringInput("SMTP用户名: ", "")
		emailPassword := readPasswordInput("SMTP密码: ")

		fmt.Println("")
		fmt.Println("请输入认证配置:")
		masterKey := readStringInput("加密密钥 (用于加密和解密请求体): ", "")

		configs := []struct {
			id          string
			groupName   string
			typeName    string
			value       string
			title       string
			description string
		}{
			{"email.host", "email", "string", emailHost, "SMTP主机地址", "用于发送邮件的SMTP主机地址"},
			{"email.port", "email", "int", strconv.Itoa(emailPort), "SMTP主机端口", "用于发送邮件的SMTP主机端口"},
			{"email.username", "email", "string", emailUsername, "SMTP用户名", "用于发送邮件的SMTP用户名/邮箱"},
			{"email.password", "email", "string", emailPassword, "SMTP密码", "用于发送邮件的SMTP密码"},
			{"email.timeout", "email", "int", "10", "SMTP超时时间(秒)", "用于发送邮件的SMTP超时时间"},
			{"email.max-retry", "email", "int", "3", "最大重试次数", "用于发送邮件的最大重试次数"},
			{"auth.master-key", "auth", "string", masterKey, "加密密钥", "用于加密和解密请求体的密钥"},
			{"auth.timestamp-window", "auth", "int", "5", "时间戳窗口(秒)", "默认5秒, 用于校验请求时间戳是否在5秒内"},
			{"auth.nonce-window", "auth", "int", "2", "nonce唯一性校验窗口(次)", "默认2次, 用于校验请求nonce是否在2次内"},
			{"auth.nonce-window-hour", "auth", "int", "24", "nonce唯一性校验窗口(小时)", "默认24小时, 用于校验请求nonce是否在24小时内"},
			{"auth.qps-limit", "auth", "int", "5", "每秒最大请求数", "默认5次, 用于限制每秒请求数"},
			{"auth.qpm-limit", "auth", "int", "150", "每分钟最大请求数", "默认150次, 用于限制每分钟请求数"},
			{"auth.ip-black-list", "auth", "string", "", "全局IP黑名单", "用于限制访问的IP地址"},
			{"auth.country-black-list", "auth", "string", "", "全局国家黑名单", "用于限制访问的国家"},
			{"gateway.local-cache-expire", "gateway", "int", "10", "本地缓存过期时间(分钟)", "默认10分钟"},
			{"gateway.redis-cache-expire", "gateway", "int", "60", "Redis缓存过期时间(分钟)", "默认60分钟"},
			{"gateway.api-root", "gateway", "string", "/_gateway/api", "网关API根路由", "默认/_gateway/api"},
			{"gateway.node-timeout", "gateway", "int", "3", "节点超时时间(秒)", "默认3秒"},
			{"gateway.total-timeout", "gateway", "int", "10", "总超时时间(秒)", "默认10秒"},
		}

		fmt.Print("  正在写入网关配置... ")
		for _, c := range configs {
			query := `INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
                      VALUES (?, ?, ?, ?, ?, ?, 1)
                      ON DUPLICATE KEY UPDATE value = ?, version = version + 1`
			_, err := db.Exec(query, c.id, c.groupName, c.typeName, c.value, c.title, c.description, c.value)
			if err != nil {
				return fmt.Errorf("写入配置失败 [%s]: %v", c.id, err)
			}
		}
		fmt.Println("✓ 成功")
	} else {
		fmt.Println("")
		fmt.Println("--- 全自定义模式 ---")

		fmt.Println("请输入网关配置:")
		localCacheExpire := readIntInput("本地缓存过期时间(分钟): ", 1, 1440, 10)
		redisCacheExpire := readIntInput("Redis缓存过期时间(分钟): ", 1, 1440, 60)
		apiRoot := readStringInput("网关API根路由: ", "/_gateway/api")
		nodeTimeout := readIntInput("节点超时时间(秒): ", 1, 60, 3)
		totalTimeout := readIntInput("总超时时间(秒): ", 1, 300, 10)

		fmt.Println("")
		fmt.Println("请输入Email配置:")
		emailHost := readStringInput("SMTP主机地址: ", "smtp.qq.com")
		emailPort := readIntInput("SMTP端口: ", 1, 65535, 587)
		emailUsername := readStringInput("SMTP用户名: ", "")
		emailPassword := readPasswordInput("SMTP密码: ")
		emailTimeout := readIntInput("SMTP超时时间(秒): ", 1, 60, 10)
		emailMaxRetry := readIntInput("最大重试次数: ", 1, 10, 3)

		fmt.Println("")
		fmt.Println("请输入认证配置:")
		masterKey := readStringInput("加密密钥: ", "")
		timestampWindow := readIntInput("时间戳窗口(秒): ", 1, 60, 5)
		nonceWindow := readIntInput("nonce唯一性校验窗口(次): ", 1, 10, 2)
		nonceWindowHour := readIntInput("nonce唯一性校验窗口(小时): ", 1, 24, 24)
		qpsLimit := readIntInput("每秒最大请求数: ", 1, 1000, 5)
		qpmLimit := readIntInput("每分钟最大请求数: ", 1, 60000, 150)
		ipBlackList := readStringInput("全局IP黑名单(每行一个IP): ", "")
		countryBlackList := readStringInput("全局国家黑名单(每行一个国家): ", "")

		configs := []struct {
			id          string
			groupName   string
			typeName    string
			value       string
			title       string
			description string
		}{
			{"gateway.local-cache-expire", "gateway", "int", strconv.Itoa(localCacheExpire), "本地缓存过期时间(分钟)", ""},
			{"gateway.redis-cache-expire", "gateway", "int", strconv.Itoa(redisCacheExpire), "Redis缓存过期时间(分钟)", ""},
			{"gateway.api-root", "gateway", "string", apiRoot, "网关API根路由", ""},
			{"gateway.node-timeout", "gateway", "int", strconv.Itoa(nodeTimeout), "节点超时时间(秒)", ""},
			{"gateway.total-timeout", "gateway", "int", strconv.Itoa(totalTimeout), "总超时时间(秒)", ""},
			{"email.host", "email", "string", emailHost, "SMTP主机地址", ""},
			{"email.port", "email", "int", strconv.Itoa(emailPort), "SMTP主机端口", ""},
			{"email.username", "email", "string", emailUsername, "SMTP用户名", ""},
			{"email.password", "email", "string", emailPassword, "SMTP密码", ""},
			{"email.timeout", "email", "int", strconv.Itoa(emailTimeout), "SMTP超时时间(秒)", ""},
			{"email.max-retry", "email", "int", strconv.Itoa(emailMaxRetry), "最大重试次数", ""},
			{"auth.master-key", "auth", "string", masterKey, "加密密钥", ""},
			{"auth.timestamp-window", "auth", "int", strconv.Itoa(timestampWindow), "时间戳窗口(秒)", ""},
			{"auth.nonce-window", "auth", "int", strconv.Itoa(nonceWindow), "nonce唯一性校验窗口(次)", ""},
			{"auth.nonce-window-hour", "auth", "int", strconv.Itoa(nonceWindowHour), "nonce唯一性校验窗口(小时)", ""},
			{"auth.qps-limit", "auth", "int", strconv.Itoa(qpsLimit), "每秒最大请求数", ""},
			{"auth.qpm-limit", "auth", "int", strconv.Itoa(qpmLimit), "每分钟最大请求数", ""},
			{"auth.ip-black-list", "auth", "string", ipBlackList, "全局IP黑名单", ""},
			{"auth.country-black-list", "auth", "string", countryBlackList, "全局国家黑名单", ""},
		}

		fmt.Print("  正在写入网关配置... ")
		for _, c := range configs {
			query := `INSERT INTO api_gateway_config (id, group_name, type, value, title, description, version)
                      VALUES (?, ?, ?, ?, ?, ?, 1)
                      ON DUPLICATE KEY UPDATE value = ?, version = version + 1`
			_, err := db.Exec(query, c.id, c.groupName, c.typeName, c.value, c.title, c.description, c.value)
			if err != nil {
				return fmt.Errorf("写入配置失败 [%s]: %v", c.id, err)
			}
		}
		fmt.Println("✓ 成功")
	}

	return nil
}

func readStringInput(prompt string, defaultValue string) string {
	fmt.Print(prompt)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return defaultValue
	}
	return input
}

func readIntInput(prompt string, min, max, defaultValue int) int {
	for {
		fmt.Print(prompt)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return defaultValue
		}
		val, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("  请输入有效数字\n")
			continue
		}
		if val < min || val > max {
			fmt.Printf("  请输入 %d 到 %d 之间的数字\n", min, max)
			continue
		}
		return val
	}
}

func readPasswordInput(prompt string) string {
	fmt.Print(prompt)
	scanner.Scan()
	return scanner.Text()
}

func readBoolInput(prompt string) bool {
	for {
		fmt.Print(prompt)
		scanner.Scan()
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if input == "" || input == "true" {
			return true
		}
		if input == "false" {
			return false
		}
		fmt.Println("  请输入 true 或 false")
	}
}

func readSQLFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func splitSQLStatements(sql string) []string {
	return strings.Split(sql, ";")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
