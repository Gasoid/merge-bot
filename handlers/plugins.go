package handlers

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/gasoid/merge-bot/logger"
)

const (
	envType    = "env"
	configType = "config"
	secretType = "secret"
)

type PluginCall func([]byte) ([]byte, error)

type PluginInput struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Diffs       []byte            `json:"diffs"`
	Vars        map[string]string `json:"vars"`
}

type Thread struct {
	NewLine int    `json:"new_line"`
	OldLine int    `json:"old_line"`
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

	rawDiffs, err := r.provider.GetRawDiffs(r.info.ProjectId, r.info.Id)
	if err != nil {
		return err
	}

	pluginVars := map[string]string{}

	for k, v := range vars {
		for _, t := range v {
			if t == envType {
				val := os.Getenv(strings.ToUpper(k))
				if val == "" {
					continue
				}

				pluginVars[k] = val
			}

			if t == configType {
				if _, ok := r.config.PluginVars[k]; ok && r.config.PluginVars[k] != "" {
					pluginVars[k] = r.config.PluginVars[k]
				}
			}

			if t == secretType {
				val, err := r.provider.GetVar(r.info.ProjectId, strings.ToUpper(k))
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

	input := PluginInput{
		Title:       r.info.Title,
		Description: r.info.Description,
		Author:      r.info.Author,
		Diffs:       rawDiffs,
		Vars:        pluginVars,
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
			r.info.ProjectId,
			r.info.Id,
			t); err != nil {
			logger.Info("CreateThreadInLine returns error", "err", err, "thread", t)
		}
	}

	return nil
}
