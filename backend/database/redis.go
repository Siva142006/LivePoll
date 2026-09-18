package database

import (
	"context"
	"fmt"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		if strings.Contains(redisURL, "://") {
			return nil, fmt.Errorf("redis url parse: %w", err)
		}
		opts = &redis.Options{Addr: redisURL}
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}
