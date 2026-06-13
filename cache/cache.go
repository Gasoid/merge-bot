package cache

import (
	"errors"
	"fmt"
	"time"
)

type CacheJson interface {
	JsonSet(key string, v any) error
	JsonGet(key string) ([]int64, error)
	JsonGetMap(key string) (map[string]int, error)
	JsonExists(key, item string) (bool, error)
	JsonAdd(key, item string, v int) error
	JsonIncr(key string, item string, v int) (bool, error)
}

type CacheLock interface {
	TryAcquireLock(key string) bool
	Unlock(key string)
}

type CacheBase interface {
	ExtendTTL(key string, ttl time.Duration) error
	Connect() error
	IsHealthy() bool
}

type Cache interface {
	CacheJson
	CacheLock
	CacheBase
}

var (
	ErrNotFound  = errors.New("key not found")
	ErrWrongType = errors.New("wrong type")
)

type CacheError struct {
	Operation string
	Err       error
}

func (e *CacheError) Error() string {
	return fmt.Sprintf("cache failed to execute operation: %s because of error: %s", e.Operation, e.Err)
}

func Init() error {
	if redisUrl == "" {
		contributors = &MemCache{}
	} else {
		contributors = &RedisCache{}
	}
	return contributors.Connect()
}
