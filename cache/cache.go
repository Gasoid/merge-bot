package cache

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/gasoid/merge-bot/config"
	"github.com/redis/go-redis/v9"
)

var (
	redisUrl string
)

const (
	ttl = 24 * time.Hour
)

func init() {
	config.StringVar(&redisUrl, "redis-url", "", "redis url redis://<user>:<pass>@localhost:6379/<db> (also via REDIS_URL)")
}

type Cache interface {
	Set(key, val string) error
	Get(key string) (string, error)
	JsonSet(key string, v string) error
	JsonASet(key string, v string, oldVal, newVal int) (bool, error)
	JsonGet(key string) (string, error)
	Connect() error
}

type MemCache struct {
	items        []string
	keys         map[string]string
	memcacheLock sync.RWMutex
}

func (m *MemCache) JsonSet(key string, v string) error {
	return m.Set(key, v)
}

func (m *MemCache) JsonASet(key string, v string, oldVal, newVal int) (bool, error) {
	return true, m.Set(key, v)
}

func (m *MemCache) JsonGet(key string) (string, error) {
	return m.Get(key)
}

func (m *MemCache) Set(key, val string) error {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	m.keys[key] = val
	return nil
}

func (m *MemCache) Get(key string) (string, error) {
	if val, ok := m.keys[key]; ok {
		return val, nil
	}

	return "", nil
}

func (m *MemCache) Connect() error {
	if m.keys == nil {
		m.keys = make(map[string]string)
	}

	return nil
}

type RedisCache struct {
	client  *redis.Client
	lockVal int
	lock    sync.RWMutex
}

func (r *RedisCache) Connect() error {
	if r.client != nil {
		return nil
	}

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return &CacheError{Operation: "Connect", Err: err}
	}

	// opt.MinRetryBackoff = 10 * time.Millisecond
	// opt.MaxRetryBackoff = 100 * time.Millisecond
	opt.MaxRetries = 5
	opt.DialTimeout = 10 * time.Second
	opt.ReadTimeout = 5 * time.Second
	opt.WriteTimeout = 5 * time.Second

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

	if _, err := r.client.Expire(context.TODO(), key, time.Duration(24)*time.Hour).Result(); err != nil {
		return &CacheError{Operation: "Expire", Err: err}
	}

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

func (r *RedisCache) JsonGetItem(key, item string) (string, error) {
	val, err := r.client.JSONGet(context.TODO(), key, "$."+item).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", &CacheError{Operation: "JsonGet", Err: err}
	}

	return val, nil
}

var casScript = redis.NewScript(`
    local val = redis.call('JSON.GET', KEYS[1], '$.' .. ARGV[1])
    local parsed = cjson.decode(val)
    if parsed[1] == tonumber(ARGV[2]) then
        redis.call('JSON.SET', KEYS[1], '$.' .. ARGV[1] , ARGV[3])
        return 1
    end
    return 0
`)

// atomic CAS
func (r *RedisCache) JsonASet(key, item string, oldVal, newVal int) (bool, error) {
	result, err := casScript.Run(context.TODO(), r.client, []string{key}, item, oldVal, newVal).Int()
	if err != nil {
		return false, &CacheError{Operation: "casScript", Err: err}
	}

	if result == 0 {
		return false, nil
	}

	return true, nil
}

func (r *RedisCache) lockName(key string) string {
	return fmt.Sprintf("%s:lock", key)
}

// func (r *RedisCache) hashName(key string) string {
// 	return fmt.Sprintf("%s:unique", key)
// }

func (r *RedisCache) Lock(key string) error {
	key = r.lockName(key)

	lockVal := rand.Intn(1000)
	for {
		done, err := r.client.SetNX(context.TODO(), key, lockVal, time.Second*300).Result()
		if err != nil {
			return &CacheError{Operation: "SetNX", Err: err}
		}

		if done {
			break
		}

		wait := rand.Intn(10)
		time.Sleep(time.Duration(wait) * time.Second)
	}

	return nil
}

func (r *RedisCache) Unlock(key string) error {
	key = r.lockName(key)
	r.lock.RLock()
	defer r.lock.RUnlock()

	_, err := r.client.DelExArgs(context.TODO(), key, redis.DelExArgs{Mode: "IFEQ", MatchValue: r.lockVal}).Result()
	if err != nil {
		return &CacheError{Operation: "DelExArgs", Err: err}
	}

	return nil
}

type CacheError struct {
	Operation string
	Err       error
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache failed to execute operation: %s because of error: %s", e.Operation, e.Err)
}

var (
	_ Cache = (*MemCache)(nil)
	_ Cache = (*RedisCache)(nil)
)
