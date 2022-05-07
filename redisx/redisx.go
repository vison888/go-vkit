package redisx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/visonlv/go-vkit/config"
	"github.com/visonlv/go-vkit/logger"
	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	c *redis.Client
}

func NewDefault() *RedisClient {
	addr := config.GetString("database.redis.addr")
	password := config.GetString("database.redis.password")
	db := config.GetInt("database.redis.db")
	if addr == "" || password == "" {
		logger.Errorf("[redis] addr:%s password:%s db:%d has empty", addr, password, db)
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	defaultClient := &RedisClient{c: rdb}

	logger.Errorf("[redis] addr:%s password:%s db:%d init success", addr, password, db)
	return defaultClient
}

func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.c.Set(ctx, key, value, expiration).Err()
}

func (c *RedisClient) SetJson(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.c.Set(ctx, key, string(bytes), expiration).Err()
}

func (c *RedisClient) GetString(ctx context.Context, key string) (string, error) {
	return c.c.Get(ctx, key).Result()
}

func (c *RedisClient) GetInt64(ctx context.Context, key string) (int64, error) {
	return c.c.Get(ctx, key).Int64()
}

func (c *RedisClient) GetInt(ctx context.Context, key string) (int, error) {
	return c.c.Get(ctx, key).Int()
}

func (c *RedisClient) GetJson(ctx context.Context, key string, to interface{}) error {
	s, err := c.c.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), to)
}

func (c *RedisClient) GetSet(ctx context.Context, key string, value string) (string, error) {
	return c.c.GetSet(ctx, key, value).Result()
}

func (c *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.c.IncrBy(ctx, key, value).Result()
}
