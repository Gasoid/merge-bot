package main

import (
	"mergebot/webhook"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type testWebhookProvider struct {
	isValid   bool
	id        int
	projectID int
	cmd       string
	secret    string
	err       error
}

func (p *testWebhookProvider) IsValid() bool {
	return p.isValid
}

func (p *testWebhookProvider) GetCmd() string {
	return p.cmd
}

func (p *testWebhookProvider) GetID() int {
	return p.id
}

func (p *testWebhookProvider) GetProjectID() int {
	return p.projectID
}

func (p *testWebhookProvider) GetSecret() string {
	return p.secret
}

func newTestProvider() webhook.Provider {
	return &testWebhookProvider{}
}

func (p *testWebhookProvider) ParseRequest(request *http.Request) error {
	return p.err
}

func TestHandler(t *testing.T) {
	e := echo.New()
	// provider := &testWebhookProvider{}
	webhook.Register("test", newTestProvider)
	req := httptest.NewRequest(http.MethodPost, "/mergebot/webhook/test/", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("provider")
	c.SetParamValues("test")
	cErr := e.NewContext(req, rec)
	type args struct {
		c echo.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test",
			args:    args{c: c},
			wantErr: false,
		},
		{
			name:    "TestErr",
			args:    args{c: cErr},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Handler(tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("Handler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
