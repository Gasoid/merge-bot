package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckTitle(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		mrInfo   *MrInfo
		expected bool
	}{
		{
			name: "valid title",
			config: &Config{
				Rules: Rules{TitleRegex: "^feat|fix|docs|style|refactor|test|chore:"},
			},
			mrInfo: &MrInfo{
				Title: "feat: add new feature",
			},
			expected: true,
		},
		{
			name: "invalid title",
			config: &Config{
				Rules: Rules{TitleRegex: "^feat|fix|docs|style|refactor|test|chore:"},
			},
			mrInfo: &MrInfo{
				Title: "invalid title",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkTitle(tt.config, tt.mrInfo)
			assert.True(t, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckDescription(t *testing.T) {
	tests := []struct {
		name               string
		config             *Config
		mrInfo             *MrInfo
		expected           bool
		expectedApplicable bool
	}{
		{
			name: "non-empty description when required",
			config: &Config{
				Rules: Rules{AllowEmptyDescription: false},
			},
			mrInfo: &MrInfo{
				Description: "This is a description",
			},
			expected:           true,
			expectedApplicable: true,
		},
		{
			name: "empty description when not allowed",
			config: &Config{
				Rules: Rules{AllowEmptyDescription: false},
			},
			mrInfo: &MrInfo{
				Description: "",
			},
			expected:           false,
			expectedApplicable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkDescription(tt.config, tt.mrInfo)
			assert.Equal(t, tt.expectedApplicable, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckApprovals(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		mrInfo   *MrInfo
		expected bool
	}{
		{
			name: "sufficient approvals",
			config: &Config{
				Rules: Rules{MinApprovals: 2},
			},
			mrInfo: &MrInfo{
				Approvals: map[string]struct{}{"user1": {}, "user2": {}},
			},
			expected: true,
		},
		{
			name: "insufficient approvals",
			config: &Config{
				Rules: Rules{MinApprovals: 2},
			},
			mrInfo: &MrInfo{
				Approvals: map[string]struct{}{"user1": {}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkApprovals(tt.config, tt.mrInfo)
			assert.True(t, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckApprovers(t *testing.T) {
	tests := []struct {
		name               string
		config             *Config
		mrInfo             *MrInfo
		expected           bool
		expectedApplicable bool
	}{
		{
			name: "all required approvers present",
			config: &Config{
				Rules: Rules{Approvers: []string{"user1", "user2"}},
			},
			mrInfo: &MrInfo{
				Approvals: map[string]struct{}{"user1": {}, "user2": {}, "user3": {}},
			},
			expected:           true,
			expectedApplicable: true,
		},
		{
			name: "missing required approver",
			config: &Config{
				Rules: Rules{Approvers: []string{"user1", "user2"}},
			},
			mrInfo: &MrInfo{
				Approvals: map[string]struct{}{"user1": {}, "user3": {}},
			},
			expected:           false,
			expectedApplicable: true,
		},
		{
			name: "no required approvers configured",
			config: &Config{
				Rules: Rules{Approvers: []string{}},
			},
			mrInfo: &MrInfo{
				Approvals: map[string]struct{}{"user1": {}},
			},
			expected:           true,
			expectedApplicable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkApprovers(tt.config, tt.mrInfo)
			assert.Equal(t, tt.expectedApplicable, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckPipelines(t *testing.T) {
	tests := []struct {
		name               string
		config             *Config
		mrInfo             *MrInfo
		expected           bool
		expectedApplicable bool
	}{
		{
			name: "no failed pipelines",
			config: &Config{
				Rules: Rules{AllowFailingPipelines: false},
			},
			mrInfo: &MrInfo{
				FailedPipelines: 0,
			},
			expected:           true,
			expectedApplicable: true,
		},
		{
			name: "failed pipelines when not allowed",
			config: &Config{
				Rules: Rules{AllowFailingPipelines: false},
			},
			mrInfo: &MrInfo{
				FailedPipelines: 1,
			},
			expected:           false,
			expectedApplicable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkPipelines(tt.config, tt.mrInfo)
			assert.Equal(t, tt.expectedApplicable, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckTests(t *testing.T) {
	tests := []struct {
		name               string
		config             *Config
		mrInfo             *MrInfo
		expected           bool
		expectedApplicable bool
	}{
		{
			name: "no failed tests",
			config: &Config{
				Rules: Rules{AllowFailingTests: false},
			},
			mrInfo: &MrInfo{
				FailedTests: 0,
			},
			expected:           true,
			expectedApplicable: true,
		},
		{
			name: "failed tests when not allowed",
			config: &Config{
				Rules: Rules{AllowFailingTests: false},
			},
			mrInfo: &MrInfo{
				FailedTests: 1,
			},
			expected:           false,
			expectedApplicable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applicable := checkTests(tt.config, tt.mrInfo)
			assert.Equal(t, tt.expectedApplicable, applicable)
			assert.Equal(t, tt.expected, result)
		})
	}
}
