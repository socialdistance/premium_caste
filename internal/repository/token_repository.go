package repository

import (
	redisapp "premium_caste/internal/storage/redis"

	"github.com/redis/go-redis/v9"

	"context"
	"time"
)

type RedisTokenRepo struct {
	Client *redisapp.Client
}

func NewRedisTokenRepo(client *redisapp.Client) *RedisTokenRepo {
	return &RedisTokenRepo{Client: client}
}

func (r *RedisTokenRepo) SaveRefreshToken(ctx context.Context, userID, token string, exp time.Duration) error {
	return r.Client.Set(ctx, refreshTokenKey(userID, token), "1", exp).Err()
}

func (r *RedisTokenRepo) GetRefreshToken(ctx context.Context, userID, token string) (bool, error) {
	val, err := r.Client.Get(ctx, refreshTokenKey(userID, token)).Result()
	if err == redis.Nil {
		return false, nil
	}
	return val == "1", err
}

func (r *RedisTokenRepo) DeleteRefreshToken(ctx context.Context, userID, token string) error {
	return r.Client.Del(ctx, refreshTokenKey(userID, token)).Err()
}

func (r *RedisTokenRepo) DeleteAllUserTokens(ctx context.Context, userID string) error {
	keys, err := r.Client.Keys(ctx, refreshTokenKey(userID, "*")).Result()
	if err != nil {
		return err
	}
	return r.Client.Del(ctx, keys...).Err()
}

func refreshTokenKey(userID, token string) string {
	return "refresh:" + userID + ":" + token
}
