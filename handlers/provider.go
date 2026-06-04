package handlers

import (
	"iter"
	"sync"

	"github.com/gasoid/merge-bot/cache/contributors"
)

const (
	configPath = ".mrbot.yaml"
)

var (
	providers   = map[string]func() RequestProvider{}
	providersMu sync.RWMutex

	StatusError         = &Error{"Is it opened?"}
	ValidError          = &Error{"Your request can't be merged, because either it has conflicts or state is not opened"}
	RepoSizeError       = &Error{"Repository size is greater than allowed size"}
	NotFoundError       = &Error{"Resource is not found"}
	DiscussionError     = &Error{"Could not find resolvable discussion for merge request"}
	CommitNotFoundError = &Error{"Commit was not found"}
)

type Error struct {
	text string
}

func (e *Error) Error() string {
	return e.text
}

func Register(name string, constructor func() RequestProvider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[name] = constructor
}

type MrInfo struct {
	ProjectId       int
	Id              int
	Labels          []string
	TargetBranch    string
	SourceBranch    string
	Approvals       map[string]struct{}
	Author          string
	FailedPipelines int
	FailedTests     int
	Title           string
	Description     string
	ConfigContent   string
	IsValid         bool
}

type Branches interface {
	ListBranches(projectId, size int, protected bool) iter.Seq[StaleBranch]
	DeleteBranch(projectId int, name string) error
}

type Comments interface {
	LeaveComment(projectId, mergeId int, message string) error
	AwardEmoji(projectId, mergeId, noteId int, emoji string) error
}

type Discussions interface {
	CreateDiscussion(projectId, mergeId int, message string) error
	UnresolveDiscussion(projectId, mergeId int) error
	CreateThreadInLine(projectId, mergeId int, thread Thread) error
}

type MergeRequest interface {
	Merge(projectId, mergeId int, message string) error
	GetMRInfo(projectId, mergeId int, path string) (*MrInfo, error)
	ListMergeRequests(projectId, size int, protected bool) iter.Seq[MR]
	FindMergeRequests(projectId int, targetBranch, label string) ([]MR, error)
	UpdateFromMaster(projectId, mergeId int) error
	AssignLabel(projectId, mergeId int, name, color string) error
	GetRawDiffs(projectId, mergeId int) ([]byte, error)
	AssignReviewers(projectId, mergeId int, users []string) error
}

type Project interface {
	CreateLabel(projectId int, name, color string) error
	GetVar(projectId int, varName string) (string, error)
	RerunPipeline(projectId, pipelineId int, ref string) (string, error)
	GetFile(projectId int, path string) ([]byte, error)
	IsHealthy() bool
	GetContributors(projectId, mergeId int) ([]Candidate, error)
}

type RequestProvider interface {
	Branches
	Comments
	MergeRequest
	Project
	Discussions
}

type Rules struct {
	MinApprovals          int      `yaml:"min_approvals"`
	Approvers             []string `yaml:"approvers"`
	AllowFailingPipelines bool     `yaml:"allow_failing_pipelines"`
	AllowFailingTests     bool     `yaml:"allow_failing_tests"`
	TitleRegex            string   `yaml:"title_regex"`
	AllowEmptyDescription bool     `yaml:"allow_empty_description"`
}

type AssignReviewers struct {
	Enabled        bool `yaml:"enabled"`
	UseCodeowners  bool `yaml:"use_codeowners"`
	ReviewerNumber int  `yaml:"reviewer_number"`
}

type Config struct {
	Rules Rules `yaml:"rules"`

	Greetings struct {
		Enabled    bool   `yaml:"enabled"`
		Resolvable bool   `yaml:"resolvable"`
		Template   string `yaml:"template"`
	} `yaml:"greetings"`

	AutoMasterMerge bool            `yaml:"auto_master_merge"`
	AssignReviewers AssignReviewers `yaml:"assign_reviewers"`

	StaleBranchesDeletion struct {
		Enabled         bool     `yaml:"enabled"`
		ExcludeBranches []string `yaml:"exclude_branches"`
		Protected       bool     `yaml:"protected"`
		Days            int      `yaml:"days"`
		BatchSize       int      `yaml:"batch_size"`
		WaitDays        int      `yaml:"wait_days"`
	} `yaml:"stale_branches_deletion"`

	PluginVars map[string]string `yaml:"plugin_vars"`
}

func New(providerName string) (*Request, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()

	if _, ok := providers[providerName]; !ok {
		return nil, &Error{text: "Provider can't be nil"}
	}

	constructor := providers[providerName]
	provider := constructor()
	if provider == nil {
		return nil, &Error{text: "Provider can't be nil"}
	}

	if err := contributors.Connect(); err != nil {
		return nil, err
	}

	return &Request{provider: provider}, nil
}
