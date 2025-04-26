package webhook

import (
	"net/http"
	"sync"
)

const (
	OnNewMR = "\anew_mr"
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
	IsNew() bool
	ParseRequest(request *http.Request) error
}

type Webhook struct {
	provider Provider
	Event    string
}

func (w *Webhook) GetCmd() string {
	return w.provider.GetCmd()
}

func (w *Webhook) IsNew() bool {
	return w.provider.IsNew()
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

	if w.provider.IsNew() {
		w.Event = OnNewMR
		return nil
	}

	w.Event = w.provider.GetCmd()

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
