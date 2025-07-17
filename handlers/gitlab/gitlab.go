package gitlab

import (
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"slices"

	"github.com/Gasoid/merge-bot/config"
	"github.com/Gasoid/merge-bot/handlers"
	"github.com/Gasoid/merge-bot/logger"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/dustin/go-humanize"
)

func init() {
	handlers.Register("gitlab", New)

	config.StringVar(&gitlabToken, "gitlab-token", "", "in order to communicate with gitlab api, bot needs token (also via GITLAB_TOKEN)")
	config.StringVar(&gitlabURL, "gitlab-url", "", "in case of self-hosted gitlab, you need to set this var up (also via GITLAB_URL)")
	config.StringVar(&maxRepoSize, "gitlab-max-repo-size", "500Mb", "max size of repo in Gb/Mb/Kb, default is 500Mb (also via GITLAB_MAX_REPO_SIZE)")
}

var (
	gitlabToken string
	gitlabURL   string
	maxRepoSize string
)

const (
	tokenUsername = "oauth2"
	gitlabTrue    = true
)

type GitlabProvider struct {
	client *gitlab.Client
	mr     *gitlab.MergeRequest
}

func (g GitlabProvider) loadMR(projectId, mergeId int) (*gitlab.MergeRequest, error) {
	mr, _, err := g.client.MergeRequests.GetMergeRequest(projectId, mergeId, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return nil, err
	}

	return mr, nil
}

func (g GitlabProvider) UpdateFromMaster(projectId, mergeId int) error {
	mr, err := g.loadMR(projectId, mergeId)
	if err != nil {
		return err
	}

	project, _, err := g.client.Projects.GetProject(
		projectId,
		&gitlab.GetProjectOptions{Statistics: gitlab.Ptr(true)},
	)
	if err != nil {
		return err
	}

	bytes, err := humanize.ParseBytes(maxRepoSize)
	if err != nil {
		return err
	}

	if uint64(project.Statistics.RepositorySize) > bytes {
		return handlers.RepoSizeError
	}

	return handlers.MergeMaster(
		tokenUsername,
		gitlabToken,
		project.HTTPURLToRepo,
		mr.SourceBranch,
		mr.TargetBranch,
	)
}

func (g *GitlabProvider) LeaveComment(projectId, mergeId int, message string) error {
	logger.Debug("leaveComment in gitlab", "message", message, "projectId", projectId)

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
	mr, err := g.loadMR(projectId, mergeId)
	if err != nil {
		return false, err
	}

	g.mr = mr

	if g.mr.State != "opened" {
		return false, nil
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
	info := handlers.MrInfo{
		ProjectId: projectId,
		Id:        mergeId,
	}

	info.IsValid, err = g.IsValid(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	info.Labels = g.mr.Labels
	info.TargetBranch = g.mr.TargetBranch
	info.SourceBranch = g.mr.SourceBranch
	info.Author = g.mr.Author.Username

	info.ConfigContent, err = g.GetFile(projectId, configPath)
	if err != nil {
		logger.Debug("i am using default config to validate a request")
	}

	info.Title = g.mr.Title
	info.Description = g.mr.Description
	info.Approvals, err = g.GetApprovals(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	info.FailedPipelines, err = g.GetFailedPipelines()
	if err != nil {
		logger.Debug("GetFailedPipelines returns error, but i am tolerating this issue", "error", err)
		info.FailedPipelines = 1
	}

	if g.mr.HeadPipeline != nil {
		report, _, err := g.client.Pipelines.GetPipelineTestReport(projectId, g.mr.HeadPipeline.IID)
		if err != nil {
			logger.Debug("GetPipelineTestReport returns error, but i am tolerating this issue", "error", err)
			info.FailedTests = 1
		} else {
			info.FailedTests = report.FailedCount
		}
	}

	return &info, nil
}

func (g GitlabProvider) GetVar(projectId int, varName string) (string, error) {
	secretVar, resp, err := g.client.ProjectVariables.GetVariable(projectId, varName, &gitlab.GetProjectVariableOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			logger.Debug("variable not found", "varName", varName, "projectId", projectId)
			return "", nil
		}

		return "", fmt.Errorf("couldn't get variable %s because gitlab instance returns err: %w", varName, err)
	}

	return secretVar.Value, nil
}

func (g GitlabProvider) ListBranches(projectId, size int) ([]handlers.StaleBranch, error) {
	branches, _, err := g.client.Branches.ListBranches(projectId, &gitlab.ListBranchesOptions{})
	if err != nil {
		return nil, err
	}

	staleBranches := make([]handlers.StaleBranch, 0, size)
	for _, b := range branches {
		if b.Default || b.Protected {
			continue
		}

		listMr, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectId,
			&gitlab.ListProjectMergeRequestsOptions{
				SourceBranch: &b.Name,
			})
		if err != nil {
			return nil, err
		}

		if len(listMr) > 0 {
			continue
		}

		staleBranches = append(staleBranches, handlers.StaleBranch{Name: b.Name, LastUpdated: *b.Commit.CreatedAt})
		if len(staleBranches) == size {
			break
		}
	}

	logger.Debug("listBranches", "staleBranches", staleBranches)

	return staleBranches, nil
}

func (g *GitlabProvider) DeleteBranch(projectId int, name string) error {
	_, err := g.client.Branches.DeleteBranch(projectId, name)
	return err
}

func (g GitlabProvider) ListMergeRequests(projectId, size int) ([]handlers.MR, error) {
	listMr, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectId,
		&gitlab.ListProjectMergeRequestsOptions{State: gitlab.Ptr("opened")})
	if err != nil {
		return nil, err
	}

	staleMRS := make([]handlers.MR, 0, size)
	for _, mr := range listMr {
		staleMRS = append(staleMRS, handlers.MR{
			Id:          mr.IID,
			Labels:      mr.Labels,
			Branch:      mr.SourceBranch,
			LastUpdated: *mr.UpdatedAt})
		if len(staleMRS) == size {
			break
		}
	}

	logger.Debug("listMRs", "mrs", staleMRS)

	return staleMRS, nil
}

