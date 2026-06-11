package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"math/rand"
	"slices"
	"sort"
	"strings"

	"github.com/gasoid/merge-bot/v3/cache"
	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/gasoid/merge-bot/v3/metrics"

	"gopkg.in/yaml.v3"
)

const (
	autoUpdateLabel      = "merge-bot:auto-update"
	autoUpdateLabelColor = "#6699cc"
	staleLabel           = "merge-bot:stale"
	staleLabelColor      = "#cccccc"
	DecrCount            = "merge"
	IncrCount            = "update"
)

var (
	vacationStatuses = []string{"ooo", "vacation", "travel", "parental leave"}
	emojiStatuses    = []string{"beach", "beach_umbrella", "palm_tree", "red_circle", "no_entry"}
	botNicks         = []string{"bot", "jira", "gitlab", "github"}
)

type Request struct {
	provider RequestProvider
	info     *MrInfo
	config   *Config
}

func (r *Request) LoadInfoAndConfig(projectId, id int64) error {
	var err error
	r.info, err = r.provider.GetMRInfo(projectId, id, configPath)
	if err != nil {
		return err
	}

	r.config, err = r.ParseConfig(r.info.ConfigContent)
	if err != nil {
		return err
	}

	return nil
}

func (r *Request) IsValid() (bool, string, error) {
	if !r.info.IsValid {
		return false, ValidError.Error(), nil
	}

	result := make([]string, len(checkers)+1)
	resultOk := true
	for i, check := range checkers {
		r := check(r.config, r.info)
		if !r.Required {
			continue
		}
		if r.Passed {
			result[i] = r.Message + " ✅"
		} else {
			result[i] = r.Message + " ❌"
			resultOk = false
		}
	}

	if r.config.Rules.Approvers == nil {
		result[len(checkers)] = "> [!important]\n> **Approvers configuration missing**\n> \n> Please configure `rules.approvers` in your merge bot config:\n> - For specific approvers: `rules.approvers: [\"user1\", \"user2\"]`\n> - For no specific approvers: `rules.approvers: []`"
		resultOk = false
	}

	return resultOk, strings.Join(result, "\n\n"), nil
}

func (r *Request) ParseConfig(content string) (*Config, error) {
	mrConfig := &Config{
		Rules: Rules{
			MinApprovals:          1,
			AllowFailingPipelines: true,
			AllowFailingTests:     true,
			TitleRegex:            ".*",
			AllowEmptyDescription: true,
		},
		Greetings: struct {
			Enabled    bool   `yaml:"enabled"`
			Resolvable bool   `yaml:"resolvable"`
			Template   string `yaml:"template"`
		}{
			Enabled:    false,
			Resolvable: false,
			Template:   "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nOnce you're done, send **!merge** command and I will merge it!",
		},
		AutoMasterMerge: false,
		AssignReviewers: AssignReviewers{
			UseCodeowners:    true,
			ReviewerNumber:   2,
			ExcludeUsernames: []string{},
		},
		StaleBranchesDeletion: struct {
			Enabled         bool     `yaml:"enabled"`
			ExcludeBranches []string `yaml:"exclude_branches"`
			Protected       bool     `yaml:"protected"`
			Days            int      `yaml:"days"`
			BatchSize       int64    `yaml:"batch_size"`
			WaitDays        int      `yaml:"wait_days"`
		}{
			Enabled:         false,
			ExcludeBranches: []string{},
			Protected:       false,
			Days:            90,
			BatchSize:       5,
			WaitDays:        1,
		},
	}

	if err := yaml.Unmarshal([]byte(content), mrConfig); err != nil {
		return nil, err
	}
	return mrConfig, nil
}

func (r *Request) LeaveComment(message string) error {
	return r.provider.LeaveComment(r.info.ProjectID, r.info.ID, message)
}

func (r Request) CreateDiscussion(message string) error {
	return r.provider.CreateDiscussion(r.info.ProjectID, r.info.ID, message)
}

