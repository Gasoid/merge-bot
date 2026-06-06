package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gasoid/merge-bot/config"
	"github.com/redis/go-redis/v9"
)

var (
	redisUrl         string
	ErrRedisUrlEmpty = errors.New("redis-url is not provided")
	casScript        = redis.NewScript(`
	local val = redis.call('JSON.NUMINCRBY', KEYS[1], '$.' .. ARGV[1] , ARGV[2])
	local parsed = cjson.decode(val)

    if parsed[1] < 0 then
	    redis.call('JSON.NUMINCRBY', KEYS[1], '$.' .. ARGV[1] , -1 * tonumber(ARGV[2]))
		return 0
	end
	return 1
`)
)

// const (
// 	ttl = 30 * 24 * time.Hour
// )

func init() {
	config.StringVar(&redisUrl, "redis-url", "", "redis url redis://<user>:<pass>@localhost:6379/<db> (also via REDIS_URL)")
}

type RedisCache struct {
	client *redis.Client
}

func (r *RedisCache) Connect() error {
	if r.client != nil {
		return nil
	}

	if redisUrl == "" {
		return ErrRedisUrlEmpty
	}

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return &CacheError{Operation: "Connect", Err: err}
	}

	r.client = redis.NewClient(opt)
	return nil
}

func (r *RedisCache) JsonSet(key string, v any) error {
	if _, err := r.client.JSONSetWithArgs(context.TODO(), key, "$", v, &redis.JSONSetArgsOptions{Mode: "NX"}).Result(); err != nil {
		return &CacheError{Operation: "JSONSetWithArgs", Err: err}
	}

	return nil
}

func (r *RedisCache) JsonAdd(key, item string, v int) error {
	if v < 0 {
		v = 0
	}

	if _, err := r.client.JSONSetWithArgs(context.TODO(), key, "$."+item, v, &redis.JSONSetArgsOptions{Mode: "NX"}).Result(); err != nil {
		return &CacheError{Operation: "JSONSetWithArgs", Err: err}
	}

	return nil
}

func (r *RedisCache) JsonGet(key string) (any, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", &CacheError{Operation: "JsonGet", Err: err}
	}

	if val == "[]" {
		return nil, nil
	}

	result := []any{}

	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("json data is invalid %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *RedisCache) JsonExists(key, item string) (bool, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$."+item).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, &CacheError{Operation: "JsonGet", Err: err}
	}

	if val == "[]" {
		return false, nil
	}

	return true, nil
}

func (r *RedisCache) JsonIncr(key, item string, v int) (bool, error) {
	result, err := casScript.Run(context.TODO(), r.client, []string{key}, item, v).Int()
	if err != nil {
		return false, &CacheError{Operation: "casScript", Err: err}
	}

	if result == 0 {
		return false, nil
	}

	return true, nil
}

var (
	_ Cache = (*RedisCache)(nil)
)
