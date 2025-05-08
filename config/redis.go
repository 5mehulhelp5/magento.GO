package config

import (
	"context"
	"os"
	"github.com/redis/go-redis/v9"
)

// RedisClient is a global Redis client instance
var RedisClient *redis.Client
//Accessed as config.RedisClient in other files

func InitRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		RedisClient = nil
		return
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASS"),
		DB:       0,
	})
}

func RedisCtx() context.Context {
	return context.Background()
}