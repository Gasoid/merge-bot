package gitlab

import (
	"iter"

	"github.com/Gasoid/merge-bot/logger"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func (g GitlabProvider) listBranches(projectId, size int) iter.Seq[*gitlab.Branch] {
	page := 1

	return func(yield func(b *gitlab.Branch) bool) {
		for {
			branches, resp, err := g.client.Branches.ListBranches(projectId, &gitlab.ListBranchesOptions{
				ListOptions: gitlab.ListOptions{
					Page:    page,
					PerPage: size * 2,
				},
			})
			if err != nil {
				logger.Error("listBranches", "err", err)
				return
			}

			for _, branch := range branches {
				if !yield(branch) {
					return
				}
			}

			if resp.NextPage == 0 {
				return
			}

			page = resp.NextPage
		}
	}
}

func (g GitlabProvider) listMergeRequests(projectId, size int, options *gitlab.ListProjectMergeRequestsOptions) iter.Seq[*gitlab.BasicMergeRequest] {
	page := 1

	return func(yield func(*gitlab.BasicMergeRequest) bool) {
		for {
			options.ListOptions = gitlab.ListOptions{
				Page:    page,
				PerPage: size * 2,
			}

			mrs, resp, err := g.client.MergeRequests.ListProjectMergeRequests(projectId, options)
			if err != nil {
				logger.Error("listMergeRequests", "err", err)
				return
			}

			for _, mergeRequest := range mrs {
				if !yield(mergeRequest) {
					return
				}
			}

			if resp.NextPage == 0 {
				return
			}

			page = resp.NextPage
		}
	}
}
