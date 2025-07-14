package semaphore

import (
	"fmt"
	"sync"
)

type current struct {
	mu      sync.RWMutex
	running int
}

type Q struct {
	cur map[string]*current
	max int
	mu  sync.RWMutex
}

func NewQ(max int) *Q {
	return &Q{
		cur: map[string]*current{},
		max: max,
	}
}

func (q *Q) Print() {
	q.mu.RLock()
	defer q.mu.RUnlock()

	fmt.Printf("length: %d", len(q.cur))

	for k := range q.cur {
		fmt.Printf("key: %s", k)
	}
}

func (q *Q) Add(key string, execFunc func()) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if i, ok := q.cur[key]; ok {
		if i.running == q.max {
			return false
		}
		i.running++
	} else {
		q.cur[key] = &current{
			running: 1,
		}
	}

	go q.run(key, q.cur[key], execFunc)
	return true
}

func (q *Q) clean(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if i, ok := q.cur[key]; ok {
		if i.running == 0 {
			delete(q.cur, key)
		}
	}
}

func (q *Q) run(key string, cur *current, execFunc func()) {
	cur.mu.Lock()
	defer cur.mu.Unlock()
	defer q.clean(key)

	defer func() {
		cur.running--
	}()

	execFunc()
}
