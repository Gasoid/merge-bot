package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/gasoid/merge-bot/logger"
	"github.com/gasoid/merge-bot/metrics"
	"github.com/gasoid/merge-bot/semaphore"

	"gopkg.in/yaml.v3"
)

const (
	autoUpdateLabel      = "merge-bot:auto-update"
	autoUpdateLabelColor = "#6699cc"
	staleLabel           = "merge-bot:stale"
	staleLabelColor      = "#cccccc"
)

var (
	deleteStaleBranches = semaphore.NewKeyedSemaphore(1)
	updateBranch        = semaphore.NewKeyedSemaphore(2)
)

type Request struct {
	provider RequestProvider
	info     *MrInfo
	config   *Config
}

func (r *Request) LoadInfoAndConfig(projectId, id int) error {
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
			ResetApprovalsOnPush: ResetApprovalsOnPush{
				Enabled:        false,
				IssueToken:     true,
				ProjectVarName: "MERGE_BOT_RESET_APPROVAL_TOKEN",
			},
		},
		Greetings: struct {
			Enabled  bool   `yaml:"enabled"`
			Template string `yaml:"template"`
		}{
			Enabled:  false,
			Template: "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nOnce you've done, send **!merge** command and i will merge it!",
		},
		AutoMasterMerge: false,
		StaleBranchesDeletion: struct {
			Enabled   bool `yaml:"enabled"`
			Days      int  `yaml:"days"`
			BatchSize int  `yaml:"batch_size"`
			WaitDays  int  `yaml:"wait_days"`
		}{
			Enabled:   false,
			Days:      90,
			BatchSize: 5,
			WaitDays:  1,
		},
	}

	if err := yaml.Unmarshal([]byte(content), mrConfig); err != nil {
		return nil, err
	}
	return mrConfig, nil
}

func (r *Request) LeaveComment(message string) error {
	return r.provider.LeaveComment(r.info.ProjectId, r.info.Id, message)
}

func (r *Request) Greetings() error {
	if !r.config.Greetings.Enabled {
		return nil
	}

	tmpl, err := template.New("greetings").Parse(r.config.Greetings.Template)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, r.config.Rules); err != nil {
		return err
	}

	return r.LeaveComment(buf.String())
}

func (r *Request) DeleteStaleBranches() error {

	if !r.config.StaleBranchesDeletion.Enabled {
		return nil
	}

	deleteStaleBranches.Add(fmt.Sprintf("clean_stale_merge_requests_%d", r.info.ProjectId),
		metrics.BackgroundRun("clean_stale_merge_requests", func() {
			if err := r.cleanStaleMergeRequests(); err != nil {
				logger.Info("cleanStaleMergeRequests", "err", err)
			}
		}))

	deleteStaleBranches.Add(fmt.Sprintf("clean_stale_branches_%d", r.info.ProjectId),
		metrics.BackgroundRun("clean_stale_branches", func() {
			if err := r.cleanStaleBranches(); err != nil {
				logger.Info("cleanStaleBranches", "err", err)
			}
		}))

	return nil
}

func (r *Request) Merge() (bool, string, error) {
	if r.config.AutoMasterMerge {
		err := r.provider.UpdateFromMaster(r.info.ProjectId, r.info.Id)
		if err != nil {
			return false, "", err
		}
	}

	if ok, text, err := r.IsValid(); ok {
		if err := r.provider.Merge(r.info.ProjectId, r.info.Id, fmt.Sprintf("%s\nMerged by MergeApproveBot", r.info.Title)); err != nil {
			return false, "", err
		}
		return true, "", nil
	} else {
		return false, text, err
	}
}

func (r Request) UpdateFromMaster() error {
	if err := r.provider.UpdateFromMaster(r.info.ProjectId, r.info.Id); err != nil {
		return err
	}
	return nil
}

func (r Request) UpdateBranches() error {
	listMr, err := r.provider.FindMergeRequests(r.info.ProjectId, r.info.TargetBranch, autoUpdateLabel)
	if err != nil {
		return err
	}

	for _, mr := range listMr {
		updateBranch.Add(fmt.Sprintf("update_branch_%d_%d", r.info.ProjectId, mr.Id),
			metrics.BackgroundRun("update_branch", func() {
				if err := r.provider.UpdateFromMaster(r.info.ProjectId, mr.Id); err != nil {
					logger.Info("UpdateFromDestination", "err", err)
				}
			}))
	}

	return nil
}

func (r Request) CreateLabels() error {
	if err := r.provider.CreateLabel(r.info.ProjectId, staleLabel, staleLabelColor); err != nil {
		return err
	}

	if err := r.provider.CreateLabel(r.info.ProjectId, autoUpdateLabel, autoUpdateLabelColor); err != nil {
		return err
	}
	return nil
}

func (r Request) RerunPipeline(pipelineId int) (string, error) {
	logger.Debug("rerun", "pipelineId", pipelineId)
	return r.provider.RerunPipeline(r.info.ProjectId, pipelineId, r.info.SourceBranch)
}

func (r Request) ResetApprovals(updatedAt time.Time) error {
	if !r.config.Rules.ResetApprovalsOnPush.Enabled {
		return nil
	}
	return r.provider.ResetApprovals(r.info.ProjectId, r.info.Id, updatedAt, r.config.Rules.ResetApprovalsOnPush)
}

func (r Request) ValidateSecret(secret string) bool {
	const mergeBotSecret = "MERGE_BOT_SECRET"

	secretVar, err := r.provider.GetVar(r.info.ProjectId, mergeBotSecret)
	if err != nil {
		logger.Info("cound't validate secret", "err", err)
		return false
	}

	return secretVar == secret
}
