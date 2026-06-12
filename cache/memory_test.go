package cache

import (
	"sync"
	"testing"
)

//nolint:errcheck
func TestMemCache_Basic(t *testing.T) {
	m := &MemCache{}
	m.Connect()

	key := "test_key"
	val := map[string]int{"item1": 1}

	err := m.JsonSet(key, val)
	if err != nil {
		t.Fatalf("JsonSet failed: %v", err)
	}

	res, err := m.JsonGetMap(key)
	if err != nil {
		t.Fatalf("JsonGetMap failed: %v", err)
	}

	if res["item1"] != 1 {
		t.Errorf("expected 1, got %d", res["item1"])
	}
}

//nolint:errcheck
func TestMemCache_JsonAddIncr(t *testing.T) {
	m := &MemCache{}
	m.Connect()

	key := "test_key"
	m.JsonSet(key, make(map[string]int))

	err := m.JsonAdd(key, "item1", 5)
	if err != nil {
		t.Fatalf("JsonAdd failed: %v", err)
	}

	exists, err := m.JsonExists(key, "item1")
	if err != nil || !exists {
		t.Errorf("JsonExists failed: %v, exists: %v", err, exists)
	}

	ok, err := m.JsonIncr(key, "item1", 2)
	if err != nil || !ok {
		t.Errorf("JsonIncr failed: %v, ok: %v", err, ok)
	}

	res, _ := m.JsonGetMap(key)
	if res["item1"] != 7 {
		t.Errorf("expected 7, got %d", res["item1"])
	}

	// Test decrement
	ok, err = m.JsonIncr(key, "item1", -10)
	if err != nil {
		t.Fatalf("JsonIncr failed: %v", err)
	}
	if ok {
		t.Errorf("JsonIncr should have failed for negative result")
	}

	if res["item1"] != 7 {
		t.Errorf("expected 7 after failed decrement, got %d", res["item1"])
	}
}

//nolint:errcheck
func TestMemCache_Concurrency(t *testing.T) {
	m := &MemCache{}
	m.Connect()

	key := "counts"
	m.JsonSet(key, map[string]int{"item": 0})

	var wg sync.WaitGroup
	numGoroutines := 100
	iterations := 100

	for range numGoroutines {
		wg.Go(func() {
			for range iterations {
				m.JsonIncr(key, "item", 1)
			}
		})
	}

	wg.Wait()

	res, _ := m.JsonGetMap(key)
	expected := numGoroutines * iterations
	if res["item"] != expected {
		t.Errorf("expected %d, got %d", expected, res["item"])
	}
}

//nolint:errcheck
func TestMemCache_ConcurrentMapAccess(t *testing.T) {
	m := &MemCache{}
	m.Connect()

	key := "counts"
	m.JsonSet(key, map[string]int{"item": 0})

	var wg sync.WaitGroup
	numGoroutines := 100

	for range numGoroutines {
		wg.Go(func() {
			for range 100 {
				res, _ := m.JsonGetMap(key)
				if res != nil {
					_ = res["item"] // Read
				}
			}
		})
	}

	for range numGoroutines {
		wg.Go(func() {
			for range 100 {
				m.JsonIncr(key, "item", 1) // Write (internal to MemCache, but uses the same map)
			}
		})
	}

	wg.Wait()
}

//nolint:errcheck
func TestMemCache_PanicCheck(t *testing.T) {
	m := &MemCache{}
	m.Connect()

	// 1. Key not found
	_, err := m.JsonGetMap("nonexistent")
	if err != nil {
		t.Errorf("JsonGetMap should return nil, nil for nonexistent key, got err: %v", err)
	}

	// 2. Wrong type
	m.JsonSet("wrong_type", []int64{1, 2, 3})

	// Expect NO panic, but an error
	res, err := m.JsonGetMap("wrong_type")
	if err == nil {
		t.Errorf("JsonGetMap should return error for wrong type, not nil. got res: %v", res)
	}
}
