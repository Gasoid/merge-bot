package handlers

import (
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

// type testConfig struct{}

// func (c *testConfig) ParseVars(varMap map[string]string) {
// }

type testProvider struct {
	err             error
	approvals       map[string]struct{}
	failedPipelines int64
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

func (p *testProvider) LeaveComment(projectID, id int64, message string) error {
	p.commentCalled = true
	p.lastComment = message
	if p.leaveCommentErr != nil {
		return p.leaveCommentErr
	}
	return p.err
}

func (p *testProvider) Merge(projectID, id int64, message string) error {
	return p.err
}

func (p *testProvider) ListBranches(projectID, size int64, protected bool) iter.Seq[StaleBranch] {
	return nil
}

func (p *testProvider) DeleteBranch(projectID int64, name string) error {
	return nil
}

func (p *testProvider) GetVar(projectID int64, varName string) (string, error) {
	return "test", nil
}

func (p *testProvider) GetMRInfo(projectID, id int64, path string) (*MrInfo, error) {
	return &MrInfo{
		ProjectID:       projectID,
		ID:              id,
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

func (p *testProvider) UpdateFromMaster(projectID, mergeID int64) error {
	return nil
}

func (p *testProvider) ListMergeRequests(projectID, size int64, protected bool) iter.Seq[MR] {
	return nil
}

func (p *testProvider) FindMergeRequests(projectID int64, targetBranch, label string) ([]MR, error) {
	return nil, p.err
}

func (p *testProvider) AssignLabel(projectID, mergeID int64, name, color string) error {
	return p.err
}

func (p *testProvider) CreateLabel(projectID int64, name, color string) error {
	return p.err
}

func (p *testProvider) RerunPipeline(projectID, pipelineID int64, ref string) (string, error) {
	return "", p.err
}

func (p *testProvider) CreateDiscussion(projectID, mergeID int64, message string) error {
	return p.err
}

func (p *testProvider) UnresolveDiscussion(projectID, mergeID int64) error {
	return p.err
}

func (p *testProvider) GetRawDiffs(projectID, mergeID int64) ([]byte, error) {
	return nil, p.err
}

func (p *testProvider) CreateThreadInLine(projectID, mergeID int64, thread Thread) error {
	return p.err
}

func (p *testProvider) AwardEmoji(projectID, mergeID, noteID int64, emoji string) error {
	return p.err
}

func (p *testProvider) GetFile(projectID int64, path string) ([]byte, error) {
	return nil, p.err
}

func (p *testProvider) GetChangedFiles(projectID, mergeID int64) ([]string, error) {
	return nil, p.err
}

func (p *testProvider) AssignReviewers(projectID, mergeID int64, users []string) error {
	return p.err
}

func (p testProvider) IsHealthy() bool {
	return true
}

func (p testProvider) GetContributors(projectID, mergeID int64) ([]Candidate, error) {
	return nil, p.err
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
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {approvers: []}", approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "opened", title: "DEVOPS-123"}}},
			wantErr: false,
		},
		{
			name:    "should fail because of title",
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {title_regex: '^[A-Z]+-[0-9]+', approvers: []}", approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "opened", title: "asd-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of approvals",
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {approvers: []}", approvals: map[string]struct{}{}, state: "opened", title: "DEVOPS-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of closed state",
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {approvers: []}", approvals: map[string]struct{}{"user1": {}}, failedPipelines: 0, state: "closed", title: "DEVOPS-123"}}},
			wantErr: true,
		},
		{
			name:    "should fail because of failed pipelines",
			args:    args{pr: &Request{provider: &testProvider{config: "rules: {allow_failing_pipelines: false, approvers: []}", failedPipelines: 2, state: "opened", title: "DEVOPS-123", approvals: map[string]struct{}{"user1": {}}}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load info and config first
			err := tt.args.pr.LoadInfoAndConfig(1, 2)
			if err != nil {
				t.Fatalf("LoadInfoAndConfig failed: %v", err)
			}

			ok, s, _ := tt.args.pr.Merge()
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
		projectID int64
		id        int64
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
			args:              args{projectID: 1, id: 1},
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
			args:              args{projectID: 1, id: 1},
			wantErr:           false,
			wantCommentCalled: true,
			expectedComment:   "Requirements:\n - Min approvals: 1\n - Title regex: .*\n\nOnce you're done, send **!merge** command and I will merge it!",
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
			args:              args{projectID: 1, id: 1},
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
			args:              args{projectID: 1, id: 1},
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
			args:              args{projectID: 1, id: 1},
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
			args:              args{projectID: 1, id: 1},
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

			// Load info and config first (this is required for the current implementation)
			err := r.LoadInfoAndConfig(tt.args.projectID, tt.args.id)
			if err != nil && !tt.wantErr {
				t.Fatalf("LoadInfoAndConfig failed: %v", err)
			}

			if err == nil {
				err = r.Greetings()
			}

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
