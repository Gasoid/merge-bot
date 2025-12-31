# Plugin support
You can extend the functionality of the Merge-Bot by creating and using plugins. Plugins are WebAssembly (WASM) modules that can be loaded at runtime to provide additional features.

Please read extism documentation to learn how to create WASM plugins: https://extism.org/docs/getting-started/wasm-plugins


## Plugin manifest for WASM

```yaml
name: Plugin Name
command: "!plugin-command" # Command to trigger the plugin, e.g. !review
runtime: "wasm"

wasm_config:
  exported_function: "review"
  path: "plugin.wasm" # Path to the compiled WASM file
  env_vars:
  - plugin_env_var_1 # Environment variables to pass to the plugin
  allowed_hosts:
  - "host.com" # Hosts that the plugin is allowed to access

```

Set up environement variable or config argument `PLUGINS` with a list of plugin urls.

```bash
PLUGINS="plugin1.yaml,https://example.com/plugin2.yaml,https://github.com/user/repo/main/plugin3.yaml"
```

The bot will download and load the plugins at startup.

## Demo plugin

To compile plugin:

```bash
GOOS="wasip1" GOARCH="wasm" go build -o plugin.wasm -buildmode=c-shared demo.go
```

Set env variable:

```bash
export DEMO_NAME="hello-plugin"
export PLUGINS="https://github.com/Gasoid/merge-bot/blob/wasm_plugin_support/plugins/demo/demo.yaml"
```
