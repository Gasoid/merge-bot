package cache

import (
	"sync"
)

type MemCache struct {
	keys         map[string]any
	memcacheLock sync.RWMutex
}

func (m *MemCache) JsonSet(key string, v any) error {
	return m.set(key, v)
}

func (m *MemCache) JsonAdd(key string, item string, v int) error {
	data, err := m.get(key)
	if err != nil {
		return err
	}

	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	val := data.(map[string]int)
	val[item] = v
	m.keys[key] = val

	return nil
}

func (m *MemCache) JsonGet(key string) (any, error) {
	return m.get(key)
}

func (m *MemCache) JsonIncr(key, item string, v int) (bool, error) {
	val, err := m.get(key)
	if err != nil {
		return false, err
	}

	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	data := val.(map[string]int)
	if _, ok := data[item]; ok {
		//old := data[item]
		data[item] += v
		if data[item] < 0 {
			//data[item] = old
			return false, nil
		}

		m.keys[key] = data
	}

	return true, nil
}

func (m *MemCache) JsonExists(key, item string) (bool, error) {
	val, err := m.get(key)
	if err != nil {
		return false, err
	}

	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	data := val.(map[string]int)

	_, ok := data[item]
	return ok, nil
}

func (m *MemCache) set(key string, val any) error {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	m.keys[key] = val
	return nil
}

func (m *MemCache) get(key string) (any, error) {
	m.memcacheLock.Lock()
	defer m.memcacheLock.Unlock()

	if val, ok := m.keys[key]; ok {
		return val, nil
	}

	return nil, nil
}

func (m *MemCache) Connect() error {
	if m.keys == nil {
		m.keys = make(map[string]any)
	}

	return nil
}

var (
	_ Cache = (*MemCache)(nil)
)
