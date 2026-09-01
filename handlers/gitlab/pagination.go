package gitlab

import (
	"iter"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

func paginate[T any](
	fetchPage func(page, perPage int64) ([]T, *gitlab.Response, error),
	size int64,
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var page int64 = 1

		for {
			items, resp, err := fetchPage(page, size)
			for _, item := range items {
				if !yield(item, err) {
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

func (g GitlabProvider) listBranches(projectID, size int64) iter.Seq2[*gitlab.Branch, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.Branch, *gitlab.Response, error) {
		return g.client.Branches.ListBranches(projectID, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage},
		})
	}, size)
}

func (g GitlabProvider) listMergeRequests(projectID, size int64, options *gitlab.ListProjectMergeRequestsOptions) iter.Seq2[*gitlab.BasicMergeRequest, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.BasicMergeRequest, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListProjectMergeRequestsOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.MergeRequests.ListProjectMergeRequests(projectID, options)
	}, size)
}

func (g GitlabProvider) listAllProjectMembers(projectID, size int64, options *gitlab.ListProjectMembersOptions) iter.Seq2[*gitlab.ProjectMember, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.ProjectMember, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListProjectMembersOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.ProjectMembers.ListAllProjectMembers(projectID, options)
	}, size)
}

func (g GitlabProvider) listSearch(projectID, size int64, query, branch string) iter.Seq2[*gitlab.Blob, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.Blob, *gitlab.Response, error) {
		options := &gitlab.SearchOptions{Ref: new(branch)}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.Search.BlobsByProject(projectID, query, options)
	}, size)
}

func (g GitlabProvider) listPipelineJobs(projectID, pipelineID, size int64, options *gitlab.ListJobsOptions) iter.Seq2[*gitlab.Job, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.Job, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListJobsOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.Jobs.ListPipelineJobs(projectID, pipelineID, options)
	}, size)
}

func (g GitlabProvider) listMergeRequestDiffs(projectID, mergeID, size int64, options *gitlab.ListMergeRequestDiffsOptions) iter.Seq2[*gitlab.MergeRequestDiff, error] {
	return paginate(func(page, perPage int64) ([]*gitlab.MergeRequestDiff, *gitlab.Response, error) {
		if options == nil {
			options = &gitlab.ListMergeRequestDiffsOptions{}
		}
		options.ListOptions = gitlab.ListOptions{Page: page, PerPage: perPage}
		return g.client.MergeRequests.ListMergeRequestDiffs(projectID, mergeID, options)
	}, size)
}
