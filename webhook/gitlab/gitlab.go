package gitlab

import (
	"io"
	"net/http"
	"strings"

	"github.com/Gasoid/merge-bot/logger"
	"github.com/Gasoid/merge-bot/webhook"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func init() {
	// config.Register(webhookSecret, "")
	webhook.Register("gitlab", New)
}

type GitlabProvider struct {
	payload   []byte
	note      string
	action    string
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
	var err error
	var ok bool
	var comment *gitlab.MergeCommentEvent
	var mr *gitlab.MergeEvent

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
		return nil
	}

	if mr, ok = event.(*gitlab.MergeEvent); ok {
		g.projectId = mr.Project.ID
		g.id = mr.ObjectAttributes.IID
		g.action = mr.ObjectAttributes.Action
	}

	return nil
}

func (g *GitlabProvider) GetCmd() string {
	logger.Debug("getCmd", "action", g.action)

	if g.action == "merge" {
		return webhook.OnMerge
	}

	if g.action == "open" {
		return webhook.OnNewMR
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

var (
	_ webhook.Provider = (*GitlabProvider)(nil)
)
