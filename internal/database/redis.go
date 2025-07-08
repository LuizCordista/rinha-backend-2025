package database

import (
	"context"
	"github.com/go-redis/redis/v8"
	"os"
)

var Rdb *redis.Client
var RedisCtx = context.Background()

func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
}
