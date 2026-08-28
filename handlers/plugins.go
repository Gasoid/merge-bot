package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gasoid/merge-bot/v3/logger"
)

const (
	envType    = "env"
	configType = "config"
	secretType = "secret"
)

type PluginCall func([]byte) ([]byte, error)

type JobRef struct {
	Name         string `json:"name"`
	Stage        string `json:"stage"`
	ID           int64  `json:"id"`
	AllowFailure bool   `json:"allow_failure"`
}

type CIInfo struct {
	PipelineStatus string    `json:"pipeline_status"`
	FailedJobs     []JobRef  `json:"failed_jobs,omitempty"`
	FailedTests    []TestRef `json:"failed_tests,omitempty"`
}

type TestRef struct {
	Name      string `json:"name"`
	Suite     string `json:"test_suite"`
	Output    string `json:"output,omitempty"`
	File      string `json:"file,omitempty"`
	ClassName string `json:"classname,omitempty"`
}

type PluginInput struct {
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Branch       string            `json:"branch"`
	TargetBranch string            `json:"target_branch"`
	Diffs        []byte            `json:"diffs"`
	Vars         map[string]string `json:"vars"`
	CIInfo       *CIInfo           `json:"ci_info,omitempty"`
}

type Thread struct {
	NewLine int64  `json:"new_line"`
	OldLine int64  `json:"old_line"`
	Body    string `json:"body"`
	NewPath string `json:"new_path"`
	OldPath string `json:"old_path"`
}

type PluginOutput struct {
	Comment string   `json:"comment"`
	Threads []Thread `json:"threads"`
}

func (r Request) RunWithContext(call PluginCall, vars map[string][]string) error {
	if r.info == nil {
		return errors.New("no MR info")
	}

	rawDiffs, err := r.provider.GetRawDiffs(r.info.ProjectID, r.info.ID)
	if err != nil {
		return err
	}

	pluginVars := map[string]string{}

	for k, v := range vars {
		for _, t := range v {
			switch t {
			case envType:
				val := os.Getenv(strings.ToUpper(k))
				if val == "" {
					continue
				}

				pluginVars[k] = val

			case configType:
				if _, ok := r.config.PluginVars[k]; ok && r.config.PluginVars[k] != "" {
					pluginVars[k] = r.config.PluginVars[k]
				}

			case secretType:
				val, err := r.provider.GetVar(r.info.ProjectID, strings.ToUpper(k))
				if err != nil {
					return err
				}

				if val == "" {
					continue
				}

				pluginVars[k] = val
			}
		}
	}

	ciInfo, err := r.GetCIInfo()
	if err != nil {
		logger.Info("GetCIInfo returned error", "error", err)
	}

	input := PluginInput{
		Title:        r.info.Title,
		Description:  r.info.Description,
		Author:       r.info.Author,
		Diffs:        rawDiffs,
		Vars:         pluginVars,
		Branch:       r.info.SourceBranch,
		TargetBranch: r.info.TargetBranch,
		CIInfo:       ciInfo,
	}

	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	out, err := call(data)
	if err != nil {
		return err
	}

	output := PluginOutput{}

	if err := json.Unmarshal(out, &output); err != nil {
		return err
	}

	if output.Comment != "" {
		if err := r.LeaveComment(output.Comment); err != nil {
			return err
		}
	}

	for _, t := range output.Threads {
		if err := r.provider.CreateThreadInLine(
			r.info.ProjectID,
			r.info.ID,
			t); err != nil {
			logger.Info("CreateThreadInLine returns error", "err", err, "thread", t)

			if err := r.LeaveComment(threadFallback(t)); err != nil {
				return err
			}
		}
	}

	return nil
}

func threadFallback(t Thread) string {
	path, line := t.NewPath, t.NewLine
	if line == 0 && t.OldLine != 0 {
		path, line = t.OldPath, t.OldLine
	}
	if path == "" {
		path = t.OldPath
	}

	location := path
	if line != 0 {
		location = fmt.Sprintf("%s:%d", path, line)
	}

	return fmt.Sprintf("> [!note]\n> **%s**\n\n%s", location, t.Body)
}
