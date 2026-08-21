package wasm

import (
	"context"
	"errors"
	"fmt"

	extism "github.com/extism/go-sdk"
	"github.com/gasoid/merge-bot/v3/handlers"
	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/gasoid/merge-bot/v3/plugins"
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

var (
	getGitFile = extism.NewHostFunctionWithStack(
		"get_git_file",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			provider, err := p.ReadString(stack[0])
			if err != nil {
				logger.Info("getGitFile can't read bytes", "error", err)
				return
			}

			command, err := handlers.New(provider)
			if err != nil {
				logger.Info("getGitFile can't create Request instance", "error", err)
				return
			}

			projectID, ID := int64(stack[1]), int64(stack[2])

			if err := command.LoadInfoAndConfig(projectID, ID); err != nil {
				logger.Error("can't load repo config", "provider", provider, "command", command, "err", err)
				return
			}

			filePath, err := p.ReadString(stack[3])
			if err != nil {
				logger.Info("getGitFile can't read bytes", "error", err)
				return
			}

			data, err := command.GetFile(filePath)
			if err != nil {
				logger.Info("getGitFile can't receive file", "error", err, "filePath", filePath)
				return
			}

			stack[0], err = p.WriteBytes(data)
			if err != nil {
				logger.Info("getGitFile can't write data", "error", err, "filePath", filePath)
				return
			}
		},
		[]extism.ValueType{extism.ValueTypePTR, extism.ValueTypeI64, extism.ValueTypeI64, extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
)

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

	compiledPlugin, err := extism.NewCompiledPlugin(ctx, extismManifest, config, []extism.HostFunction{getGitFile})
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
