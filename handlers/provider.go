package handlers

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"sync"

	"github.com/dustin/go-humanize/english"
)

const (
	configPath = ".mrbot.yaml"
)

var (
	providers   = map[string]func() RequestProvider{}
	providersMu sync.RWMutex

	StatusError            = &Error{"Is it opened?"}
	ValidError             = &Error{"Your request can't be merged, because either it has conflicts or state is not opened"}
	RepoSizeError          = &Error{"Repository size is greater than allowed size"}
	NotFoundError          = &Error{"Resource is not found"}
	DiscussionError        = &Error{"Could not find resolvable discussion for merge request"}
	CommitNotFoundError    = &Error{"Commit was not found"}
	ReviewersAssignedError = &Error{"MR has reviewers"}
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
	ProjectID       int64
	ID              int64
	Labels          []string
	TargetBranch    string
	SourceBranch    string
	Approvals       map[string]struct{}
	Reviewers       []string
	Author          string
	FailedPipelines int64
	FailedTests     int64
	Title           string
	Description     string
	ConfigContent   string
	IsValid         bool
}

type Candidate struct {
	Username    string
	Count       int
	StatusEmoji string
	Status      string
	Timezone    string
	IsCodeOwner bool
}

func (c Candidate) IsAvailable() bool {
	status := strings.ToLower(c.Status)

	for _, s := range vacationStatuses {
		if strings.Contains(status, s) {
			return false
		}
	}

	return !slices.Contains(emojiStatuses, c.StatusEmoji)
}

func (c Candidate) IsBot() bool {
	for _, s := range botNicks {
		if strings.Contains(strings.ToLower(c.Username), s) {
			return true
		}
	}
	return false
}

type RouletteResult struct {
	TotalPlayers       int
	UnavailablePlayers int
	Winners            []string
}

func (r RouletteResult) String() string {
	const rules string = `
<details>
<summary>
Roulette rules:
</summary>
<pre>
- Fetched all MR authors for last 3 months
- Filtered only users with contributors permissions
- Excluded:
  - usernames from .mrbot.yaml config
  - inactive users and bots
  - users with emoji status: 🏖️, 🔴, ⛔, 🌴
  - users with status: ooo, vacation, travel and parental leave
- CODEOWNERS have higher priority
</pre>
</details>
`

	formatUsernames := make([]string, 0, len(r.Winners))
	for _, u := range r.Winners {
		formatUsernames = append(formatUsernames, "@"+u)
	}

	unavailableMessage := ""
	if r.UnavailablePlayers > 0 {
		players := english.Plural(r.UnavailablePlayers, "player", "")
		unavailableMessage = fmt.Sprintf(", %s - unavailable", players)
	}

	return fmt.Sprintf(
		"🎲 **Review Roulette** — %d contributors in the pool%s\n\n 🧠 Reviewers selected: %s\n\n %s",
		r.TotalPlayers,
		unavailableMessage,
		strings.Join(formatUsernames, ", "),
		rules,
	)
}

type Branches interface {
	ListBranches(projectID, size int64, protected bool) iter.Seq[StaleBranch]
	DeleteBranch(projectID int64, name string) error
}

type Comments interface {
	LeaveComment(projectID, mergeID int64, message string) error
	AwardEmoji(projectID, mergeID, noteID int64, emoji string) error
}

type Discussions interface {
	CreateDiscussion(projectID, mergeID int64, message string) error
	UnresolveDiscussion(projectID, mergeID int64) error
	CreateThreadInLine(projectID, mergeID int64, thread Thread) error
}

type MergeRequest interface {
	Merge(projectID, mergeID int64, message string) error
	GetMRInfo(projectID, mergeID int64, path string) (*MrInfo, error)
	ListMergeRequests(projectID, size int64, protected bool) iter.Seq[MR]
	FindMergeRequests(projectID int64, targetBranch, label string) ([]MR, error)
	UpdateFromMaster(projectID, mergeID int64) error
	AssignLabel(projectID, mergeID int64, name, color string) error
	GetRawDiffs(projectID, mergeID int64) ([]byte, error)
	AssignReviewers(projectID, mergeID int64, users []string) error
}

type Project interface {
	CreateLabel(projectID int64, name, color string) error
	GetVar(projectID int64, varName string) (string, error)
	RerunPipeline(projectID, pipelineID int64, ref string) (string, error)
	GetFile(projectID int64, path string) ([]byte, error)
	IsHealthy() bool
	GetContributors(projectID, mergeID int64) ([]Candidate, error)
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
	Enabled          bool     `yaml:"enabled"`
	UseCodeowners    bool     `yaml:"use_codeowners"`
	ReviewerNumber   int      `yaml:"reviewer_number"`
	ExcludeUsernames []string `yaml:"exclude_usernames"`
}

type Config struct {
	Rules Rules `yaml:"rules"`

	Greetings struct {
		Enabled    bool   `yaml:"enabled"`
		Resolvable bool   `yaml:"resolvable"`
		Template   string `yaml:"template"`
	} `yaml:"greetings"`

	AutoMasterMerge bool            `yaml:"auto_master_merge"`
	AssignReviewers AssignReviewers `yaml:"review_roulette"`

	StaleBranchesDeletion struct {
		Enabled         bool     `yaml:"enabled"`
		ExcludeBranches []string `yaml:"exclude_branches"`
		Protected       bool     `yaml:"protected"`
		Days            int      `yaml:"days"`
		BatchSize       int64    `yaml:"batch_size"`
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

	return &Request{provider: provider}, nil
}
