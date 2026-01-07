package plugins

import (
	"io"
	"iter"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gasoid/merge-bot/config"
	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/logger"
	"gopkg.in/yaml.v3"
)

const (
	githubURL = "https://github.com/"
	gitlabURL = "https://gitlab.com/"
)

var (
	plugins   string
	engines   = map[string]func([]byte, map[string][]string) (HandlerFunc, error){}
	enginesMu sync.RWMutex
)

type HandlerFunc func(command *handlers.Request, args string) error

func init() {
	config.StringVar(&plugins, "plugins", "", "comma list of plugin urls (also via PLUGINS)")
}

type Var struct {
	Name string   `yaml:"name"`
	Type []string `yaml:"type"`
}

type PluginManifest struct {
	Name    string      `yaml:"name"`
	Command string      `yaml:"command"`
	Runtime string      `yaml:"runtime"`
	Handler HandlerFunc `yaml:"-"`
	Vars    []Var       `yaml:"vars"`
}

func Register(name string, constructor func([]byte, map[string][]string) (HandlerFunc, error)) {
	enginesMu.Lock()
	defer enginesMu.Unlock()
	engines[name] = constructor
}

//nolint:errcheck
func downloadFile(fileUrl string) ([]byte, error) {
	resp, err := http.Get(fileUrl)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func getFile(filePath string) ([]byte, error) {
	if strings.HasPrefix(filePath, "https://") {
		return downloadFile(filePath)
	}

	return os.ReadFile(filePath)
}

func GetRawLink(link string) string {
	switch {
	case strings.HasPrefix(link, githubURL):
		if strings.Contains(link, "/releases/download/") {
			return link
		}

		link = strings.Replace(link, githubURL, "https://raw.githubusercontent.com/", 1)
		link = strings.Replace(link, "blob/", "", 1)

	case strings.HasPrefix(link, gitlabURL):
		link = strings.Replace(link, "-/blob/", "-/raw/", 1)

	}
	return link
}

func getManifest(filePath string) ([]byte, error) {
	filePath = GetRawLink(filePath)

	body, err := getFile(filePath)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func Load() iter.Seq[PluginManifest] {

	return func(yield func(PluginManifest) bool) {
		if plugins == "" {
			return
		}

		for pluginUrl := range strings.SplitSeq(plugins, ",") {
			pluginUrl = strings.TrimSpace(pluginUrl)
			if pluginUrl == "" {
				logger.Info("GetHandlers", "err", "url is empty")
				continue
			}

			manifest := PluginManifest{}

			manifestFile, err := getManifest(pluginUrl)
			if err != nil {
				logger.Error("getManifest", "plugin_url", pluginUrl, "err", err)
				continue
			}

			if err := yaml.Unmarshal(manifestFile, &manifest); err != nil {
				logger.Error("plugins.Load yaml Unmarshal failed", "err", err)
				continue
			}

			enginesMu.RLock()
			defer enginesMu.RUnlock()

			vars := map[string][]string{}
			for _, v := range manifest.Vars {
				vars[v.Name] = v.Type
			}

			if constructor, ok := engines[manifest.Runtime]; ok {
				handler, err := constructor(manifestFile, vars)
				if err != nil {
					logger.Error("constructor failed",
						"plugin_url", pluginUrl,
						"plugin_name", manifest.Name,
						"command", manifest.Command,
						"err", err)
					continue
				}

				manifest.Handler = handler

				if !yield(manifest) {
					return
				}
			}
		}
	}
}
