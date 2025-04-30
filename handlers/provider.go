package handlers

import (
	"sync"
)

const (
	configPath = ".mrbot.yaml"
)

var (
	providers   = map[string]func() RequestProvider{}
	providersMu sync.RWMutex

	StatusError   = &Error{"Is it opened?"}
	ValidError    = &Error{"Your request can't be merged, because either it has conflicts or state is not opened"}
	RepoSizeError = &Error{"Repository size is greater than allowed size"}
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
	Approvals       map[string]struct{}
	FailedPipelines int
	FailedTests     int
	Title           string
	Description     string
	ConfigContent   string
	IsValid         bool
}

type RequestProvider interface {
	Merge(projectId, mergeId int, message string) error
	LeaveComment(projectId, mergeId int, message string) error
	GetMRInfo(projectId, mergeId int, path string) (*MrInfo, error)
	UpdateFromMaster(projectId, mergeId int) error
}

type Config struct {
	MinApprovals          int      `yaml:"min_approvals"`
	Approvers             []string `yaml:"approvers"`
	AllowFailingPipelines bool     `yaml:"allow_failing_pipelines"`
	AllowFailingTests     bool     `yaml:"allow_failing_tests"`
	TitleRegex            string   `yaml:"title_regex"`
	AllowEmptyDescription bool     `yaml:"allow_empty_description"`
	EnableGreetings       bool     `yaml:"greetings_enabled"`
	GreetingsTemplate     string   `yaml:"greetings_template"`
	AutoMasterMerge       bool     `yaml:"auto_master_merge"`
}

func New(providerName string) (*Request, error) {
	providersMu.Lock()
	defer providersMu.Unlock()

	if _, ok := providers[providerName]; !ok {
		return nil, &Error{text: "Provider can't be nil"}
	}

	constructor := providers[providerName]
	provider := constructor()
	if provider == nil {
		return nil, &Error{text: "Provider can't be nil"}
	}

	return &Request{provider: provider}, nil
}
