package cache

import (
	"fmt"
	"time"
)

type Cache interface {
	JsonSet(key string, v any) error
	JsonGet(key string) ([]int64, error)
	JsonGetMap(key string) (map[string]int, error)
	JsonExists(key, item string) (bool, error)
	JsonAdd(key, item string, v int) error
	JsonIncr(key string, item string, v int) (bool, error)
	ExtendTTL(key string, ttl time.Duration) error
	TryAcquireLock(key string) bool
	Unlock(key string)
	Connect() error
}

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
