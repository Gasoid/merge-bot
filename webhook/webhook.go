package webhook

import (
	"net/http"
	"strings"
	"sync"
)

const (
	OnNewMR     = "\anewMREvent"
	OnMerge     = "\amergeEvent"
	OnUpdate    = "\aupdateEvent"
	OnCommit    = "\acommitEvent"
	spaceSymbol = " "
)

var (
	providers   = map[string]func() Provider{}
	providersMu sync.RWMutex
	AuthError   = &Error{text: "credentials or headers are wrong"}
	// SignatureError = &Error{text: "signature is wrong"}
	PayloadError = &Error{text: "post body is wrong"}
)

func Register(name string, constructor func() Provider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = constructor
}

type Error struct {
	text string
	// err  error
}

func (e *Error) Error() string {
	return e.text
}

type Provider interface {
	GetCmd() string
	GetID() int
	GetProjectID() int
	ParseRequest(request *http.Request) error
	GetSecret() string
	GetNoteID() int
}

type Webhook struct {
	provider Provider
	Event    string
	Args     string
	NoteID   int
}

func (w Webhook) GetSecret() string {
	return w.provider.GetSecret()
}

func (w *Webhook) GetCmd() string {
	return w.provider.GetCmd()
}

func (w *Webhook) GetID() int {
	return w.provider.GetID()
}

func (w *Webhook) GetProjectID() int {
	return w.provider.GetProjectID()
}

func (w *Webhook) ParseRequest(request *http.Request) error {
	if request == nil {
		return &Error{text: "Request is not provided"}
	}

	if err := w.provider.ParseRequest(request); err != nil {
		return err
	}

	if w.provider.GetCmd() != "" {
		result := strings.SplitN(w.provider.GetCmd(), spaceSymbol, 2)
		if len(result) > 0 {
			w.Event = result[0]
		}

		if len(result) > 1 {
			w.Args = strings.TrimSpace(result[1])
		}

		w.NoteID = w.provider.GetNoteID()
	}

	return nil
}

func New(providerName string) (*Webhook, error) {
	var (
		constructor func() Provider
		ok          bool
	)

	providersMu.Lock()
	defer providersMu.Unlock()

	if constructor, ok = providers[providerName]; !ok {
		return nil, &Error{text: "Provider is not registered"}
	}

	webhook := constructor()
	if webhook == nil {
		return nil, &Error{text: "Provider is nil"}
	}

	return &Webhook{provider: webhook}, nil
}
