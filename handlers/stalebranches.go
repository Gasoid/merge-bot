package handlers

import (
	"fmt"
	"time"

	"github.com/gasoid/merge-bot/logger"
	"github.com/gasoid/merge-bot/metrics"
)

type StaleBranch struct {
	Name        string
	LastUpdated time.Time
	Protected   bool
}

func (r Request) cleanStaleBranches() error {
	logger.Debug("deletion of stale branches has been run")
	var (
		branchesDeleted = 0
		days            = r.config.StaleBranchesDeletion.Days
		now             = time.Now()
		excludeBranches = make(map[string]struct{}, len(r.config.StaleBranchesDeletion.ExcludeBranches))
	)

	defer func() {
		duration := time.Since(now)
		metrics.BranchDeletionDuration(duration)
	}()

	for _, s := range r.config.StaleBranchesDeletion.ExcludeBranches {
		excludeBranches[s] = struct{}{}
	}

	for b := range r.provider.ListBranches(r.info.ProjectId, r.config.StaleBranchesDeletion.BatchSize, r.config.StaleBranchesDeletion.Protected) {
		if branchesDeleted >= r.config.StaleBranchesDeletion.BatchSize {
			break
		}

		if _, ok := excludeBranches[b.Name]; ok {
			continue
		}

		span := now.Sub(b.LastUpdated)
		if span > time.Duration(time.Duration(days)*24*time.Hour) {
			// branch is stale
			// delete branch
			logger.Debug("branch info", "name", b.Name, "createdAt", b.LastUpdated.String())
			if err := r.provider.DeleteBranch(r.info.ProjectId, b.Name); err != nil {
				return fmt.Errorf("DeleteBranch returns error: %w", err)
			}
			branchesDeleted++
			metrics.BranchDeletionInc()
		}
	}

	return nil
}
