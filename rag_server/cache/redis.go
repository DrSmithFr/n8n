package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
)

func InitCache() (*redis.Client, context.Context) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://redis:6379/0"
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opts)
	var ctx = context.Background()

	return rdb, ctx
}
