package redisx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/visonlv/go-vkit/config"
	"github.com/visonlv/go-vkit/logger"
)

type RedisClient struct {
	c *redis.Client
}

type RedisKey struct {
	Code   string
	Expire time.Duration
}

func NewDefault() *RedisClient {
	addr := config.GetString("database.redis.addr")
	password := config.GetString("database.redis.password")
	db := config.GetInt("database.redis.db")
	if addr == "" || password == "" {
		logger.Errorf("[redis] addr:%s password:%s db:%d has empty", addr, password, db)
		panic("pamar error")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		logger.Errorf("[redis] ping fail err:%s", err)
		panic(err)
	}

	logger.Infof("[redis] addr:%s password:%s db:%d init success", addr, password, db)
	return &RedisClient{c: rdb}
}

func (c *RedisClient) Set(key *RedisKey, sub string, value interface{}) error {
	fullKey := key.Code + ":" + sub
	return c.c.Set(context.Background(), fullKey, value, key.Expire).Err()
}

func (c *RedisClient) SetJson(key *RedisKey, sub string, value interface{}) error {
	fullKey := key.Code + ":" + sub
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.c.Set(context.Background(), fullKey, string(bytes), key.Expire).Err()
}

func (c *RedisClient) GetString(key *RedisKey, sub string) (string, error) {
	fullKey := key.Code + ":" + sub
	return c.c.Get(context.Background(), fullKey).Result()
}

func (c *RedisClient) GetInt64(key *RedisKey, sub string) (int64, error) {
	fullKey := key.Code + ":" + sub
	return c.c.Get(context.Background(), fullKey).Int64()
}

func (c *RedisClient) GetInt(key *RedisKey, sub string) (int, error) {
	return c.c.Get(context.Background(), key.Code).Int()
}

func (c *RedisClient) GetJson(key *RedisKey, sub string, to interface{}) error {
	fullKey := key.Code + ":" + sub
	s, err := c.c.Get(context.Background(), fullKey).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), to)
}

func (c *RedisClient) GetSet(key *RedisKey, sub string, value string) (string, error) {
	fullKey := key.Code + ":" + sub
	return c.c.GetSet(context.Background(), fullKey, value).Result()
}

func (c *RedisClient) IncrBy(key *RedisKey, sub string, value int64) (int64, error) {
	fullKey := key.Code + ":" + sub
	return c.c.IncrBy(context.Background(), fullKey, value).Result()
}
