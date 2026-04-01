package app

import (
	"elake-api-gateway/internal/repository"
)

// App 应用
type App struct {
	DB    *repository.DBManager
	Redis *repository.RedisManager
	IPDB  *repository.IPDBManager
}