func (r Request) UnresolveDiscussion() error {
	if !r.config.Greetings.Resolvable || !r.config.Greetings.Enabled {
		return nil
	}

	return r.provider.UnresolveDiscussion(r.info.ProjectID, r.info.ID)
}

func (r Request) getGreetingsText() (string, error) {
	tmpl, err := template.New("greetings").Parse(r.config.Greetings.Template)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, r.config.Rules); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r *Request) Greetings() error {
	if !r.config.Greetings.Enabled {
		return nil
	}

	renderedMessage, err := r.getGreetingsText()
	if err != nil {
		return err
	}

	if r.config.Greetings.Resolvable {
		return r.CreateDiscussion(renderedMessage)
	}

	return r.LeaveComment(renderedMessage)
}

func (r *Request) DeleteStaleBranches() error {

	if !r.config.StaleBranchesDeletion.Enabled {
		return nil
	}

	if !cache.TryAcquireBranchDeletionLock(r.info.ProjectID) {
		return nil
	}

	defer cache.BranchDeletionUnlock(r.info.ProjectID)

	metrics.BackgroundRunInc("clean_stale_merge_requests")

	if err := r.cleanStaleMergeRequests(); err != nil {
		logger.Info("cleanStaleMergeRequests", "err", err)
	}

	metrics.BackgroundRunInc("clean_stale_branches")

	if err := r.cleanStaleBranches(); err != nil {
		logger.Info("cleanStaleBranches", "err", err)
	}

	return nil
}

func (r *Request) Merge() (bool, string, error) {
	if r.config.AutoMasterMerge {
		err := r.provider.UpdateFromMaster(r.info.ProjectID, r.info.ID)
		if err != nil {
			return false, "", err
		}
	}

	if ok, text, err := r.IsValid(); ok {
		if err := r.provider.Merge(r.info.ProjectID, r.info.ID, fmt.Sprintf("%s\nMerged by MergeApproveBot", r.info.Title)); err != nil {
			return false, "", err
		}
		return true, "", nil
	} else {
		return false, text, err
	}
}

func (r Request) UpdateFromMaster() error {
	if err := r.provider.UpdateFromMaster(r.info.ProjectID, r.info.ID); err != nil {
		return err
	}
	return nil
}

func (r Request) UpdateBranches() error {
	listMr, err := r.provider.FindMergeRequests(r.info.ProjectID, r.info.TargetBranch, autoUpdateLabel)
	if err != nil {
		return err
	}

	if !cache.TryAcquireUpdateLock(r.info.ProjectID) {
		return nil
	}

	defer cache.UpdateUnlock(r.info.ProjectID)

	for _, mr := range listMr {
		metrics.BackgroundRunInc("update_branch")

		if err := r.provider.UpdateFromMaster(r.info.ProjectID, mr.ID); err != nil {
			logger.Info("UpdateFromDestination", "err", err)
		}
	}

	return nil
}

func (r Request) CreateLabels() error {
	if err := r.provider.CreateLabel(r.info.ProjectID, staleLabel, staleLabelColor); err != nil {
		return err
	}

	if err := r.provider.CreateLabel(r.info.ProjectID, autoUpdateLabel, autoUpdateLabelColor); err != nil {
		return err
	}
	return nil
}

func (r Request) RerunPipeline(pipelineID int64) (string, error) {
	logger.Debug("rerun", "pipelineId", pipelineID)
	return r.provider.RerunPipeline(r.info.ProjectID, pipelineID, r.info.SourceBranch)
}

func (r Request) ValidateSecret(secret string) bool {
	const mergeBotSecret = "MERGE_BOT_SECRET"

	secretVar, err := r.provider.GetVar(r.info.ProjectID, mergeBotSecret)
	if err != nil {
		logger.Info("cound't validate secret", "err", err)
		return false
	}

	return secretVar == secret
}

