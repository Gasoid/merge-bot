package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// type testConfig struct{}

// func (c *testConfig) ParseVars(varMap map[string]string) {
// }

type testProvider struct {
	err             error
	approvals       map[string]struct{}
	failedPipelines int
	state           string
	title           string
	config          string
}

func newTestProvider() RequestProvider {
	return &testProvider{}
}

func (p *testProvider) LeaveComment(projectId, id int, message string) error {
	return p.err
}

func (p *testProvider) Merge(projectId, id int, message string) error {
	return p.err
}

func (p *testProvider) ListBranches(projectId int) ([]Branch, error) {
	return nil, nil
}

func (p *testProvider) DeleteBranch(projectId int, name string) error {
	return nil
}

func (p *testProvider) GetMRInfo(projectId, id int, path string) (*MrInfo, error) {
	return &MrInfo{
		Title:           p.title,
		ConfigContent:   p.config,
		Approvals:       p.approvals,
		FailedPipelines: p.failedPipelines,
		IsValid:         p.IsValid(),
	}, p.err
}

func (p *testProvider) IsValid() bool {
	if p.err != nil {
		return false
	}
	if p.state != "opened" {
		return false
	}
	return true
}

func (p *testProvider) UpdateFromMaster(projectId, mergeId int) error {
	return nil
}

func Test_Merge(t *testing.T) {
	// config.New(&testConfig{})
	type args struct {
		pr *Request
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "should not fail",
			args:    args{pr: &Request{provider: &testProvider{approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "opened", title: "DEVOPS-123"}}},
			wantErr: false,
		},
		{
			name:    "should fail because of title",
			args:    args{pr: &Request{provider: &testProvider{config: "title_regex: ^[A-Z]+-[0-9]+", approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "opened", title: "asd-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of approvals",
			args:    args{pr: &Request{provider: &testProvider{approvals: map[string]struct{}{}, state: "opened", title: "DEVOPS-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of closed state",
			args:    args{pr: &Request{provider: &testProvider{approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "closed", title: "DEVOPS-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of failed pipelines",
			args:    args{pr: &Request{provider: &testProvider{config: "allow_failing_pipelines: false", failedPipelines: 2, state: "opened", title: "DEVOPS-123", approvals: map[string]struct{}{"user1": {}}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, s, _ := tt.args.pr.Merge(1, 2)
			if tt.wantErr {
				assert.NotEmpty(t, s)
				assert.Equal(t, false, ok)
			} else {
				assert.Equal(t, true, ok)
			}
		})
	}
}
