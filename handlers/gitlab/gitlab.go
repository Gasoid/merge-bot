package gitlab

import (
	b64 "encoding/base64"
	"log/slog"
	"mergebot/handlers"
	"os"

	"github.com/xanzy/go-gitlab"
)

func init() {
	handlers.Register("gitlab", New)
}

const (
	gitlabToken   = "GITLAB_TOKEN"
	gitlabUrl     = "GITLAB_URL"
	tokenUsername = "oauth2"
	maxRepoSize   = 1000 * 1000 * 500 // 500Mb
)

type GitlabProvider struct {
	client *gitlab.Client
	mr     *gitlab.MergeRequest
}

func (g *GitlabProvider) loadMR(projectId, mergeId int) error {
	if g.mr != nil {
		return nil
	}

	mr, _, err := g.client.MergeRequests.GetMergeRequest(projectId, mergeId, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return err
	}

	g.mr = mr
	return nil
}

func (g *GitlabProvider) UpdateFromMaster(projectId, mergeId int) error {
	if err := g.loadMR(projectId, mergeId); err != nil {
		return err
	}

	project, _, err := g.client.Projects.GetProject(
		projectId,
		&gitlab.GetProjectOptions{Statistics: gitlab.Bool(true)},
	)
	if err != nil {
		return err
	}

	if project.Statistics.RepositorySize > maxRepoSize {
		return handlers.RepoSizeError
	}

	return handlers.MergeMaster(
		tokenUsername,
		os.Getenv(gitlabToken),
		project.HTTPURLToRepo,
		g.mr.SourceBranch,
		g.mr.TargetBranch,
	)
}

func (g *GitlabProvider) LeaveComment(projectId, mergeId int, message string) error {
	slog.Debug("leaveComment in gitlab", "message", message, "projectId", projectId)

	_, _, err := g.client.Notes.CreateMergeRequestNote(
		projectId,
		mergeId,
		&gitlab.CreateMergeRequestNoteOptions{Body: &message},
	)

	return err
}

func (g *GitlabProvider) Merge(projectId, mergeId int, message string) error {
	t := true
	_, _, err := g.client.MergeRequests.AcceptMergeRequest(projectId,
		mergeId,
		&gitlab.AcceptMergeRequestOptions{Squash: &t, ShouldRemoveSourceBranch: &t, SquashCommitMessage: &message},
	)

	return err
}

func (g *GitlabProvider) GetApprovals(projectId, mergeId int) (map[string]struct{}, error) {
	page := 1
	approvals := map[string]struct{}{}
	for {
		notes, resp, err := g.client.Notes.ListMergeRequestNotes(
			projectId,
			mergeId,
			&gitlab.ListMergeRequestNotesOptions{ListOptions: gitlab.ListOptions{Page: page}})
		if err != nil {
			return nil, err
		}

		for _, note := range notes {
			if g.mr.Author.ID == note.Author.ID {
				continue
			}

			if note.System {
				if note.Body == "approved this merge request" {
					approvals[note.Author.Username] = struct{}{}
				}
				if note.Body == "unapproved this merge request" {
					delete(approvals, note.Author.Username)
				}
			}
		}
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage

	}
	return approvals, nil
}

func (g *GitlabProvider) GetFailedPipelines() (int, error) {
	if g.mr.HeadPipeline != nil && g.mr.HeadPipeline.Status != string(gitlab.DeploymentStatusSuccess) {
		return 1, nil
	}

	return 0, nil
}

func (g *GitlabProvider) IsValid(projectId, mergeId int) (bool, error) {
	if err := g.loadMR(projectId, mergeId); err != nil {
		return false, err
	}

	if g.mr.State != "opened" {
		return false, handlers.StatusError
	}

	return !g.mr.HasConflicts, nil
}

func (g *GitlabProvider) GetFile(projectId int, path string) (string, error) {
	project, _, err := g.client.Projects.GetProject(projectId, &gitlab.GetProjectOptions{})
	if err != nil {
		return "", err
	}

	gitlabFile, _, err := g.client.RepositoryFiles.GetFile(projectId, path, &gitlab.GetFileOptions{Ref: &project.DefaultBranch})
	if err != nil {
		return "", err
	}

	content, _ := b64.StdEncoding.DecodeString(gitlabFile.Content)
	return string(content), nil
}

func (g *GitlabProvider) GetMRInfo(projectId, mergeId int, configPath string) (*handlers.MrInfo, error) {
	var err error
	info := handlers.MrInfo{}
	info.IsValid, err = g.IsValid(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	info.ConfigContent, err = g.GetFile(projectId, configPath)
	if err != nil {
		slog.Info("i am using default config to validate a request")
	}

	info.Title = g.mr.Title
	info.Description = g.mr.Description
	info.Approvals, err = g.GetApprovals(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	info.FailedPipelines, err = g.GetFailedPipelines()
	if err != nil {
		return nil, err
	}

	if g.mr.HeadPipeline != nil {
		report, _, err := g.client.Pipelines.GetPipelineTestReport(projectId, g.mr.HeadPipeline.IID)
		if err != nil {
			return nil, err
		}
		info.FailedTests = report.FailedCount
	}

	return &info, nil
}

func (g *GitlabProvider) ListBranches(projectId int) ([]handlers.Branch, error) {
	branches, _, err := g.client.Branches.ListBranches(projectId, &gitlab.ListBranchesOptions{})
	if err != nil {
		return nil, err
	}

	staleBranches := []handlers.Branch{}
	for _, b := range branches {
		if b.Default || b.Protected {
			continue
		}

		staleBranches = append(staleBranches, handlers.Branch{Name: b.Name, LastUpdated: *b.Commit.CreatedAt})
	}
	return staleBranches, nil
}

func (g *GitlabProvider) DeleteBranch(projectId int, name string) error {
	_, err := g.client.Branches.DeleteBranch(projectId, name)
	return err
}

func New() handlers.RequestProvider {
	var err error
	var p GitlabProvider

	token := os.Getenv(gitlabToken)
	if token == "" {
		slog.Error("gitlab init", "err", "gitlab requires token, please set env variable GITLAB_TOKEN")
		return nil
	}

	urlInstance := os.Getenv(gitlabUrl)

	if urlInstance != "" {
		p.client, err = gitlab.NewClient(token, gitlab.WithBaseURL(urlInstance))
	} else {
		p.client, err = gitlab.NewClient(token)
	}
	if err != nil {
		slog.Error("gitlabProvider new", "err", err)
		return nil
	}

	return &p
}

var (
	_ handlers.RequestProvider = (*GitlabProvider)(nil)
)
