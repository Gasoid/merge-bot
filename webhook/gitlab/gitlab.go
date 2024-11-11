package gitlab

import (
	"io"
	"log/slog"
	"mergebot/webhook"
	"net/http"
	"strings"

	"github.com/xanzy/go-gitlab"
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
}

func New() webhook.Provider {
	return &GitlabProvider{}
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
	slog.Debug("getCmd", "note", g.note)
	if strings.HasPrefix(g.note, "!") {
		return g.note
	}
	return ""
}

func (g *GitlabProvider) IsNew() bool {
	return g.action == "open"
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
