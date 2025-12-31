package wasm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/plugins"
	"github.com/stretchr/testify/assert/yaml"
)

type PluginWasmConfig struct {
	ExportedFunction string   `yaml:"exported_function"`
	Path             string   `yaml:"path"`
	EnvVars          []string `yaml:"env_vars"`
	AllowedHosts     []string `yaml:"allowed_hosts"`
}

type PluginManifest struct {
	Name       string           `yaml:"name"`
	Command    string           `yaml:"command"`
	WasmConfig PluginWasmConfig `yaml:"wasm_config"`
}

func init() {
	plugins.Register("wasm", BuildWasmPlugin)
}

func BuildWasmPlugin(manifestFile []byte) (plugins.HandlerFunc, error) {
	manifest := PluginManifest{}

	if err := yaml.Unmarshal(manifestFile, &manifest); err != nil {
		return nil, err
	}

	ctx := context.Background()
	envMap := map[string]string{}
	for _, v := range manifest.WasmConfig.EnvVars {
		envMap[v] = os.Getenv(strings.ToUpper(v)) // TODO: config.StringVar
	}

	if manifest.WasmConfig.Path == "" {
		return nil, errors.New("WasmConfig.Path is empty")
	}

	wasmPath := plugins.GetRawLink(manifest.WasmConfig.Path)

	extismManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{
				Path: wasmPath,
			},
		},
		AllowedHosts: manifest.WasmConfig.AllowedHosts,
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

		return command.RunWithContext(func(input []byte) ([]byte, error) {
			exit, output, err := plugin.Call(manifest.WasmConfig.ExportedFunction, input)
			if err != nil {
				return nil, fmt.Errorf("plugin %s returns error: %w", manifest.Name, err)
			}

			if exit != 0 {
				return nil, fmt.Errorf("plugin %s returns exit code: %d", manifest.Name, exit)
			}

			errMessage := plugin.GetError()
			if errMessage != "" {
				return nil, fmt.Errorf("plugin %s returns error: %s", manifest.Name, errMessage)
			}

			if len(output) == 0 {
				return nil, fmt.Errorf("plugin %s returns nothing", manifest.Name)
			}

			return output, nil
		})

	}, nil
}
