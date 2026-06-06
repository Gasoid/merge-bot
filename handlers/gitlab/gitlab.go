package gitlab

import (
	"bytes"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"
	"time"

	cache "github.com/gasoid/merge-bot/cache/contributors"
	"github.com/gasoid/merge-bot/config"
	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/logger"
	"github.com/hairyhenderson/go-codeowners"
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
	findMRSize    = 10
	// sortDesc              = "desc"
)

type GitlabProvider struct {
	client        *gitlab.Client
	mr            *gitlab.MergeRequest
	currentUserId int
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
		&gitlab.GetProjectOptions{Statistics: new(true)},
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

func (g GitlabProvider) findDiscussion(projectId, mergeId int) (string, string, int, error) {
	discussions, _, err := g.client.Discussions.ListMergeRequestDiscussions(
		projectId,
		mergeId,
		&gitlab.ListMergeRequestDiscussionsOptions{})
	if err != nil {
		return "", "", 0, err
	}

	for _, d := range discussions {
		if len(d.Notes) == 0 {
			continue
		}

		note := d.Notes[0]
		if !note.Resolvable {
			continue
		}

		if note.Author.ID != g.currentUserId {
			continue
		}

		return d.ID, note.Body, note.ID, nil
	}

	logger.Info("could not find resolvable discussion", "merge request", mergeId, "project", projectId)

	return "", "", 0, handlers.DiscussionError
}

func (g GitlabProvider) UpdateDiscussion(projectId, mergeId int, message string) error {
	discussionId, body, noteId, err := g.findDiscussion(projectId, mergeId)
	if err != nil {
		return err
	}

	if body == message {
		return nil
	}

	_, _, err = g.client.Discussions.UpdateMergeRequestDiscussionNote(
		projectId,
		mergeId,
		discussionId,
		noteId,
		&gitlab.UpdateMergeRequestDiscussionNoteOptions{
			Body: new(message),
		})
	if err != nil {
		return err
	}

	return nil
}

func (g GitlabProvider) UnresolveDiscussion(projectId, mergeId int) error {
	discussionId, _, noteId, err := g.findDiscussion(projectId, mergeId)
	if err != nil {
		return err
	}

	_, _, err = g.client.Discussions.UpdateMergeRequestDiscussionNote(
		projectId,
		mergeId,
		discussionId,
		noteId,
		&gitlab.UpdateMergeRequestDiscussionNoteOptions{
			Resolved: new(false),
		})
	if err != nil {
		return err
	}
	return nil
}

func (g GitlabProvider) CreateDiscussion(projectId, mergeId int, message string) error {
	logger.Debug("createDiscussion in gitlab", "message", message, "projectId", projectId)

	_, _, err := g.client.Discussions.CreateMergeRequestDiscussion(
		projectId,
		mergeId,
		&gitlab.CreateMergeRequestDiscussionOptions{
			Body: &message,
		},
	)
	return err
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

func (g *GitlabProvider) AwardEmoji(projectId, mergeId, noteId int, emoji string) error {
	_, _, err := g.client.AwardEmoji.CreateMergeRequestAwardEmojiOnNote(
		projectId, mergeId, noteId,
		&gitlab.CreateAwardEmojiOptions{
			Name: emoji,
		})

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

	approvals := map[string]struct{}{}
	approvalsState, _, err := g.client.MergeRequests.GetMergeRequestApprovals(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	for _, user := range approvalsState.ApprovedBy {
		if g.mr.Author.ID == user.User.ID {
			continue
		}

		approvals[user.User.Username] = struct{}{}
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

func (g *GitlabProvider) GetFile(projectId int, path string) ([]byte, error) {
	project, _, err := g.client.Projects.GetProject(projectId, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, err
	}

	gitlabFile, _, err := g.client.RepositoryFiles.GetFile(projectId, path, &gitlab.GetFileOptions{Ref: &project.DefaultBranch})
	if err != nil {
		return nil, err
	}

	content, err := b64.StdEncoding.DecodeString(gitlabFile.Content)
	if err != nil {
		return nil, err
	}

	return content, nil
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

	b, err := g.GetFile(projectId, configPath)
	if err != nil {
		logger.Debug("i am using default config to validate a request")
		info.ConfigContent = ""
	} else {
		info.ConfigContent = string(b)
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

func (g GitlabProvider) ListBranches(projectId, size int, protected bool) iter.Seq[handlers.StaleBranch] {

	return func(yield func(handlers.StaleBranch) bool) {
		for b := range g.listBranches(projectId, size) {
			if b.Default {
				continue
			}

			listMr, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectId,
				&gitlab.ListProjectMergeRequestsOptions{
					SourceBranch: &b.Name,
					State:        new("opened"),
				})
			if err != nil {
				logger.Error("ListProjectMergeRequests", "err", err)
				continue
			}

			if len(listMr) > 0 {
				continue
			}

			if !protected {
				if b.Protected {
					continue
				}
			}

			if !yield(handlers.StaleBranch{Name: b.Name, LastUpdated: *b.Commit.CreatedAt, Protected: b.Protected}) {
				return
			}
		}
	}
}

func (g *GitlabProvider) DeleteBranch(projectId int, name string) error {
	_, err := g.client.Branches.DeleteBranch(projectId, name)
	return err
}

func (g GitlabProvider) ListMergeRequests(projectId, size int, protected bool) iter.Seq[handlers.MR] {
	listMr := g.listMergeRequests(projectId, size,
		&gitlab.ListProjectMergeRequestsOptions{
			State:   new("opened"),
			OrderBy: new("updated_at"),
			Sort:    new("asc"),
		})

	return func(yield func(handlers.MR) bool) {
		for mr := range listMr {
			b, _, err := g.client.Branches.GetBranch(projectId, mr.SourceBranch)
			if err != nil {
				logger.Error("GetBranch fails", "err", err)
				continue
			}

			if !protected {
				if b.Protected {
					continue
				}
			}

			if !yield(handlers.MR{
				Id:          mr.IID,
				Labels:      mr.Labels,
				Branch:      mr.SourceBranch,
				Protected:   b.Protected,
				LastUpdated: *mr.UpdatedAt}) {
				return
			}
		}
	}
}

func (g GitlabProvider) FindMergeRequests(projectId int, targetBranch, label string) ([]handlers.MR, error) {
	mrs := make([]handlers.MR, 0)

	listMr := g.listMergeRequests(projectId, findMRSize,
		&gitlab.ListProjectMergeRequestsOptions{
			State:        new("opened"),
			Labels:       &gitlab.LabelOptions{label},
			TargetBranch: &targetBranch,
		})

	for mr := range listMr {
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
	labels, _, err := g.client.Labels.ListLabels(projectId, &gitlab.ListLabelsOptions{Search: new(name)})
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
			&gitlab.CreateLabelOptions{Name: new(name), Color: new(color)}); err != nil {
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

func (g GitlabProvider) RerunPipeline(projectId, pipelineId int, ref string) (string, error) {
	pipelineVars, _, err := g.client.Pipelines.GetPipelineVariables(projectId, pipelineId)
	if err != nil {
		return "", err
	}

	runVars := make([]*gitlab.PipelineVariableOptions, 0, len(pipelineVars))
	for _, v := range pipelineVars {
		runVars = append(runVars, &gitlab.PipelineVariableOptions{Key: &v.Key, Value: &v.Value, VariableType: &v.VariableType})
	}

	pipeline, _, err := g.client.Pipelines.CreatePipeline(projectId, &gitlab.CreatePipelineOptions{
		Variables: &runVars,
		Ref:       &ref,
	})
	if err != nil {
		return "", err
	}

	return pipeline.WebURL, nil
}

func (g GitlabProvider) GetRawDiffs(projectId, mergeId int) ([]byte, error) {
	result, _, err := g.client.MergeRequests.ShowMergeRequestRawDiffs(projectId, mergeId, &gitlab.ShowMergeRequestRawDiffsOptions{})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (g GitlabProvider) getChangedFiles(projectId, mergeId int) ([]string, error) {
	result, _, err := g.client.MergeRequests.ListMergeRequestDiffs(projectId, mergeId, &gitlab.ListMergeRequestDiffsOptions{})
	if err != nil {
		return nil, err
	}

	changedFiles := make([]string, 0, len(result))
	for _, l := range result {
		if l.NewPath == l.OldPath {
			changedFiles = append(changedFiles, l.NewPath)
			continue
		}

		if l.NewPath != "" {
			changedFiles = append(changedFiles, l.NewPath)
		}

		if l.OldPath != "" {
			changedFiles = append(changedFiles, l.OldPath)
		}
	}

	return changedFiles, nil
}

func (g GitlabProvider) codeOwners(projectId, mergeId int) (map[string]struct{}, error) {
	candidates := map[string]struct{}{}

	b, err := g.GetFile(projectId, "CODEOWNERS")
	if err != nil {
		return nil, err
	}

	changedFiles, err := g.getChangedFiles(projectId, mergeId)
	if err != nil {
		return nil, err
	}

	for _, f := range changedFiles {
		owners, err := codeowners.FromReader(bytes.NewReader(b), "")
		if err != nil {
			return nil, err
		}

		for _, o := range owners.Owners(f) {
			candidates[o] = struct{}{}
		}
	}

	return candidates, nil
}

func (g GitlabProvider) AssignReviewers(projectId, mergeId int, users []string) error {
	usersIds := []int{}

	for _, u := range users {
		listUsers, _, err := g.client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &u})
		if err != nil {
			return err
		}

		if len(listUsers) == 1 {
			usersIds = append(usersIds, listUsers[0].ID)
		}
	}

	_, _, err := g.client.MergeRequests.UpdateMergeRequest(projectId, mergeId, &gitlab.UpdateMergeRequestOptions{
		ReviewerIDs: &usersIds,
	})

	return err
}

func (g GitlabProvider) GetContributors(projectId, mergeId int) ([]handlers.Candidate, error) {
	candidates := []handlers.Candidate{}

	emails, err := cache.GetContributors(projectId)
	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		now := time.Now()
		months3back := now.Add(-1 * time.Hour * 24 * 30 * 3)

		commits, _, err := g.client.Commits.ListCommits(projectId, &gitlab.ListCommitsOptions{
			Since: &months3back,
		})
		if err != nil {
			return nil, err
		}

		seen := make(map[string]struct{}, 10)

		for _, c := range commits {
			seen[c.AuthorEmail] = struct{}{}
		}

		emails := make([]string, 0, len(seen))
		for k := range seen {
			emails = append(emails, k)
		}

		if err := cache.SetContributors(projectId, emails); err != nil {
			return nil, err
		}
	}

	for _, e := range emails {
		members, _, err := g.client.ProjectMembers.ListAllProjectMembers(projectId, &gitlab.ListProjectMembersOptions{
			Query: &e,
		})
		if err != nil {
			continue
		}

		if len(members) != 1 {
			continue
		}

		if members[0].AccessLevel >= gitlab.MaintainerPermissions {
			// g.client.MergeRequests.ListProjectMergeRequests(projectId, &gitlab.ListProjectMergeRequestsOptions{
			// 	ReviewerID: ,
			// })
			status, _, err := g.client.Users.GetUserStatus(members[0].ID)
			if err != nil {
				continue
			}

			user, _, err := g.client.Users.GetUser(members[0].ID, gitlab.GetUsersOptions{})
			if err != nil {
				continue
			}

			codeowners, err := g.codeOwners(projectId, mergeId)
			if err != nil {
				continue
			}

			_, isCodeOwner := codeowners[members[0].Username]

			candidates = append(candidates, handlers.Candidate{
				Username:    members[0].Username,
				StatusEmoji: status.Emoji,
				Status:      status.Message,
				Timezone:    user.Location,
				IsCodeOwner: isCodeOwner})
		}
	}

	return candidates, nil
}

func (g GitlabProvider) CreateThreadInLine(projectId, mergeId int, thread handlers.Thread) error {
	if g.mr == nil {
		return errors.New("no mr information")
	}

	position := &gitlab.PositionOptions{
		BaseSHA:      &g.mr.DiffRefs.BaseSha,
		HeadSHA:      &g.mr.DiffRefs.HeadSha,
		StartSHA:     &g.mr.DiffRefs.StartSha,
		PositionType: new("text"),
		NewPath:      &thread.NewPath,
		OldPath:      &thread.OldPath,
	}

	if thread.NewLine == 0 && thread.OldLine == 0 {
		return errors.New("no lines included")
	}

	if thread.NewLine != 0 {
		position.NewLine = &thread.NewLine
	}

	if thread.OldLine != 0 {
		position.OldLine = &thread.OldLine
	}

	_, _, err := g.client.Discussions.CreateMergeRequestDiscussion(
		projectId, mergeId,
		&gitlab.CreateMergeRequestDiscussionOptions{
			Body:     new(thread.Body),
			Position: position,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (g GitlabProvider) IsHealthy() bool {
	version, _, err := g.client.Version.GetVersion()
	if version == nil || err != nil {
		return false
	}

	return true
}

func newGitlabClient(token, instanceUrl string) *gitlab.Client {
	var (
		err error
		c   *gitlab.Client
	)

	if token == "" {
		logger.Error("gitlab init", "err", "gitlab requires token, please set env variable GITLAB_TOKEN")
		return nil
	}

	if instanceUrl != "" {
		c, err = gitlab.NewClient(token, gitlab.WithBaseURL(instanceUrl))
	} else {
		c, err = gitlab.NewClient(token)
	}

	if err != nil {
		logger.Error("gitlabProvider new", "err", err)
		return nil
	}

	return c
}

func New() handlers.RequestProvider {
	var p GitlabProvider

	p.client = newGitlabClient(gitlabToken, gitlabURL)
	user, _, err := p.client.Users.CurrentUser()
	if err != nil {
		logger.Error("gitlab client could not get currentUser", "err", err)
		return nil
	}

	p.currentUserId = user.ID
	return &p
}

var (
	_ handlers.RequestProvider = (*GitlabProvider)(nil)
)
