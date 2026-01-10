package gitlab

import (
	"io"
	"net/http"
	"strings"

	"github.com/gasoid/merge-bot/logger"
	"github.com/gasoid/merge-bot/webhook"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	mergeAction  = "merge"
	openAction   = "open"
	updateAction = "update"
	pushAction   = "push"
)

func init() {
	webhook.Register("gitlab", New)
}

type GitlabProvider struct {
	payload   []byte
	note      string
	noteId    int
	action    string
	updatedAt string
	projectId int
	id        int
	secret    string
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

		if mr.ObjectAttributes.OldRev != "" {
			g.action = pushAction
		} else {
			g.action = mr.ObjectAttributes.Action
		}

		g.updatedAt = mr.ObjectAttributes.UpdatedAt
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
	case pushAction:
		return webhook.OnCommit
	}

	logger.Debug("getCmd", "note", g.note)
	if strings.HasPrefix(g.note, "!") {
		return g.note
	}
	return ""
}

func (g *GitlabProvider) GetID() int {
	return g.id
}

func (g *GitlabProvider) GetProjectID() int {
	return g.projectId
}

func (g *GitlabProvider) GetNoteID() int {
	return g.noteId
}

var (
	_ webhook.Provider = (*GitlabProvider)(nil)
)
