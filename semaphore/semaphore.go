package semaphore

import (
	"fmt"
	"sync"
)

type keyState struct {
	mu      sync.RWMutex
	running int
}

// counting semaphore
type KeyedSemaphore struct {
	counters  map[string]*keyState
	maxPerKey int
	mu        sync.RWMutex
}

func NewKeyedSemaphore(maxPerKey int) *KeyedSemaphore {
	return &KeyedSemaphore{
		counters:  map[string]*keyState{},
		maxPerKey: maxPerKey,
	}
}

func (s *KeyedSemaphore) Print() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Printf("length: %d", len(s.counters))

	for k := range s.counters {
		fmt.Printf("key: %s", k)
	}
}

func (s *KeyedSemaphore) Add(key string, task func()) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, ok := s.counters[key]; ok {
		if state.running == s.maxPerKey {
			return false
		}
		state.running++
	} else {
		s.counters[key] = &keyState{
			running: 1,
		}
	}

	go s.run(key, s.counters[key], task)
	return true
}

func (s *KeyedSemaphore) clean(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if state, ok := s.counters[key]; ok {
		if state.running == 0 {
			delete(s.counters, key)
		}
	}
}

func (s *KeyedSemaphore) run(key string, state *keyState, task func()) {
	state.mu.Lock()
	defer state.mu.Unlock()
	defer s.clean(key)

	defer func() {
		state.running--
	}()

	task()
}
