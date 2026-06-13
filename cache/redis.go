package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gasoid/merge-bot/v3/config"
	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/redis/go-redis/v9"
)

var (
	redisUrl         string
	ErrRedisUrlEmpty = errors.New("redis-url is not provided")
	casScript        = redis.NewScript(`
	local val = redis.call('JSON.NUMINCRBY', KEYS[1], '$["' .. ARGV[1] .. '"]' , ARGV[2])
	local parsed = cjson.decode(val)

    if parsed[1] < 0 then
	    redis.call('JSON.NUMINCRBY', KEYS[1], '$["' .. ARGV[1] .. '"]' , -1 * tonumber(ARGV[2]))
		return 0
	end
	return 1
`)
)

const (
	lockTTL = 30 * time.Minute
)

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
	if _, err := r.client.JSONSet(context.TODO(), key, "$", v).Result(); err != nil {
		logger.Debug("redis error", "err", err, "key", key, "v", v)
		return &CacheError{Operation: "JsonSet", Err: err}
	}

	return nil
}

func (r *RedisCache) ExtendTTL(key string, ttl time.Duration) error {
	if _, err := r.client.Expire(context.TODO(), key, ttl).Result(); err != nil {
		if err != redis.Nil {
			return &CacheError{Operation: "Expire", Err: err}
		}
	}

	return nil
}

func (r *RedisCache) JsonAdd(key, item string, v int) error {
	if v < 0 {
		v = 0
	}

	if _, err := r.client.JSONSetWithArgs(context.TODO(), key, "$[\""+escapeChars(item)+"\"]", v, &redis.JSONSetArgsOptions{Mode: "NX"}).Result(); err != nil {
		return &CacheError{Operation: "JsonAdd", Err: err}
	}

	return nil
}

func (r *RedisCache) JsonGet(key string) ([]int64, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, &CacheError{Operation: "JsonGet", Err: err}
	}

	if val == "[]" {
		return nil, nil
	}

	result := [][]int64{}

	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("%w: expected []int64 for key %s", ErrWrongType, key)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func (r *RedisCache) JsonGetMap(key string) (map[string]int, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, &CacheError{Operation: "JsonGet", Err: err}
	}

	if val == "[]" {
		return nil, nil
	}

	result := []map[string]int{}

	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("%w: expected map[string]int for key %s", ErrWrongType, key)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result[0], nil
}

func escapeChars(data string) string {
	data = strings.ReplaceAll(data, "[", "\\[")
	data = strings.ReplaceAll(data, "]", "\\]")
	return strings.ReplaceAll(data, "\"", "\\\"")
}

func (r *RedisCache) JsonExists(key, item string) (bool, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$[\""+escapeChars(item)+"\"]").Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, &CacheError{Operation: "JsonExists", Err: err}
	}

	if val == "[]" {
		return false, nil
	}

	return true, nil
}

func (r *RedisCache) JsonIncr(key, item string, v int) (bool, error) {
	result, err := casScript.Run(context.TODO(), r.client, []string{key}, escapeChars(item), v).Int()
	if err != nil {
		return false, &CacheError{Operation: "JsonIncr", Err: err}
	}

	if result == 0 {
		return false, nil
	}

	return true, nil
}

func (r *RedisCache) TryAcquireLock(key string) bool {
	_, err := r.client.SetArgs(context.TODO(), key, true, redis.SetArgs{Mode: "NX", TTL: lockTTL}).Result()
	if err == nil {
		return true
	}

	if err == redis.Nil {
		return false
	}

	logger.Info("can't aquire a lock", "error", &CacheError{Operation: "TryAcquireLock", Err: err})
	return false
}

func (r *RedisCache) Unlock(key string) {
	if _, err := r.client.Del(context.TODO(), key).Result(); err != nil {
		logger.Info("can't delete a lock", "error", &CacheError{Operation: "Unlock", Err: err})
		return
	}
}

func (r *RedisCache) IsHealthy() bool {
	err := r.client.Ping(context.TODO()).Err()
	if err != nil {
		logger.Error("failed to connect to redis", "err", err)
		return false
	}

	return true
}

var (
	_ Cache = (*RedisCache)(nil)
)
