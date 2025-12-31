package handlers

import (
	"encoding/json"
)

type PluginCall func([]byte) ([]byte, error)

type PluginInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Diffs       []byte `json:"diffs"`
}

type PluginOutput struct {
	Comment string `json:"comment"`
}

func (r Request) RunWithContext(call PluginCall) error {
	rawDiffs, err := r.provider.GetRawDiffs(r.info.ProjectId, r.info.Id)
	if err != nil {
		return err
	}

	input := PluginInput{
		Title:       r.info.Title,
		Description: r.info.Description,
		Author:      r.info.Author,
		Diffs:       rawDiffs,
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

	return nil
}
