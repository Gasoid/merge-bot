package webhook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type testProvider struct {
	isNew     bool
	isValid   bool
	id        int
	projectID int
	cmd       string
	secret    string
	err       error
}

func (p *testProvider) IsNew() bool {
	return p.isNew
}

func (p *testProvider) IsValid() bool {
	return p.isValid
}

func (p *testProvider) GetCmd() string {
	return p.cmd
}

func (p *testProvider) GetID() int {
	return p.id
}

func (p *testProvider) GetProjectID() int {
	return p.projectID
}

func (p *testProvider) GetSecret() string {
	return p.secret
}

func newTestProvider() Provider {
	// if p.err != nil {
	// 	return nil
	// }
	return &testProvider{}
}

func (p *testProvider) ParseRequest(request *http.Request) error {
	return p.err
}

func TestNew(t *testing.T) {
	// e := echo.New()
	provider := &testProvider{}
	Register("test", newTestProvider)
	// req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	// req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	// rec := httptest.NewRecorder()
	// c := e.NewContext(req, rec)
	type args struct {
		providerName string
		// request      *http.Request
		providerErr error
	}
	tests := []struct {
		name string
		args args
		//want    *Webhook
		wantErr bool
	}{
		{
			name: "NoProvider",
			args: args{providerName: "noProvider"},
			//want:    nil,
			wantErr: true,
		},
		{
			name: "TestProvider",
			args: args{providerName: "test"},
			//want:    nil,
			wantErr: false,
		},
		// {
		// 	name: "ProviderErr",
		// 	args: args{providerName: "test", providerErr: &Error{}},
		// 	//want:    nil,
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.err = tt.args.providerErr
			_, err := New(tt.args.providerName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseRequest(t *testing.T) {
	provider := &testProvider{}
	Register("test", newTestProvider)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	// rec := httptest.NewRecorder()
	// c := e.NewContext(req, rec)
	type args struct {
		providerName string
		request      *http.Request
		providerErr  error
	}
	tests := []struct {
		name string
		args args
		//want    *Webhook
		wantErr bool
	}{
		{
			name: "Shouldn't fail",
			args: args{providerName: "test", request: req},
			//want:    nil,
			wantErr: false,
		},
		{
			name: "Should fail",
			args: args{providerName: "test", request: nil},
			//want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.err = tt.args.providerErr
			w, _ := New(tt.args.providerName)

			assert.NotNil(t, w)

			err := w.ParseRequest(tt.args.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
