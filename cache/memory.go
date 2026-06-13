package cache

import (
	"fmt"
	"maps"
	"sync"
	"time"
)

type MemCache struct {
	mu           sync.Mutex
	locks        map[string]bool
	keys         map[string]any
	memcacheLock sync.RWMutex
}

func (m *MemCache) JsonSet(key string, v any) error {
	return m.set(key, v)
}

func (m *MemCache) JsonAdd(key string, item string, v int) error {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	data, ok := m.keys[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	val, ok := data.(map[string]int)
	if !ok {
		return fmt.Errorf("%w: expected map[string]int for key %s", ErrWrongType, key)
	}

	if v < 0 {
		v = 0
	}

	val[item] = v
	return nil
}

func (m *MemCache) JsonGet(key string) ([]int64, error) {
	m.memcacheLock.RLock()
	defer m.memcacheLock.RUnlock()

	val, ok := m.keys[key]
	if !ok || val == nil {
		return nil, nil
	}

	data, ok := val.([]int64)
	if !ok {
		return nil, fmt.Errorf("%w: expected []int64 for key %s", ErrWrongType, key)
	}

	res := make([]int64, len(data))
	copy(res, data)
	return res, nil
}

func (m *MemCache) JsonGetMap(key string) (map[string]int, error) {
	m.memcacheLock.RLock()
	defer m.memcacheLock.RUnlock()

	val, ok := m.keys[key]
	if !ok || val == nil {
		return nil, nil
	}

	data, ok := val.(map[string]int)
	if !ok {
		return nil, fmt.Errorf("%w: expected map[string]int for key %s", ErrWrongType, key)
	}

	res := make(map[string]int, len(data))
	maps.Copy(res, data)
	return res, nil
}

func (m *MemCache) JsonIncr(key, item string, v int) (bool, error) {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	data, ok := m.keys[key]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	val, ok := data.(map[string]int)
	if !ok {
		return false, fmt.Errorf("%w: expected map[string]int for key %s", ErrWrongType, key)
	}

	if _, ok := val[item]; ok {
		old := val[item]
		val[item] += v
		if val[item] < 0 {
			val[item] = old
			return false, nil
		}
	}

	return true, nil
}

func (m *MemCache) JsonExists(key, item string) (bool, error) {
	m.memcacheLock.RLock()
	defer m.memcacheLock.RUnlock()

	data, ok := m.keys[key]
	if !ok {
		return false, nil
	}

	val, ok := data.(map[string]int)
	if !ok {
		return false, nil
	}

	_, exists := val[item]
	return exists, nil
}

func (m *MemCache) ExtendTTL(key string, ttl time.Duration) error {
	return nil
}

func (m *MemCache) set(key string, val any) error {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	if m.keys == nil {
		m.keys = make(map[string]any)
	}

	m.keys[key] = val
	return nil
}

func (m *MemCache) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.locks == nil {
		m.locks = make(map[string]bool)
	}

	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	if m.keys == nil {
		m.keys = make(map[string]any)
	}

	return nil
}

func (m *MemCache) TryAcquireLock(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.locks[key] {
		m.locks[key] = true
		return true
	} else {
		return false
	}
}

func (m *MemCache) Unlock(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.locks[key] = false
}

var (
	_ Cache = (*MemCache)(nil)
)
