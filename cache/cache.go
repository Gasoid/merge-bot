package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

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

const (
	ttl = 30 * 24 * time.Hour
)

func init() {
	config.StringVar(&redisUrl, "redis-url", "", "redis url redis://<user>:<pass>@localhost:6379/<db> (also via REDIS_URL)")
}

type Cache interface {
	Set(key, val string) error
	Get(key string) (string, error)
	JsonSet(key string, v string) error
	JsonExists(key, item string) (bool, error)
	JsonAdd(key, item string, v int) error
	JsonIncr(key string, item string, v int) (bool, error)
	JsonGet(key string) (string, error)
	Connect() error
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

func (r *RedisCache) Set(key, val string) error {
	if _, err := r.client.SetNX(context.TODO(), key, val, ttl).Result(); err != nil {
		return &CacheError{Operation: "SetNX", Err: err}
	}

	return nil
}

func (r *RedisCache) Get(key string) (string, error) {
	val, err := r.client.Get(context.TODO(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}

		return "", &CacheError{Operation: "Get", Err: err}
	}

	return val, nil
}

func (r *RedisCache) JsonSet(key, v string) error {
	if _, err := r.client.JSONSetWithArgs(context.TODO(), key, "$", v, &redis.JSONSetArgsOptions{Mode: "NX"}).Result(); err != nil {
		return &CacheError{Operation: "JSONSetWithArgs", Err: err}
	}

	// if _, err := r.client.Expire(context.TODO(), key, ttl).Result(); err != nil {
	// 	return &CacheError{Operation: "Expire", Err: err}
	// }

	return nil
}

func (r *RedisCache) JsonAdd(key, item string, v int) error {
	if _, err := r.client.JSONSetWithArgs(context.TODO(), key, "$."+item, v, &redis.JSONSetArgsOptions{Mode: "NX"}).Result(); err != nil {
		return &CacheError{Operation: "JSONSetWithArgs", Err: err}
	}

	// if _, err := r.client.Expire(context.TODO(), key, ttl).Result(); err != nil {
	// 	return &CacheError{Operation: "Expire", Err: err}
	// }

	return nil
}

func (r *RedisCache) JsonGet(key string) (string, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", &CacheError{Operation: "JsonGet", Err: err}
	}

	return val, nil
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

type CacheError struct {
	Operation string
	Err       error
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache failed to execute operation: %s because of error: %s", e.Operation, e.Err)
}

var (
	_ Cache = (*RedisCache)(nil)
)
