package storage

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

func NewClient(addr, password string, db int) *Client {
	return &Client{
		Client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

func (c *Client) Close() {
	c.Conn().Close()
}
