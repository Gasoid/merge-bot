package wasm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/v3/handlers"
	"github.com/gasoid/merge-bot/v3/logger"

	html2md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type baseParams struct {
	Branch string `json:"branch"`
}

func (b baseParams) isValid() bool {
	return b.Branch != ""
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
	Url string `json:"url"`
}

func (g fetchWebContentParams) isValid() bool {
	if g.Url == "" {
		return false
	}

	parsedURL, err := url.Parse(g.Url)
	if err != nil {
		return false
	}

	for _, suffix := range allowedDomains {
		if parsedURL.Host == suffix || strings.HasSuffix(parsedURL.Host, "."+suffix) {
			return true
		}
	}

	return false
}

type fetchWebContentResult struct {
	baseResult
	Content []byte `json:"content"`
}

const (
	maxFetchContentSize int64 = 5 * 1024 * 1024 // 5MB
)

func returnError(p *extism.CurrentPlugin, stack []uint64, msg string, args ...any) {
	logger.Info(msg, args...)
	data, err := json.Marshal(baseResult{Error: msg})
	if err != nil {
		logger.Info("returnError can't marshal data", "error", err)
		return
	}

	stack[0], err = p.WriteBytes(data)
	if err != nil {
		logger.Info("returnError can't write data", "error", err)
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
			var (
				command *handlers.Request
				ok      bool
			)

			if v := ctx.Value(commandCtxKey); v == nil {
				returnError(p, stack, "getGitFile can't get command from context")
				return
			} else {
				if command, ok = v.(*handlers.Request); !ok {
					returnError(p, stack, "getGitFile can't get command from context")
					return
				}
			}

			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				returnError(p, stack, "getGitFile can't read paramsBytes", "error", err)
				return
			}

			params := getGitFileParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				returnError(p, stack, "getGitFile can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				returnError(p, stack, "params of getGitFile are invalid")
				return
			}

			logger.Debug("getGitFile called from plugin", "filePath", params.FilePath, "branch", params.Branch)

			data, err := command.GetFile(params.Branch, params.FilePath)
			if err != nil {
				returnError(p, stack, "getGitFile can't receive file", "error", err, "filePath", params.FilePath)
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
			var (
				command *handlers.Request
				ok      bool
			)

			if v := ctx.Value(commandCtxKey); v == nil {
				returnError(p, stack, "searchCode can't get command from context")
				return
			} else {
				if command, ok = v.(*handlers.Request); !ok {
					returnError(p, stack, "searchCode can't get command from context")
					return
				}
			}

			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				returnError(p, stack, "searchCode can't read paramsBytes", "error", err)
				return
			}

			params := searchCodeParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				returnError(p, stack, "searchCode can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				returnError(p, stack, "params of searchCode are invalid")
				return
			}

			logger.Debug("searchCode called from plugin", "query", params.Query, "branch", params.Branch)

			results := command.SearchCode(params.Branch, params.Query)
			returnData(p, stack, &searchCodeResult{Results: results})
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)

	allowedDomains = []string{
		"go.dev",
		"golang.org",
		"docs.python.org",
		"developer.mozilla.org",
		"github.com",
		"gitlab.com",
		"docs.rs",
		"crates.io",
		"doc.rust-lang.org",
		"npmjs.com",
		"nodejs.org",
		"pypi.org",
		"kubernetes.io",
		"helm.sh",
		"grpc.io",
		"protobuf.dev",
		"postgresql.org",
		"redis.io",
		"learn.microsoft.com",
		"aws.amazon.com",
		"spec.openapis.org",
		"graphql.org",
		"swagger.io",
	}

	//nolint:errcheck
	fetchWebContent = extism.NewHostFunctionWithStack(
		"fetch_web_content",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			paramsBytes, err := p.ReadBytes(stack[0])
			if err != nil {
				returnError(p, stack, "fetchWebContent can't read paramsBytes", "error", err)
				return
			}

			params := fetchWebContentParams{}

			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				returnError(p, stack, "fetchWebContent can't unmarshal params", "error", err)
				return
			}

			if !params.isValid() {
				returnError(p, stack, "params of fetchWebContent are invalid")
				return
			}

			logger.Debug("fetchWebContent called from plugin", "url", params.Url)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.Url, nil)
			if err != nil {
				returnError(p, stack, "fetchWebContent can't build request", "err", err, "url", params.Url)
				return
			}

			client := &http.Client{
				Timeout: 15 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			res, err := client.Do(req)
			if err != nil {
				returnError(p, stack, "fetchWebContent can't get url", "err", err, "url", params.Url)
				return
			}

			defer res.Body.Close()

			limitReader := http.MaxBytesReader(nil, res.Body, maxFetchContentSize)

			if res.StatusCode != http.StatusOK {
				returnError(p, stack, "fetchWebContent got bad status", "status", res.StatusCode, "url", params.Url)
				return
			}

			content, err := html2md.ConvertReader(limitReader)
			if err != nil {
				returnError(p, stack, "fetchWebContent can't convert to md", "err", err, "url", params.Url)
				return
			}

			returnData(p, stack, &fetchWebContentResult{Content: content})
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
)
