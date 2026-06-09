package cache

import (
	"fmt"

	"github.com/gasoid/merge-bot/v3/logger"
)

var (
	contributors Cache
)

const (
	countsPrefix       = "mergebot:counts"
	contributorsPrefix = "mergebot:contributors"
	updateLocksPrefix  = "mergebot:update:locks"
	locksPrefix        = "mergebot:locks"
)

func contributorsKey(id int64) string {
	return fmt.Sprintf("%s:%d", contributorsPrefix, id)
}

func countsKey(id int64) string {
	return fmt.Sprintf("%s:%d", countsPrefix, id)
}

func locksKey(id int64) string {
	return fmt.Sprintf("%s:%d", locksPrefix, id)
}

func updateLockKey(id int64) string {
	return fmt.Sprintf("%s:%d", updateLocksPrefix, id)
}

func SetCounts(id int64, counts map[string]int) error {
	return contributors.JsonSet(countsKey(id), counts)
}

func GetCounts(id int64) (map[string]int, error) {
	val, err := contributors.JsonGet(countsKey(id))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	if candidates, ok := val.(map[string]int); ok {
		return candidates, nil
	}

	return nil, nil
}

func IncrCount(id int64, item string) (bool, error) {
	ok, err := contributors.JsonExists(countsKey(id), item)
	if err != nil {
		return false, err
	}

	if ok {
		return contributors.JsonIncr(countsKey(id), item, 1)
	} else {
		return true, contributors.JsonAdd(countsKey(id), item, 1)
	}
}

func DecrCount(id int64, item string) (bool, error) {
	ok, err := contributors.JsonExists(countsKey(id), item)
	if err != nil {
		return false, err
	}

	if ok {
		return contributors.JsonIncr(countsKey(id), item, -1)
	}

	return false, nil
}

func GetContributors(id int64) ([]int64, error) {
	val, err := contributors.JsonGet(contributorsKey(id))
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	if candidates, ok := val.([]int64); ok {
		return candidates, nil
	}

	return nil, nil
}

func SetContributors(id int64, candidates []int64) error {
	logger.Debug("save contributors", "size", len(candidates))
	return contributors.JsonSet(contributorsKey(id), candidates)
}

func TryAcquireBranchDeletionLock(id int64) bool {
	return contributors.TryAcquireLock(locksKey(id))
}

func BranchDeletionUnlock(id int64) {
	contributors.Unlock(locksKey(id))
}

func TryAcquireUpdateLock(id int64) bool {
	return contributors.TryAcquireLock(updateLockKey(id))
}

func UpdateUnlock(id int64) {
	contributors.Unlock(updateLockKey(id))
}
