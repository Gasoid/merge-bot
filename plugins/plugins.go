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
	engines   = map[string]func([]byte) (HandlerFunc, error){}
	enginesMu sync.RWMutex
)

type HandlerFunc func(command *handlers.Request, args string) error

func init() {
	config.StringVar(&plugins, "plugins", "", "comma list of plugin urls (also via PLUGINS)")
}

type PluginManifest struct {
	Name    string      `yaml:"name"`
	Command string      `yaml:"command"`
	Runtime string      `yaml:"runtime"`
	Handler HandlerFunc `yaml:"-"`
}

func Register(name string, constructor func([]byte) (HandlerFunc, error)) {
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

func getManifest(filePath string) ([]byte, error) {

	switch {
	case strings.HasPrefix(filePath, githubURL):
		filePath = strings.Replace(filePath, githubURL, "https://raw.githubusercontent.com/", 1)
		filePath = strings.Replace(filePath, "blob/", "", 1)

	case strings.HasPrefix(filePath, gitlabURL):
		filePath = strings.Replace(filePath, "-/blob/", "-/raw/", 1)

	}

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

			if constructor, ok := engines[manifest.Runtime]; ok {
				handler, err := constructor(manifestFile)
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
