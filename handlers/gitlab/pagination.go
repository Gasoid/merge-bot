package gitlab

import (
	"iter"

	"github.com/gasoid/merge-bot/v3/logger"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

func paginate[T any](
	fetchPage func(page, perPage int64) ([]T, *gitlab.Response, error),
	size int64,
) iter.Seq[T] {
	return func(yield func(T) bool) {
		var page int64 = 1

		for {
			items, resp, err := fetchPage(page, size)
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

func (g GitlabProvider) listBranches(projectID, size int64) iter.Seq[*gitlab.Branch] {
	return paginate(func(page, perPage int64) ([]*gitlab.Branch, *gitlab.Response, error) {
		return g.client.Branches.ListBranches(projectID, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage},
		})
	}, size)
}

func (g GitlabProvider) listMergeRequests(projectID, size int64, options *gitlab.ListProjectMergeRequestsOptions) iter.Seq[*gitlab.BasicMergeRequest] {
	return paginate(func(page, perPage int64) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListProjectMergeRequestsOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.MergeRequests.ListProjectMergeRequests(projectID, options)
	}, size)
}

func (g GitlabProvider) listAllProjectMembers(projectID, size int64, options *gitlab.ListProjectMembersOptions) iter.Seq[*gitlab.ProjectMember] {
	return paginate(func(page, perPage int64) ([]*gitlab.ProjectMember, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListProjectMembersOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.ProjectMembers.ListAllProjectMembers(projectID, options)
	}, size)
}
