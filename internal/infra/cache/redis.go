package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheService struct {
	client *redis.Client
}

func NewRedisCacheService(addr, password string, db int) *RedisCacheService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCacheService{client: rdb}
}

func (s *RedisCacheService) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get key from redis: %w", err)
	}
	return val, nil
}

func (s *RedisCacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	err := s.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key in redis: %w", err)
	}
	return nil
}

func (s *RedisCacheService) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisCacheService) Close() error {
	return s.client.Close()
}
