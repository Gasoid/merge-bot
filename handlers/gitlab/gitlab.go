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

	"github.com/gasoid/merge-bot/v3/cache"
	"github.com/gasoid/merge-bot/v3/config"
	"github.com/gasoid/merge-bot/v3/handlers"
	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/hairyhenderson/go-codeowners"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"

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
	currentUserID int64
}

func (g GitlabProvider) loadMR(projectID, mergeID int64) (*gitlab.MergeRequest, error) {
	mr, _, err := g.client.MergeRequests.GetMergeRequest(projectID, mergeID, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return nil, err
	}

	return mr, nil
}

func (g GitlabProvider) UpdateFromMaster(projectID, mergeID int64) error {
	mr, err := g.loadMR(projectID, mergeID)
	if err != nil {
		return err
	}

	project, _, err := g.client.Projects.GetProject(
		projectID,
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

func (g GitlabProvider) findDiscussion(projectID, mergeID int64) (string, string, int64, error) {
	discussions, _, err := g.client.Discussions.ListMergeRequestDiscussions(
		projectID,
		mergeID,
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

		if note.Author.ID != g.currentUserID {
			continue
		}

		return d.ID, note.Body, note.ID, nil
	}

	logger.Info("could not find resolvable discussion", "merge request", mergeID, "project", projectID)

	return "", "", 0, handlers.DiscussionError
}

func (g GitlabProvider) UpdateDiscussion(projectID, mergeID int64, message string) error {
	discussionId, body, noteId, err := g.findDiscussion(projectID, mergeID)
	if err != nil {
		return err
	}

	if body == message {
		return nil
	}

	_, _, err = g.client.Discussions.UpdateMergeRequestDiscussionNote(
		projectID,
		mergeID,
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

func (g GitlabProvider) UnresolveDiscussion(projectID, mergeID int64) error {
	discussionId, _, noteId, err := g.findDiscussion(projectID, mergeID)
	if err != nil {
		return err
	}

	_, _, err = g.client.Discussions.UpdateMergeRequestDiscussionNote(
		projectID,
		mergeID,
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

func (g GitlabProvider) CreateDiscussion(projectID, mergeID int64, message string) error {
	logger.Debug("createDiscussion in gitlab", "message", message, "projectId", projectID)

	_, _, err := g.client.Discussions.CreateMergeRequestDiscussion(
		projectID,
		mergeID,
		&gitlab.CreateMergeRequestDiscussionOptions{
			Body: &message,
		},
	)
	return err
}

func (g *GitlabProvider) LeaveComment(projectID, mergeID int64, message string) error {
	logger.Debug("leaveComment in gitlab", "message", message, "projectId", projectID)

	_, _, err := g.client.Notes.CreateMergeRequestNote(
		projectID,
		mergeID,
		&gitlab.CreateMergeRequestNoteOptions{Body: &message},
	)

	return err
}

func (g *GitlabProvider) AwardEmoji(projectID, mergeID, noteID int64, emoji string) error {
	_, _, err := g.client.AwardEmoji.CreateMergeRequestAwardEmojiOnNote(
		projectID, mergeID, noteID,
		&gitlab.CreateAwardEmojiOptions{
			Name: emoji,
		})

	return err
}

func (g *GitlabProvider) Merge(projectID, mergeID int64, message string) error {
	t := true
	_, _, err := g.client.MergeRequests.AcceptMergeRequest(projectID,
		mergeID,
		&gitlab.AcceptMergeRequestOptions{Squash: &t, ShouldRemoveSourceBranch: &t, SquashCommitMessage: &message},
	)

	return err
}

func (g *GitlabProvider) GetApprovals(projectID, mergeID int64) (map[string]struct{}, error) {

	approvals := map[string]struct{}{}
	approvalsState, _, err := g.client.MergeRequests.GetMergeRequestApprovals(projectID, mergeID)
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

func (g *GitlabProvider) GetFailedPipelines() (int64, error) {
	if g.mr.HeadPipeline != nil && g.mr.HeadPipeline.Status != string(gitlab.DeploymentStatusSuccess) {
		return 1, nil
	}

	return 0, nil
}

func (g *GitlabProvider) IsValid(projectID, mergeID int64) (bool, error) {
	mr, err := g.loadMR(projectID, mergeID)
	if err != nil {
		return false, err
	}

	g.mr = mr

	if g.mr.State != "opened" {
		return false, nil
	}

	return !g.mr.HasConflicts, nil
}

func (g *GitlabProvider) GetFile(projectID int64, path string) ([]byte, error) {
	project, _, err := g.client.Projects.GetProject(projectID, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, err
	}

	gitlabFile, _, err := g.client.RepositoryFiles.GetFile(projectID, path, &gitlab.GetFileOptions{Ref: &project.DefaultBranch})
	if err != nil {
		return nil, err
	}

	content, err := b64.StdEncoding.DecodeString(gitlabFile.Content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (g *GitlabProvider) GetMRInfo(projectID, mergeID int64, configPath string) (*handlers.MrInfo, error) {
	var err error
	info := handlers.MrInfo{
		ProjectID: projectID,
		ID:        mergeID,
	}

	info.IsValid, err = g.IsValid(projectID, mergeID)
	if err != nil {
		return nil, err
	}

	info.Labels = g.mr.Labels
	info.TargetBranch = g.mr.TargetBranch
	info.SourceBranch = g.mr.SourceBranch
	info.Author = g.mr.Author.Username

	for _, r := range g.mr.Reviewers {
		info.Reviewers = append(info.Reviewers, r.Username)
	}

	b, err := g.GetFile(projectID, configPath)
	if err != nil {
		logger.Debug("i am using default config to validate a request")
		info.ConfigContent = ""
	} else {
		info.ConfigContent = string(b)
	}

	info.Title = g.mr.Title
	info.Description = g.mr.Description
	info.Approvals, err = g.GetApprovals(projectID, mergeID)
	if err != nil {
		return nil, err
	}

	info.FailedPipelines, err = g.GetFailedPipelines()
	if err != nil {
		logger.Debug("GetFailedPipelines returns error, but i am tolerating this issue", "error", err)
		info.FailedPipelines = 1
	}

	if g.mr.HeadPipeline != nil {
		report, _, err := g.client.Pipelines.GetPipelineTestReport(projectID, g.mr.HeadPipeline.IID)
		if err != nil {
			logger.Debug("GetPipelineTestReport returns error, but i am tolerating this issue", "error", err)
			info.FailedTests = 1
		} else {
			info.FailedTests = report.FailedCount
		}
	}

	return &info, nil
}

func (g GitlabProvider) GetVar(projectID int64, varName string) (string, error) {
	secretVar, resp, err := g.client.ProjectVariables.GetVariable(projectID, varName, &gitlab.GetProjectVariableOptions{})
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			logger.Debug("variable not found", "varName", varName, "projectId", projectID)
			return "", nil
		}

		return "", fmt.Errorf("couldn't get variable %s because gitlab instance returns err: %w", varName, err)
	}

	return secretVar.Value, nil
}

func (g GitlabProvider) ListBranches(projectID, size int64, protected bool) iter.Seq[handlers.StaleBranch] {

	return func(yield func(handlers.StaleBranch) bool) {
		for b := range g.listBranches(projectID, size) {
			if b.Default {
				continue
			}

			listMr, _, err := g.client.MergeRequests.ListProjectMergeRequests(projectID,
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

func (g *GitlabProvider) DeleteBranch(projectID int64, name string) error {
	_, err := g.client.Branches.DeleteBranch(projectID, name)
	return err
}

func (g GitlabProvider) ListMergeRequests(projectID, size int64, protected bool) iter.Seq[handlers.MR] {
	listMr := g.listMergeRequests(projectID, size,
		&gitlab.ListProjectMergeRequestsOptions{
			State:   new("opened"),
			OrderBy: new("updated_at"),
			Sort:    new("asc"),
		})

	return func(yield func(handlers.MR) bool) {
		for mr := range listMr {
			b, _, err := g.client.Branches.GetBranch(projectID, mr.SourceBranch)
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
				ID:          mr.IID,
				Labels:      mr.Labels,
				Branch:      mr.SourceBranch,
				Protected:   b.Protected,
				LastUpdated: *mr.UpdatedAt}) {
				return
			}
		}
	}
}

func (g GitlabProvider) FindMergeRequests(projectID int64, targetBranch, label string) ([]handlers.MR, error) {
	mrs := make([]handlers.MR, 0)

	listMr := g.listMergeRequests(projectID, findMRSize,
		&gitlab.ListProjectMergeRequestsOptions{
			State:        new("opened"),
			Labels:       &gitlab.LabelOptions{label},
			TargetBranch: &targetBranch,
		})

	for mr := range listMr {
		mrs = append(mrs, handlers.MR{
			ID:          mr.IID,
			Labels:      mr.Labels,
			Branch:      mr.SourceBranch,
			LastUpdated: *mr.UpdatedAt})
	}

	logger.Debug("FindMergeRequests", "mrs", mrs)

	return mrs, nil
}

func (g GitlabProvider) CreateLabel(projectID int64, name, color string) error {
	labels, _, err := g.client.Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{Search: new(name)})
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
			projectID,
			&gitlab.CreateLabelOptions{Name: new(name), Color: new(color)}); err != nil {
			return fmt.Errorf("could't create label: %w", err)
		}
	}
	return nil
}

func (g GitlabProvider) AssignLabel(projectID, mergeID int64, name, color string) error {
	mr, _, err := g.client.MergeRequests.GetMergeRequest(projectID, mergeID, &gitlab.GetMergeRequestsOptions{})
	if err != nil {
		return fmt.Errorf("could't get merge request: %w", err)
	}

	if slices.Contains(mr.Labels, name) {
		return nil
	}

	if err := g.CreateLabel(projectID, name, color); err != nil {
		return err
	}

	if _, _, err := g.client.MergeRequests.UpdateMergeRequest(
		projectID,
		mergeID,
		&gitlab.UpdateMergeRequestOptions{AddLabels: &gitlab.LabelOptions{name}}); err != nil {
		return fmt.Errorf("could't update mergeRequest: %w", err)
	}
	return nil
}

func (g GitlabProvider) RerunPipeline(projectID, pipelineID int64, ref string) (string, error) {
	pipelineVars, _, err := g.client.Pipelines.GetPipelineVariables(projectID, pipelineID)
	if err != nil {
		return "", err
	}

	runVars := make([]*gitlab.PipelineVariableOptions, 0, len(pipelineVars))
	for _, v := range pipelineVars {
		runVars = append(runVars, &gitlab.PipelineVariableOptions{Key: &v.Key, Value: &v.Value, VariableType: &v.VariableType})
	}

	pipeline, _, err := g.client.Pipelines.CreatePipeline(projectID, &gitlab.CreatePipelineOptions{
		Variables: &runVars,
		Ref:       &ref,
	})
	if err != nil {
		return "", err
	}

	return pipeline.WebURL, nil
}

func (g GitlabProvider) GetRawDiffs(projectID, mergeID int64) ([]byte, error) {
	result, _, err := g.client.MergeRequests.ShowMergeRequestRawDiffs(projectID, mergeID, &gitlab.ShowMergeRequestRawDiffsOptions{})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (g GitlabProvider) getChangedFiles(projectID, mergeID int64) ([]string, error) {
	result, _, err := g.client.MergeRequests.ListMergeRequestDiffs(projectID, mergeID, &gitlab.ListMergeRequestDiffsOptions{})
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

func (g GitlabProvider) codeOwners(projectID, mergeID int64) (map[string]struct{}, error) {
	candidates := map[string]struct{}{}

	b, err := g.GetFile(projectID, "CODEOWNERS")
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	changedFiles, err := g.getChangedFiles(projectID, mergeID)
	if err != nil {
		return nil, err
	}

	for _, f := range changedFiles {
		owners, err := codeowners.FromReader(bytes.NewReader(b), "")
		if err != nil {
			return nil, err
		}

		// it seems it is not right logic
		// TODO: rewrite it to more desirable logic
		for _, o := range owners.Owners(f) {
			candidates[o] = struct{}{}
		}
	}

	return candidates, nil
}

func (g GitlabProvider) AssignReviewers(projectID, mergeID int64, users []string) error {
	logger.Debug("AssignReviewers started", "users", users)

	usersIDs := []int64{}

	for _, u := range users {
		listUsers, _, err := g.client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &u})
		if err != nil {
			return err
		}

		if len(listUsers) == 1 {
			usersIDs = append(usersIDs, listUsers[0].ID)
		}
	}

	logger.Debug("UpdateMergeRequest", "users", usersIDs)
	_, _, err := g.client.MergeRequests.UpdateMergeRequest(projectID, mergeID, &gitlab.UpdateMergeRequestOptions{
		ReviewerIDs: &usersIDs,
	})

	return err
}

func (g GitlabProvider) GetContributors(projectID, mergeID int64) ([]handlers.Candidate, error) {
	const (
		batch int64 = 50
	)

	candidates := []handlers.Candidate{}

	userIDs, err := cache.GetContributors(projectID)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}

	if len(userIDs) == 0 {
		now := time.Now()
		months3back := now.Add(-1 * time.Hour * 24 * 30 * 3)

		for mr := range g.listMergeRequests(projectID, batch, &gitlab.ListProjectMergeRequestsOptions{
			UpdatedAfter: &months3back,
		}) {
			userIDs = append(userIDs, mr.Author.ID)
			for _, r := range mr.Reviewers {
				counts[r.Username]++
			}
		}

		seen := make(map[int64]struct{}, 10)

		for _, id := range userIDs {
			seen[id] = struct{}{}
		}

		uniqueIDs := make([]int64, 0, len(seen))
		for k := range seen {
			uniqueIDs = append(uniqueIDs, k)
		}

		if err := cache.SetContributors(projectID, uniqueIDs); err != nil {
			return nil, err
		}

		userIDs = uniqueIDs
	}

	codeowners, err := g.codeOwners(projectID, mergeID)
	if err != nil {
		return nil, err
	}

	for m := range g.listAllProjectMembers(projectID, batch, &gitlab.ListProjectMembersOptions{
		UserIDs: &userIDs,
	}) {
		if m.AccessLevel < gitlab.MaintainerPermissions {
			continue
		}

		if m.State != "active" {
			continue
		}

		status, _, err := g.client.Users.GetUserStatus(m.ID)
		if err != nil {
			logger.Error("GetUserStatus", "err", err)
			continue
		}

		// user, _, err := g.client.Users.GetUser(m.ID, &gitlab.GetUserOptions{})
		// if err != nil {
		// 	continue
		// }

		_, isCodeOwner := codeowners[m.Username]

		candidates = append(candidates, handlers.Candidate{
			Username:    m.Username,
			StatusEmoji: status.Emoji,
			Status:      status.Message,
			Count:       counts[m.Username],
			// Timezone:    user.Location,
			IsCodeOwner: isCodeOwner})
	}

	return candidates, nil
}

func (g GitlabProvider) CreateThreadInLine(projectID, mergeID int64, thread handlers.Thread) error {
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
		projectID, mergeID,
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

	p.currentUserID = user.ID
	return &p
}

var (
	_ handlers.RequestProvider = (*GitlabProvider)(nil)
)
