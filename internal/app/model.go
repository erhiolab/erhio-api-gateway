package app

import (
	"elake-api-gateway/internal/repository"
)

// App 应用
type App struct {
	Redis *repository.RedisManager
}