func (r Request) AwardEmoji(noteID int64, emoji string) error {
	return r.provider.AwardEmoji(r.info.ProjectID, r.info.ID, noteID, emoji)
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

func (r Request) spinRoulette(num int) ([]string, error) {
	gamblers, err := r.provider.GetContributors(r.info.ProjectID, r.info.ID)
	if err != nil {
		return nil, err
	}

	counts, err := cache.GetCounts(r.info.ProjectID)
	if err != nil {
		return nil, err
	}

	if len(counts) > 0 {
		for i := range gamblers {
			if v, ok := counts[gamblers[i].Username]; ok {
				gamblers[i].Count = v
			}
		}
	}

	rand.Shuffle(len(gamblers)/2, func(i, j int) {
		gamblers[i], gamblers[j] = gamblers[j], gamblers[i]
	})

	sort.Slice(gamblers, func(i, j int) bool {
		if r.config.AssignReviewers.UseCodeowners {
			if gamblers[i].IsCodeOwner && !gamblers[j].IsCodeOwner {
				return true
			}

			if gamblers[j].IsCodeOwner && !gamblers[i].IsCodeOwner {
				return false
			}
		}

		return gamblers[i].Count < gamblers[j].Count
	})

	usernames := make([]string, 0, r.config.AssignReviewers.ReviewerNumber)

	for _, g := range gamblers {
		if !g.IsAvailable() {
			continue
		}

		if g.IsBot() {
			continue
		}

		if g.Username == r.info.Author {
			continue
		}

		if slices.Contains(r.config.AssignReviewers.ExcludeUsernames, g.Username) {
			continue
		}

		usernames = append(usernames, g.Username)
		if len(usernames) == num {
			break
		}
	}

	return usernames, nil
}

func (r Request) reviewRoulette(num int) error {
	if len(r.info.Reviewers) != 0 {
		if err := r.provider.LeaveComment(r.info.ProjectID, r.info.ID, "🎲 Merge Request has assigned reviewers already"); err != nil {
			return err
		}
		return nil
	}

	usernames, err := r.spinRoulette(max(num, 1))
	if err != nil {
		return err
	}

	logger.Debug("usernames for review", "usernames", usernames)
	for _, u := range usernames {
		if _, err := cache.IncrCount(r.info.ProjectID, u); err != nil {
			logger.Error("can't increment count", "err", err)
		}
	}

	if len(usernames) == 0 {
		return nil
	}

	formatUsernames := make([]string, 0, len(usernames))
	for _, u := range usernames {
		formatUsernames = append(formatUsernames, "@"+u)
	}

	rouletteMessage := fmt.Sprintf("🎲 **Review Roulette** — %d contributors in the pool\nReviewers selected: %s", len(formatUsernames), strings.Join(formatUsernames, ","))
	if err := r.provider.LeaveComment(r.info.ProjectID, r.info.ID, rouletteMessage); err != nil {
		return err
	}

	return r.provider.AssignReviewers(r.info.ProjectID, r.info.ID, usernames)
}

func (r Request) AssignReviewers() error {
	if !r.config.AssignReviewers.Enabled {
		return nil
	}

	ok, _, err := r.IsValid()
	if err != nil {
		return err
	}

	if ok {
		return r.reviewRoulette(r.config.AssignReviewers.ReviewerNumber)
	} else {
		return nil
	}
}

func (r Request) ReviewRoulette(num int) error {
	if num == 0 {
		num = r.config.AssignReviewers.ReviewerNumber
	}

	return r.reviewRoulette(num)
}

func (r Request) UpdateReviewRouletteCounts() error {
	metrics.BackgroundRunInc("update_review_roulette_counts")

	gamblers, err := r.provider.GetContributors(r.info.ProjectID, r.info.ID)
	if err != nil {
		return err
	}

	counts, err := cache.GetCounts(r.info.ProjectID)
	if err != nil {
		return err
	}

	if len(counts) > 0 {
		return nil
	}

	counts = make(map[string]int, len(gamblers))

	for _, c := range gamblers {
		counts[c.Username] = c.Count
	}

	if err := cache.SetCounts(r.info.ProjectID, counts); err != nil {
		return err
	}

	return nil
}
