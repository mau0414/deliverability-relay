package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func NewRedis() *Redis {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return &Redis{
		client: client,
	}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}