package gitlab

import (
	"iter"

	"github.com/gasoid/merge-bot/logger"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func paginate[T any](
	fetchPage func(page, perPage int) ([]T, *gitlab.Response, error),
	size int,
) iter.Seq[T] {
	return func(yield func(T) bool) {
		page := 1
		perPage := size * 2

		for {
			items, resp, err := fetchPage(page, perPage)
			if err != nil {
				logger.Error("pagination error", "err", err)
				return
			}

			for _, item := range items {
				if !yield(item) {
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

func (g GitlabProvider) listBranches(projectId, size int) iter.Seq[*gitlab.Branch] {
	return paginate(func(page, perPage int) ([]*gitlab.Branch, *gitlab.Response, error) {
		return g.client.Branches.ListBranches(projectId, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage},
		})
	}, size)
}

func (g GitlabProvider) listMergeRequests(projectId, size int, options *gitlab.ListProjectMergeRequestsOptions) iter.Seq[*gitlab.BasicMergeRequest] {
	return paginate(func(page, perPage int) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListProjectMergeRequestsOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.MergeRequests.ListProjectMergeRequests(projectId, options)
	}, size)
}

func (g GitlabProvider) listMergeRequestNotes(projectId, mergeId, size int) iter.Seq[*gitlab.Note] {
	return paginate(func(page, perPage int) ([]*gitlab.Note, *gitlab.Response, error) {
		return g.client.Notes.ListMergeRequestNotes(projectId, mergeId, &gitlab.ListMergeRequestNotesOptions{
			ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage},
		})
	}, size)
}
