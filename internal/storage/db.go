package storage

import (
	"elake-api-gateway/internal/config"
	"elake-api-gateway/internal/logger"
	"elake-api-gateway/internal/service/healthManager"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// InitDB 初始化数据库连接
func InitDB() *sqlx.DB {
	cfg := config.Get()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
	)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		logger.Log.Fatal("DB 连接失败:", zap.Error(err))
	}
	db.SetMaxOpenConns(cfg.DB.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConnections)
	db.SetConnMaxLifetime(time.Duration(cfg.DB.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.DB.ConnMaxIdleTime) * time.Second)
	logger.Log.Info("DB 连接成功")
	healthManager.Global().Register(
		"DB",
		cfg.Health.DBHealthCheckFailThreshold,
		cfg.Health.DBHealthCheckOKThreshold,
	)
	// 健康检查
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.Health.DBHealthCheckInterval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			err := db.Ping()
			healthManager.Global().Report("DB", err)
		}
	}()
	return db
}
