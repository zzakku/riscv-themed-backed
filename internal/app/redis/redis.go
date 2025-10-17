package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const servicePrefix = "r-vBackend."

type Client struct {
	client *redis.Client
}

func New(ctx context.Context, rHost string, rPort int, rPswd string, rDT time.Duration, rRT time.Duration) (*Client, error) {

	redisClient := redis.NewClient(&redis.Options{
		Addr:        rHost + ":" + strconv.Itoa(rPort),
		Password:    rPswd,
		DB:          0,
		DialTimeout: rDT,
		ReadTimeout: rRT,
	})
	// Проверяем подключение
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("cant ping redis: %w", err)
	}

	return &Client{
		client: redisClient,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
