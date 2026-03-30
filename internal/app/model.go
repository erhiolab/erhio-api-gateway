package app

import (
	"github.com/redis/go-redis/v9"
)

// App 应用
type App struct {
	Redis *redis.Client
}
