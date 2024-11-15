package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

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

func (r *Request) IsValid(projectId, id int) (bool, string, error) {
	err := r.LoadInfoAndConfig(projectId, id)
	if err != nil {
		return false, "", err
	}

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
		MinApprovals:          1,
		AllowFailingPipelines: true,
		AllowFailingTests:     true,
		TitleRegex:            ".*",
		AllowEmptyDescription: true,
		GreetingsTemplate:     "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nOnce you've done, send **!merge** command and i will merge it!",
		AutoMasterMerge:       false,
	}

	err := yaml.Unmarshal([]byte(content), mrConfig)
	if err != nil {
		return nil, err
	}
	return mrConfig, nil
}

func (r *Request) LeaveComment(projectId, id int, message string) error {
	return r.provider.LeaveComment(projectId, id, message)
}

func (r *Request) Greetings(projectId, id int) error {
	err := r.LoadInfoAndConfig(projectId, id)
	if err != nil {
		return err
	}

	tmpl, err := template.New("greetings").Parse(r.config.GreetingsTemplate)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, r.config)
	if err != nil {
		return err
	}

	return r.LeaveComment(projectId, id, buf.String())
}

func (r *Request) Merge(projectId, id int) (bool, string, error) {
	err := r.LoadInfoAndConfig(projectId, id)
	if err != nil {
		return false, "", err
	}

	if r.config.AutoMasterMerge {
		// ignore error
		_ = r.provider.UpdateFromMaster(projectId, id)
	}

	if ok, text, err := r.IsValid(projectId, id); ok {
		err := r.provider.Merge(projectId, id, fmt.Sprintf("%s\nMerged by MergeApproveBot", r.info.Title))
		if err != nil {
			return false, "", err
		}
		return true, "", nil
	} else {
		return false, text, err
	}
}
