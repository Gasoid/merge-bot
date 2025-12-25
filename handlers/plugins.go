package handlers

import "encoding/json"

type PluginCall func([]byte) error

type PluginContext struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Diffs       []byte `json:"diffs"`
}

func (r *Request) RunWithContext(call PluginCall) error {
	rawDiffs, err := r.provider.GetRawDiffs(r.info.ProjectId, r.info.Id)
	if err != nil {
		return err
	}

	ctx := PluginContext{
		Title:       r.info.Title,
		Description: r.info.Description,
		Author:      r.info.Author,
		Diffs:       rawDiffs,
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		return err
	}

	return call(data)
}
