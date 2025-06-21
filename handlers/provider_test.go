package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	Register("test", newTestProvider)
	type args struct {
		providerName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "ok",
			args:    args{providerName: "test"},
			wantErr: false,
		},
		{
			name:    "notOk",
			args:    args{providerName: "test1"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.providerName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequest_ParseConfig(t *testing.T) {
	type fields struct {
		provider RequestProvider
	}
	type args struct {
		projectId int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				provider: &testProvider{title: "hi"},
			},
			args: args{projectId: 1},
			want: &Config{Rules: Rules{
				MinApprovals:          1,
				AllowFailingPipelines: true,
				AllowFailingTests:     true,
			},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				provider: tt.fields.provider,
			}
			got, err := r.ParseConfig("")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want.Rules.MinApprovals, got.Rules.MinApprovals)
		})
	}
}
