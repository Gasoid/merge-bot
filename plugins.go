package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/config"
	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/logger"
	"github.com/stretchr/testify/assert/yaml"
)

const (
	githubURL = "https://github.com/"
	gitlabURL = "https://gitlab.com/"
)

var (
	plugins string
)

type handlerFunc func(command *handlers.Request, args string) error

func init() {
	config.StringVar(&plugins, "plugins", "", "comma list of plugin urls (also via PLUGINS)")
}

type PluginConfig struct {
	ExportedFunction string   `yaml:"exported_function"`
	Path             string   `yaml:"path"`
	EnvVars          []string `yaml:"env_vars"`
	AllowedHosts     []string `yaml:"allowed_hosts"`
}

type PluginManifest struct {
	Name    string       `yaml:"name"`
	Command string       `yaml:"command"`
	Config  PluginConfig `yaml:"config"`
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

func getManifest(filePath string) (PluginManifest, error) {
	manifest := PluginManifest{}

	switch {
	case strings.HasPrefix(filePath, githubURL):
		filePath = strings.Replace(filePath, githubURL, "https://raw.githubusercontent.com", 1)
		filePath = strings.Replace(filePath, "blob", "refs/heads", 1)

	case strings.HasPrefix(filePath, gitlabURL):
		filePath = strings.Replace(filePath, "-/blob/", "-/raw/", 1)

	}

	body, err := getFile(filePath)
	if err != nil {
		return manifest, err
	}

	if err := yaml.Unmarshal(body, &manifest); err != nil {
		return manifest, err
	}

	return manifest, nil
}

func loadPlugins() {
	if plugins == "" {
		return
	}

	for pluginUrl := range strings.SplitSeq(plugins, ",") {
		pluginUrl = strings.TrimSpace(pluginUrl)
		if pluginUrl == "" {
			logger.Info("loadPlugins", "err", "url is empty")
			continue
		}

		manifest, err := getManifest(pluginUrl)
		if err != nil {
			logger.Error("getManifest", "plugin_url", pluginUrl, "err", err)
			continue
		}

		handler, err := buildWasmPlugin(manifest)
		if err != nil {
			logger.Error("buildWasmPlugin",
				"plugin_url", pluginUrl,
				"plugin_name", manifest.Name,
				"command", manifest.Command,
				"err", err)
			continue
		}

		logger.Info("plugin loaded", "plugin name", manifest.Name)
		handle(manifest.Command, handler)
	}
}

func buildWasmPlugin(manifest PluginManifest) (handlerFunc, error) {
	ctx := context.Background()
	envMap := map[string]string{}
	for _, v := range manifest.Config.EnvVars {
		envMap[v] = os.Getenv(strings.ToUpper(v)) // TODO: config.StringVar
	}

	extismManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{
				Path: manifest.Config.Path,
			},
		},
		AllowedHosts: manifest.Config.AllowedHosts,
		Config:       envMap,
	}

	config := extism.PluginConfig{
		EnableWasi: true,
	}

	compiledPlugin, err := extism.NewCompiledPlugin(ctx, extismManifest, config, []extism.HostFunction{})
	if err != nil {
		return nil, err
	}

	//nolint:errcheck
	return func(command *handlers.Request, _ string) error {
		plugin, err := compiledPlugin.Instance(ctx, extism.PluginInstanceConfig{})
		if err != nil {
			return fmt.Errorf("can't create instance of plugin %s, error %w", manifest.Name, err)
		}
		defer plugin.Close(ctx)

		return command.RunWithContext(func(ctx []byte) error {
			exit, out, err := plugin.Call(manifest.Config.ExportedFunction, ctx)
			if err != nil {
				return fmt.Errorf("plugin %s returns error: %w", manifest.Name, err)
			}

			if exit != 0 {
				return fmt.Errorf("plugin %s returns exit code: %d", manifest.Name, exit)
			}

			errMessage := plugin.GetError()
			if errMessage != "" {
				return fmt.Errorf("plugin %s returns error: %s", manifest.Name, errMessage)
			}

			if len(out) == 0 {
				return fmt.Errorf("plugin %s returns nothing", manifest.Name)
			}
			return command.LeaveComment(string(out))
		})

	}, nil
}
