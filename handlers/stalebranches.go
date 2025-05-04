package handlers

import (
	"log/slog"
	"time"
)

type Branch struct {
	Name        string
	LastUpdated time.Time
}

func (r *Request) cleanStaleBranches(projectId int) {
	candidates, err := r.provider.ListBranches(projectId)
	if err != nil {
		slog.Error("ListBranches returns error", "err", err)
		return
	}

	days := r.config.StaleBranchesDeletion.Days
	for _, b := range candidates {
		now := time.Now()
		span := now.Sub(b.LastUpdated)
		if span > time.Duration(time.Duration(days)*24*time.Hour) {
			// branch is stale
			// delete branch
			slog.Debug("branch info", "name", b.Name, "createdAt", b.LastUpdated.String())
			if err := r.provider.DeleteBranch(projectId, b.Name); err != nil {
				slog.Error("DeleteBranch returns error", "branch", b.Name, "err", err)
			}
		}
	}
}
