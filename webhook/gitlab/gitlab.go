package gitlab

import (
	"io"
	"net/http"
	"strings"

	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/gasoid/merge-bot/v3/webhook"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

const (
	mergeAction  = "merge"
	openAction   = "open"
	updateAction = "update"
	closeAction  = "close"
)

func init() {
	webhook.Register("gitlab", New)
}

type GitlabProvider struct {
	payload   []byte
	note      string
	noteId    int64
	action    string
	updatedAt string
	projectId int64
	id        int64
	secret    string
	mrEvent   *gitlab.MergeEvent
	commit    string
}

func New() webhook.Provider {
	return &GitlabProvider{}
}

func (g GitlabProvider) GetSecret() string {
	return g.secret
}

func (g *GitlabProvider) ParseRequest(request *http.Request) error {
	var (
		err     error
		ok      bool
		comment *gitlab.MergeCommentEvent
		mr      *gitlab.MergeEvent
	)

	eventHeader := request.Header.Get("X-Gitlab-Event")
	if strings.TrimSpace(eventHeader) == "" {
		return webhook.AuthError
	}

	eventType := gitlab.EventType(eventHeader)

	g.payload, err = io.ReadAll(request.Body)
	if err != nil || len(g.payload) == 0 {
		return webhook.PayloadError
	}

	event, err := gitlab.ParseWebhook(eventType, g.payload)
	if err != nil {
		return webhook.PayloadError
	}

	g.secret = request.Header.Get("X-Gitlab-Token")

	if comment, ok = event.(*gitlab.MergeCommentEvent); ok {
		g.projectId = comment.ProjectID
		g.id = comment.MergeRequest.IID
		g.note = comment.ObjectAttributes.Note
		g.noteId = comment.ObjectAttributes.ID
		return nil
	}

	if mr, ok = event.(*gitlab.MergeEvent); ok {
		g.projectId = mr.Project.ID
		g.id = mr.ObjectAttributes.IID
		g.updatedAt = mr.ObjectAttributes.UpdatedAt
		g.action = mr.ObjectAttributes.Action
		g.mrEvent = mr
		g.commit = mr.ObjectAttributes.LastCommit.ID
		return nil
	}

	return nil
}

func (g *GitlabProvider) GetCmd() string {
	logger.Debug("getCmd", "action", g.action)

	switch g.action {
	case mergeAction:
		return webhook.OnMerge
	case openAction:
		return webhook.OnNewMR
	case updateAction:
		return webhook.OnUpdate
	case closeAction:
		return webhook.OnClose
	}

	logger.Debug("getCmd", "note", g.note)
	if strings.HasPrefix(g.note, "!") {
		return g.note
	}
	return ""
}

func (g *GitlabProvider) IsReviewersChanged() bool {
	return g.mrEvent != nil && (len(g.mrEvent.Changes.Reviewers.Current) > 0 || len(g.mrEvent.Changes.Reviewers.Previous) > 0)
}

func (g *GitlabProvider) GetCurrentReviewers() []int64 {
	if g.mrEvent == nil {
		return nil
	}

	reviewers := make([]int64, 0, len(g.mrEvent.Changes.Reviewers.Current))
	for _, r := range g.mrEvent.Changes.Reviewers.Current {
		reviewers = append(reviewers, r.ID)
	}
	return reviewers
}

func (g *GitlabProvider) GetPreviousReviewers() []int64 {
	if g.mrEvent == nil {
		return nil
	}

	reviewers := make([]int64, 0, len(g.mrEvent.Changes.Reviewers.Previous))
	for _, r := range g.mrEvent.Changes.Reviewers.Previous {
		reviewers = append(reviewers, r.ID)
	}

	return reviewers
}

func (g *GitlabProvider) GetCommitID() string {
	return g.commit
}

func (g *GitlabProvider) GetID() int64 {
	return g.id
}

func (g *GitlabProvider) GetProjectID() int64 {
	return g.projectId
}

func (g *GitlabProvider) GetNoteID() int64 {
	return g.noteId
}

var (
	_ webhook.Provider = (*GitlabProvider)(nil)
)
