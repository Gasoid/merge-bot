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
	Lock(key string) error
	Unlock(key string) error
	Set(key, val string) error
	Get(key string) (string, error)
	Add(key, candidate string, score float64) error
	Connect() error
}

type MemCache struct {
	items        []string
	keys         map[string]string
	memcacheLock sync.RWMutex
}

func (m *MemCache) Add(key, item string, score float64) error {
	return nil
}

func (m *MemCache) Set(key, val string) error {
	m.keys[key] = val
	return nil
}

func (m *MemCache) Get(key string) (string, error) {
	if val, ok := m.keys[key]; ok {
		return val, nil
	}

	return "", nil
}

func (m *MemCache) Lock(key string) error {
	m.memcacheLock.Lock()
	return nil
}

func (m *MemCache) Unlock(key string) error {
	m.memcacheLock.Unlock()
	return nil
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
		return &CacheError{Operation: "Set", Err: err}
	}

	return nil
}

func (r *RedisCache) Get(key string) (string, error) {
	val, err := r.client.Get(context.TODO(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}

		return "", &CacheError{Operation: "Set", Err: err}
	}

	return val, nil
}

func (r *RedisCache) Add(key, item string, score float64) error {
	if _, err := r.client.ZAdd(context.TODO(), key, redis.Z{Member: item, Score: score}).Result(); err != nil {
		return &CacheError{Operation: "ZAdd", Err: err}
	}

	exists, err := r.client.SIsMember(context.TODO(), r.hashName(key), item).Result()
	if err != nil {
		return &CacheError{Operation: "SIsMember", Err: err}
	}

	if exists {
		return nil
	}

	if _, err := r.client.SAdd(context.TODO(), r.hashName(key), item).Result(); err != nil {
		return &CacheError{Operation: "SAdd", Err: err}
	}

	if _, err := r.client.RPush(context.TODO(), key, item).Result(); err != nil {
		return &CacheError{Operation: "RPush", Err: err}
	}

	r.client.Expire(context.TODO(), key, time.Duration(24)*time.Hour)
	r.client.Expire(context.TODO(), r.hashName(key), time.Duration(24)*time.Hour)

	return nil
}

func (r *RedisCache) Pop(key string) (string, error) {
	if err := r.Lock(key); err != nil {
		return "", &CacheError{Operation: "Lock", Err: err}
	}

	defer r.Unlock(key)

	item, err := r.client.LPop(context.TODO(), key).Result()
	if err != nil {
		return "", &CacheError{Operation: "LPop", Err: err}
	}

	if _, err := r.client.SRem(context.TODO(), r.hashName(key), item).Result(); err != nil {
		return "", &CacheError{Operation: "SRem", Err: err}
	}

	return item, nil
}

func (r *RedisCache) lockName(key string) string {
	return fmt.Sprintf("%s:lock", key)
}

func (r *RedisCache) hashName(key string) string {
	return fmt.Sprintf("%s:unique", key)
}

func (r *RedisCache) Lock(key string) error {
	key = r.lockName(key)
	r.lock.Lock()
	defer r.lock.Unlock()

	r.lockVal = rand.Intn(1000)
	for {
		done, err := r.client.SetNX(context.TODO(), key, r.lockVal, time.Second*300).Result()
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
