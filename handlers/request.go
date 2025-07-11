package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/Gasoid/mergebot/logger"

	"gopkg.in/yaml.v3"
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

	result := make([]string, len(checkers))
	resultOk := true
	for i, c := range checkers {
		ok, enabled := c.checkFunc(r.config, r.info)
		if !enabled {
			continue
		}
		if ok {
			result[i] = c.text + " ✅"
		} else {
			result[i] = c.text + " ❌"
			resultOk = false
		}
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
	if r.config.StaleBranchesDeletion.Enabled {
		if err := r.cleanStaleMergeRequests(); err != nil {
			return err
		}

		if err := r.cleanStaleBranches(); err != nil {
			return err
		}
	}

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

func (r *Request) UpdateFromMaster() error {
	if err := r.provider.UpdateFromMaster(r.info.ProjectId, r.info.Id); err != nil {
		return err
	}
	return nil
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
