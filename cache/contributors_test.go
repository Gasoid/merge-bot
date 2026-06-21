package cache

import (
	"testing"
)

//nolint:errcheck
func TestContributors_Counts(t *testing.T) {
	// Ensure cache is initialized with MemCache
	redisUrl = ""
	Init()

	id := int64(123)
	counts := map[string]int{
		"user1": 5,
		"user2": 10,
	}

	// Test SetCounts
	err := SetCounts(id, counts)
	if err != nil {
		t.Fatalf("SetCounts failed: %v", err)
	}

	// Test GetCounts
	res, err := GetCounts(id)
	if err != nil {
		t.Fatalf("GetCounts failed: %v", err)
	}

	if res["user1"] != 5 || res["user2"] != 10 {
		t.Errorf("expected counts map %v, got %v", counts, res)
	}

	// Test IncrCount existing item
	ok, err := IncrCount(id, "user1")
	if err != nil || !ok {
		t.Errorf("IncrCount failed: %v, ok: %v", err, ok)
	}

	res, _ = GetCounts(id)
	if res["user1"] != 6 {
		t.Errorf("expected user1 count to be 6, got %d", res["user1"])
	}

	// Test IncrCount new item
	ok, err = IncrCount(id, "user3")
	if err != nil || !ok {
		t.Errorf("IncrCount for new item failed: %v, ok: %v", err, ok)
	}

	res, _ = GetCounts(id)
	if res["user3"] != 1 {
		t.Errorf("expected user3 count to be 1, got %d", res["user3"])
	}

	// Test DecrCount
	ok, err = DecrCount(id, "user2")
	if err != nil || !ok {
		t.Errorf("DecrCount failed: %v, ok: %v", err, ok)
	}

	res, _ = GetCounts(id)
	if res["user2"] != 9 {
		t.Errorf("expected user2 count to be 9, got %d", res["user2"])
	}

	// Test DecrCount below zero (should stay at old value and return false based on MemCache implementation)
	// Actually DecrCount calls JsonIncr(..., -1).
	// In our MemCache implementation, if it goes below 0, it reverts and returns false.
	countsZero := map[string]int{"low": 0}
	SetCounts(id, countsZero)
	ok, err = DecrCount(id, "low")
	if err != nil {
		t.Errorf("DecrCount failed with error: %v", err)
	}
	if ok {
		t.Errorf("DecrCount should return false when decrementing zero")
	}
}

//nolint:errcheck
func TestContributors_Candidates(t *testing.T) {
	redisUrl = ""
	Init()

	id := int64(456)
	candidates := []int64{1, 2, 3}

	err := SetContributors(id, candidates)
	if err != nil {
		t.Fatalf("SetContributors failed: %v", err)
	}

	res, err := GetContributors(id)
	if err != nil {
		t.Fatalf("GetContributors failed: %v", err)
	}

	if len(res) != 3 || res[0] != 1 || res[2] != 3 {
		t.Errorf("expected %v, got %v", candidates, res)
	}
}

//nolint:errcheck
func TestContributors_Locks(t *testing.T) {
	redisUrl = ""
	Init()

	id := int64(789)

	// Branch Deletion Lock
	if !AcquireBranchDeletionLease(id) {
		t.Error("failed to acquire branch deletion lock")
	}
	if AcquireBranchDeletionLease(id) {
		t.Error("should not be able to acquire held branch deletion lock")
	}
	ReleaseBranchDeletionLease(id)
	if !AcquireBranchDeletionLease(id) {
		t.Error("failed to re-acquire branch deletion lock after unlock")
	}

	// Update Lock
	if !AcquireUpdateLease(id) {
		t.Error("failed to acquire update lock")
	}
	if AcquireUpdateLease(id) {
		t.Error("should not be able to acquire held update lock")
	}
	ReleaseUpdateLease(id)
	if !AcquireUpdateLease(id) {
		t.Error("failed to re-acquire update lock after unlock")
	}
}
