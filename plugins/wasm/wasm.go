package wasm

import (
	"context"
	"errors"
	"fmt"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/plugins"
	"github.com/stretchr/testify/assert/yaml"
)

type PluginWasmConfig struct {
	ExportedFunction string   `yaml:"exported_function"`
	Path             string   `yaml:"path"`
	Url              string   `yaml:"url"`
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

func BuildWasmPlugin(manifestFile []byte, vars map[string][]string) (plugins.HandlerFunc, error) {
	manifest := PluginManifest{}

	if err := yaml.Unmarshal(manifestFile, &manifest); err != nil {
		return nil, err
	}

	ctx := context.Background()

	if manifest.WasmConfig.Path == "" && manifest.WasmConfig.Url == "" {
		return nil, errors.New("either Path or Url must be set")
	}

	var wasmPath extism.Wasm
	if manifest.WasmConfig.Path != "" {
		wasmPath = extism.WasmFile{
			Path: manifest.WasmConfig.Path,
		}
	} else {
		wasmPath = extism.WasmUrl{
			Url: plugins.GetRawLink(manifest.WasmConfig.Url),
		}
	}

	extismManifest := extism.Manifest{
		Wasm: []extism.Wasm{
			wasmPath,
		},
		AllowedHosts: manifest.WasmConfig.AllowedHosts,
		//Config:       envMap,
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
		}, vars)

	}, nil
}
