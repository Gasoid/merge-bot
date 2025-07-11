package handlers

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

var (
	cleanStaleMergeRquestsLock sync.Mutex
)

const (
	staleLabel      = "merge-bot:stale"
	staleLabelColor = "#cccccc"
)

type StaleMergeRequest struct {
	Id          int
	Branch      string
	Labels      []string
	LastUpdated time.Time
}

func (r *Request) cleanStaleMergeRequests(projectId int) error {
	cleanStaleMergeRquestsLock.Lock()
	defer cleanStaleMergeRquestsLock.Unlock()

	days := r.config.StaleBranchesDeletion.Days
	coolDays := r.config.StaleBranchesDeletion.WaitDays
	now := time.Now()

	candidates, err := r.provider.ListMergeRequests(projectId, r.config.StaleBranchesDeletion.BatchSize)
	if err != nil {
		return fmt.Errorf("ListMergeRequests returns error: %w", err)
	}

	for _, mr := range candidates {
		span := now.Sub(mr.LastUpdated)
		if slices.Contains(mr.Labels, staleLabel) {
			if span > time.Duration(time.Duration(coolDays)*24*time.Hour) {
				if err := r.provider.DeleteBranch(projectId, mr.Branch); err != nil {
					return fmt.Errorf("DeleteBranch returns error: %w", err)
				}
			}
		}

		if span > time.Duration(time.Duration(days)*24*time.Hour) {
			// mr is stale
			if err := r.provider.AssignLabel(projectId, mr.Id, staleLabel, staleLabelColor); err != nil {
				return fmt.Errorf("AssignLabel returns error: %w", err)
			}

			message := fmt.Sprintf("This MR is stale because it has been open %d days with no activity. Remove stale label othewise this will be closed in %d days.", days, coolDays)
			if err := r.provider.LeaveComment(projectId, mr.Id, message); err != nil {
				return fmt.Errorf("LeaveComment returns error: %w", err)
			}

		}
	}

	return nil
}