func (g GitlabProvider) FindMergeRequests(projectId int, targetBranch, label string) ([]handlers.MR, error) {
	listMr, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectId,
		&gitlab.ListProjectMergeRequestsOptions{
			State:        gitlab.Ptr("opened"),
			Labels:       &gitlab.LabelOptions{label},
			TargetBranch: &targetBranch,
		})
	if err != nil {
		return nil, err
	}

	mrs := make([]handlers.MR, 0)
	for _, mr := range listMr {
		mrs = append(mrs, handlers.MR{
			Id:          mr.IID,
			Labels:      mr.Labels,
			Branch:      mr.SourceBranch,
			LastUpdated: *mr.UpdatedAt})
	}

	logger.Debug("FindMergeRequests", "mrs", mrs)

	return mrs, nil
}

func (g GitlabProvider) CreateLabel(projectId int, name, color string) error {
	labels, _, err := g.client.Labels.ListLabels(projectId, &gitlab.ListLabelsOptions{Search: gitlab.Ptr(name)})
	if err != nil {
		return fmt.Errorf("listLabels failed to search: %w", err)
	}

	labelFound := false
	for _, l := range labels {
		if l.Name == name {
			labelFound = true
		}
	}

	if !labelFound {
		if _, _, err := g.client.Labels.CreateLabel(
			projectId,
			&gitlab.CreateLabelOptions{Name: gitlab.Ptr(name), Color: gitlab.Ptr(color)}); err != nil {
			return fmt.Errorf("could't create label: %w", err)
		}
	}
	return nil
}

func (g GitlabProvider) AssignLabel(projectId, mergeId int, name, color string) error {
	mr, _, err := g.client.MergeRequests.GetMergeRequest(projectId, mergeId, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return fmt.Errorf("could't get merge request: %w", err)
	}

	if slices.Contains(mr.Labels, name) {
		return nil
	}

	if err := g.CreateLabel(projectId, name, color); err != nil {
		return err
	}

	if _, _, err := g.client.MergeRequests.UpdateMergeRequest(
		projectId,
		mergeId,
		&gitlab.UpdateMergeRequestOptions{AddLabels: &gitlab.LabelOptions{name}}); err != nil {
		return fmt.Errorf("could't update mergeRequest: %w", err)
	}
	return nil
}

func (g GitlabProvider) RerunPipeline(projectId, pipelineId int, ref string) error {
	pipelineVars, _, err := g.client.Pipelines.GetPipelineVariables(projectId, pipelineId)
	if err != nil {
		return err
	}

	runVars := make([]*gitlab.PipelineVariableOptions, 0, len(pipelineVars))
	for _, v := range pipelineVars {
		runVars = append(runVars, &gitlab.PipelineVariableOptions{Key: &v.Key, Value: &v.Value, VariableType: &v.VariableType})
	}

	if _, _, err := g.client.Pipelines.CreatePipeline(projectId, &gitlab.CreatePipelineOptions{
		Variables: &runVars,
		Ref:       &ref,
	}); err != nil {
		return err
	}

	return nil
}

func New() handlers.RequestProvider {
	var err error
	var p GitlabProvider

	token := gitlabToken
	if token == "" {
		logger.Error("gitlab init", "err", "gitlab requires token, please set env variable GITLAB_TOKEN")
		return nil
	}

	urlInstance := gitlabURL

	if urlInstance != "" {
		p.client, err = gitlab.NewClient(token, gitlab.WithBaseURL(urlInstance))
	} else {
		p.client, err = gitlab.NewClient(token)
	}
	if err != nil {
		logger.Error("gitlabProvider new", "err", err)
		return nil
	}

	return &p
}

var (
	_ handlers.RequestProvider = (*GitlabProvider)(nil)
)
