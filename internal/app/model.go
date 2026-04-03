package app

import (
	"elake-api-gateway/internal/repository"
	"elake-api-gateway/internal/storage"
)

// App 应用
type App struct {
	DB         *repository.DBManager
	Redis      *repository.RedisManager
	LocalCache *storage.LocalCache
	IPDB       *repository.IPDBManager
}
