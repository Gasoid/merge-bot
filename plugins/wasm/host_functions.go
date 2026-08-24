package wasm

import (
	"context"
	"encoding/json"
	"net/http"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/v3/handlers"
	"github.com/gasoid/merge-bot/v3/logger"

	html2md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type baseParams struct {
	Provider  string `json:"provider"`
	ProjectID int64  `json:"project_id"`
	ID        int64  `json:"id"`
	Branch    string `json:"branch"`
}

func (b baseParams) isValid() bool {
	if b.Provider != "" && b.ProjectID <= 0 && b.ID <= 0 && b.Branch != "" {
		return true
	}

	return false
}

type baseResult struct {
	Error string `json:"error"`
}

type getGitFileParams struct {
	baseParams
	FilePath string `json:"file_path"`
}

func (g getGitFileParams) isValid() bool {
	if g.FilePath == "" {
		return false
	}

	return g.baseParams.isValid()
}

type getGitFileResult struct {
	baseResult
	Data []byte `json:"data"`
}

type searchCodeParams struct {
	baseParams
	Query string `json:"query"`
}

func (g searchCodeParams) isValid() bool {
	if g.Query == "" {
		return false
	}

	return g.baseParams.isValid()
}

type searchCodeResult struct {
	baseResult
	Results []handlers.Search `json:"results"`
}

type fetchWebContentParams struct {
	baseParams
	Url string `json:"url"`
}

func (g fetchWebContentParams) isValid() bool {
	if g.Url == "" {
		return false
	}

	return g.baseParams.isValid()
}

type fetchWebContentResult struct {
	baseResult
	Content []byte `json:"content"`
}

func exitWithError(p *extism.CurrentPlugin, stack []uint64, msg string, args ...any) {
	logger.Info(msg, args...)
	data, err := json.Marshal(baseResult{Error: msg})
	if err != nil {
		logger.Info("exitWithError can't marshal data", "error", err)
		return
	}

	stack[0], err = p.WriteBytes(data)
	if err != nil {
		logger.Info("exitWithError can't write data", "error", err)
		return
	}
}

func returnData(p *extism.CurrentPlugin, stack []uint64, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		logger.Info("returnData can't marshal data", "error", err)
		return
	}

	stack[0], err = p.WriteBytes(data)
	if err != nil {
		logger.Info("returnData can't write data", "error", err)
		return
	}
}

var (
	getGitFile = extism.NewHostFunctionWithStack(
		"get_git_file",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				exitWithError(p, stack, "getGitFile can't read paramsBytes", "error", err)
				return
			}

			params := getGitFileParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				exitWithError(p, stack, "getGitFile can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				exitWithError(p, stack, "params of getGitFile are invalid")
				return
			}

			command, err := handlers.New(params.Provider)
			if err != nil {
				exitWithError(p, stack, "getGitFile can't create Request instance", "error", err)
				return
			}

			if err := command.LoadInfoAndConfig(params.ProjectID, params.ID); err != nil {
				exitWithError(p, stack, "can't load repo config", "provider", params.Provider, "err", err)
				return
			}

			data, err := command.GetFile(params.Branch, params.FilePath)
			if err != nil {
				exitWithError(p, stack, "getGitFile can't receive file", "error", err, "filePath", params.FilePath)
				return
			}

			returnData(p, stack, &getGitFileResult{Data: data})
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)

	searchCode = extism.NewHostFunctionWithStack(
		"search_code",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				exitWithError(p, stack, "searchCode can't read paramsBytes", "error", err)
				return
			}

			params := searchCodeParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				exitWithError(p, stack, "searchCode can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				exitWithError(p, stack, "params of searchCode are invalid")
				return
			}

			command, err := handlers.New(params.Provider)
			if err != nil {
				logger.Info("searchCode can't create Request instance", "error", err)
				return
			}

			if err := command.LoadInfoAndConfig(params.ProjectID, params.ID); err != nil {
				exitWithError(p, stack, "searchCode can't load repo config", "provider", params.Provider, "err", err)
				return
			}

			results := command.SearchCode(params.Branch, params.Query)
			returnData(p, stack, &searchCodeResult{Results: results})
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)

	//nolint:errcheck
	fetchWebContent = extism.NewHostFunctionWithStack(
		"fetch_web_content",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				exitWithError(p, stack, "fetchWebContent can't read paramsBytes", "error", err)
				return
			}

			params := fetchWebContentParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				exitWithError(p, stack, "fetchWebContent can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				exitWithError(p, stack, "params of fetchWebContent are invalid")
				return
			}

			command, err := handlers.New(params.Provider)
			if err != nil {
				logger.Info("fetchWebContent can't create Request instance", "error", err)
				return
			}

			if err := command.LoadInfoAndConfig(params.ProjectID, params.ID); err != nil {
				exitWithError(p, stack, "fetchWebContent can't load repo config", "provider", params.Provider, "err", err)
				return
			}

			res, err := http.Get(params.Url)
			if err != nil {
				exitWithError(p, stack, "fetchWebContent can't get url", "err", err, "url", params.Url)
				return
			}

			defer res.Body.Close()

			content, err := html2md.ConvertReader(res.Body)
			if err != nil {
				exitWithError(p, stack, "fetchWebContent can't convert to md", "err", err, "url", params.Url)
				return
			}

			returnData(p, stack, &fetchWebContentResult{Content: content})
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
)
