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
	// Fields for tracking comment behavior
	commentCalled   bool
	lastComment     string
	leaveCommentErr error
}

func newTestProvider() RequestProvider {
	return &testProvider{}
}

func (p *testProvider) LeaveComment(projectId, id int, message string) error {
	p.commentCalled = true
	p.lastComment = message
	if p.leaveCommentErr != nil {
		return p.leaveCommentErr
	}
	return p.err
}

func (p *testProvider) Merge(projectId, id int, message string) error {
	return p.err
}

func (p *testProvider) ListBranches(projectId, size int) ([]Branch, error) {
	return nil, nil
}

func (p *testProvider) DeleteBranch(projectId int, name string) error {
	return nil
}

func (p *testProvider) GetVar(projectId int, varName string) (string, error) {
	return "test", nil
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
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {title_regex: '^[A-Z]+-[0-9]+'}", approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "opened", title: "asd-123"}}},
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
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {allow_failing_pipelines: false}", failedPipelines: 2, state: "opened", title: "DEVOPS-123", approvals: map[string]struct{}{"user1": {}}}}},
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

func TestRequest_Greetings(t *testing.T) {
	type fields struct {
		provider RequestProvider
	}
	type args struct {
		projectId int
		id        int
	}
	tests := []struct {
		name              string
		fields            fields
		args              args
		wantErr           bool
		wantCommentCalled bool
		expectedComment   string
	}{
		{
			name: "greetings disabled - should not leave comment",
			fields: fields{
				provider: &testProvider{
					title:  "Test MR",
					config: `greetings: {enabled: false}`,
					state:  "opened",
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           false,
			wantCommentCalled: false,
		},
		{
			name: "greetings enabled with default template - should leave comment",
			fields: fields{
				provider: &testProvider{
					title:  "Test MR",
					config: `greetings: {enabled: true}`,
					state:  "opened",
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           false,
			wantCommentCalled: true,
			expectedComment:   "Requirements:\n - Min approvals: 1\n - Title regex: .*\n\nOnce you've done, send **!merge** command and i will merge it!",
		},
		{
			name: "greetings enabled with custom template - should leave comment",
			fields: fields{
				provider: &testProvider{
					title: "Test MR",
					config: `greetings:
  enabled: true
  template: "Hello! You need {{ .MinApprovals }} approvals."`,
					state: "opened",
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           false,
			wantCommentCalled: true,
			expectedComment:   "Hello! You need 1 approvals.",
		},
		{
			name: "invalid template - should return error",
			fields: fields{
				provider: &testProvider{
					title: "Test MR",
					config: `greetings:
  enabled: true
  template: "Invalid template {{ .NonExistentField }"`,
					state: "opened",
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           true,
			wantCommentCalled: false,
		},
		{
			name: "provider error on GetMRInfo - should return error",
			fields: fields{
				provider: &testProvider{
					err: assert.AnError,
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           true,
			wantCommentCalled: false,
		},
		{
			name: "provider error on LeaveComment - should return error",
			fields: fields{
				provider: &testProvider{
					title:           "Test MR",
					config:          `greetings: {enabled: true}`,
					state:           "opened",
					leaveCommentErr: assert.AnError,
				},
			},
			args:              args{projectId: 1, id: 1},
			wantErr:           true,
			wantCommentCalled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the provider's comment tracking
			if tp, ok := tt.fields.provider.(*testProvider); ok {
				tp.commentCalled = false
				tp.lastComment = ""
			}

			r := &Request{
				provider: tt.fields.provider,
			}
			err := r.Greetings(tt.args.projectId, tt.args.id)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tp, ok := tt.fields.provider.(*testProvider); ok {
				assert.Equal(t, tt.wantCommentCalled, tp.commentCalled, "Expected comment called status mismatch")
				if tt.wantCommentCalled && tt.expectedComment != "" {
					assert.Equal(t, tt.expectedComment, tp.lastComment, "Expected comment content mismatch")
				}
			}
		})
	}
}
