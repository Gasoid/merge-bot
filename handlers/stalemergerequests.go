package handlers

import (
	"fmt"
	"slices"
	"time"

	"github.com/dustin/go-humanize/english"
	"github.com/gasoid/merge-bot/v3/metrics"
)

type MR struct {
	Id          int
	Branch      string
	Protected   bool
	Labels      []string
	LastUpdated time.Time
}

func (r Request) cleanStaleMergeRequests() error {
	var (
		days            = r.config.StaleBranchesDeletion.Days
		coolDays        = r.config.StaleBranchesDeletion.WaitDays
		now             = time.Now()
		branchesDeleted = 0
		excludeBranches = make(map[string]struct{}, len(r.config.StaleBranchesDeletion.ExcludeBranches))
	)

	defer func() {
		duration := time.Since(now)
		metrics.MrDeletionDuration(duration)
	}()

	for _, s := range r.config.StaleBranchesDeletion.ExcludeBranches {
		excludeBranches[s] = struct{}{}
	}

	for mr := range r.provider.ListMergeRequests(r.info.ProjectId, r.config.StaleBranchesDeletion.BatchSize, r.config.StaleBranchesDeletion.Protected) {
		if branchesDeleted >= r.config.StaleBranchesDeletion.BatchSize {
			break
		}

		span := now.Sub(mr.LastUpdated)
		if slices.Contains(mr.Labels, staleLabel) {
			if span > time.Duration(time.Duration(coolDays)*24*time.Hour) {
				if err := r.provider.DeleteBranch(r.info.ProjectId, mr.Branch); err != nil {
					return fmt.Errorf("DeleteBranch returns error: %w", err)
				}
				branchesDeleted++
				metrics.MrDeletionInc()
				continue
			}
		}

		if _, ok := excludeBranches[mr.Branch]; ok {
			continue
		}

		if span > time.Duration(time.Duration(days)*24*time.Hour) {
			// mr is stale
			if err := r.provider.AssignLabel(r.info.ProjectId, mr.Id, staleLabel, staleLabelColor); err != nil {
				return fmt.Errorf("AssignLabel returns error: %w", err)
			}

			pluralDays := english.Plural(days, "day", "")
			pluralCoolDays := english.Plural(coolDays, "day", "")

			message := fmt.Sprintf("This MR is stale because it has been open %s with no activity. Remove stale label othewise this will be closed in %s.", pluralDays, pluralCoolDays)
			if err := r.provider.LeaveComment(r.info.ProjectId, mr.Id, message); err != nil {
				return fmt.Errorf("LeaveComment returns error: %w", err)
			}

		}
	}

	return nil
}
