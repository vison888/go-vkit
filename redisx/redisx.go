package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vison888/go-vkit/logger"
)

// redis.Nil
type RedisClient struct {
	c *redis.Client
}

type RedisKey struct {
	Code   string
	Expire time.Duration
}

func NewClient(addr, password string, db int) (*RedisClient, error) {
	if addr == "" {
		logger.Errorf("[redis] NewClient fail:pamar error addr:%s password:%s db:%d ", addr, password, db)
		return nil, errors.New("pamar error")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		logger.Errorf("[redis] NewClient fail:%s addr:%s password:%s db:%d ", err.Error(), addr, password, db)
		return nil, err
	}

	logger.Infof("[redis] NewClient success addr:%s password:%s db:%d ", addr, password, db)
	return &RedisClient{c: rdb}, nil
}

func GetFullKey(key *RedisKey, sub string) string {
	if sub == "" {
		return key.Code
	}
	fullKey := key.Code + ":" + sub
	return fullKey
}

func (c *RedisClient) Set(key *RedisKey, sub string, value any) error {
	fullKey := GetFullKey(key, sub)
	return c.c.Set(context.Background(), fullKey, value, key.Expire).Err()
}

func (c *RedisClient) Del(key *RedisKey, sub string) error {
	fullKey := GetFullKey(key, sub)
	return c.c.Del(context.Background(), fullKey).Err()
}

func (c *RedisClient) SetJson(key *RedisKey, sub string, value any) error {
	fullKey := GetFullKey(key, sub)
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.c.Set(context.Background(), fullKey, string(bytes), key.Expire).Err()
}

func (c *RedisClient) GetString(key *RedisKey, sub string) (string, error) {
	fullKey := GetFullKey(key, sub)
	return c.c.Get(context.Background(), fullKey).Result()
}

func (c *RedisClient) GetInt64(key *RedisKey, sub string) (int64, error) {
	fullKey := GetFullKey(key, sub)
	return c.c.Get(context.Background(), fullKey).Int64()
}

func (c *RedisClient) GetInt(key *RedisKey, sub string) (int, error) {
	return c.c.Get(context.Background(), key.Code).Int()
}

func (c *RedisClient) GetJson(key *RedisKey, sub string, to any) error {
	fullKey := GetFullKey(key, sub)
	s, err := c.c.Get(context.Background(), fullKey).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), to)
}

func (c *RedisClient) GetSet(key *RedisKey, sub string, value string) (string, error) {
	fullKey := GetFullKey(key, sub)
	return c.c.GetSet(context.Background(), fullKey, value).Result()
}

func (c *RedisClient) IncrBy(key *RedisKey, sub string, value int64) (int64, error) {
	fullKey := GetFullKey(key, sub)
	return c.c.IncrBy(context.Background(), fullKey, value).Result()
}

func (c *RedisClient) GetHashJson(key *RedisKey, sub string, hash string, to any) error {
	fullKey := GetFullKey(key, sub)
	s, err := c.c.HGet(context.Background(), fullKey, hash).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), to)
}

func (c *RedisClient) GetHashAllJson(key *RedisKey, sub string) (map[string]string, error) {
	fullKey := GetFullKey(key, sub)
	s, err := c.c.HGetAll(context.Background(), fullKey).Result()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (c *RedisClient) SetHashJson(key *RedisKey, sub string, hash string, value any) error {
	fullKey := GetFullKey(key, sub)
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	intCmd := c.c.HSet(context.Background(), fullKey, hash, string(bytes)).Err()
	if intCmd != nil {
		return intCmd
	}

	if key.Expire > 0 {
		c.c.Expire(context.Background(), fullKey, key.Expire)
	}

	return nil
}

func (c *RedisClient) HIncrBy(key *RedisKey, sub string, hash string, incr int64) error {
	fullKey := GetFullKey(key, sub)
	intCmd := c.c.HIncrBy(context.Background(), fullKey, hash, incr).Err()
	if intCmd != nil {
		return intCmd
	}

	if key.Expire > 0 {
		c.c.Expire(context.Background(), fullKey, key.Expire)
	}

	return nil
}
