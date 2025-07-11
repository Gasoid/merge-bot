package handlers

import (
	"fmt"
	"sync"
	"time"

	"github.com/Gasoid/mergebot/logger"
)

var (
	cleanStaleBranchesLock sync.Mutex
)

type StaleBranch struct {
	Name        string
	LastUpdated time.Time
}

func (r Request) cleanStaleBranches() error {
	cleanStaleBranchesLock.Lock()
	defer cleanStaleBranchesLock.Unlock()

	logger.Debug("deletion of stale branches has been run")

	candidates, err := r.provider.ListBranches(r.info.ProjectId, r.config.StaleBranchesDeletion.BatchSize)
	if err != nil {
		return fmt.Errorf("ListBranches returns error: %w", err)
	}

	days := r.config.StaleBranchesDeletion.Days
	now := time.Now()
	for _, b := range candidates {
		span := now.Sub(b.LastUpdated)
		if span > time.Duration(time.Duration(days)*24*time.Hour) {
			// branch is stale
			// delete branch
			logger.Debug("branch info", "name", b.Name, "createdAt", b.LastUpdated.String())
			if err := r.provider.DeleteBranch(r.info.ProjectId, b.Name); err != nil {
				return fmt.Errorf("DeleteBranch returns error: %w", err)
			}
		}
	}

	return nil
}
