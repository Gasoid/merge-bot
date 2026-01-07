# Plugin support
You can extend the functionality of the Merge-Bot by creating and using plugins. Plugins are WebAssembly (WASM) modules that can be loaded at runtime to provide additional features.

Please read extism documentation to learn how to create WASM plugins: https://extism.org/docs/getting-started/wasm-plugins


## Plugin manifest for WASM

```yaml
name: Plugin Name
command: "!plugin-command" # Command to trigger the plugin, e.g. !review
runtime: "wasm"

vars:
- name: plugin_env_var_1 # Environment variables to pass to the plugin
  type: ["env", "config", "secret"] # source types: env - from environment variable, config - from .mrbot.yaml config file, secret - from CI/CD secret

wasm_config:
  exported_function: "review"
  url: "https://github.com/user/repo/plugin-file.yaml" # either url or path must be set
  path: "/path/to/plugin.wasm" # Path to the compiled WASM file
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

### Helm chart configuration:

Plugin config

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-config
data:
  demo-plugin.yaml: |
    name: Hello plugin
    command: "!hello"
    runtime: "wasm"
    vars:
    - name: DEMO_NAME
      type: ["env"]

    wasm_config:
      exported_function: "hello"
      url: "https://github.com/Gasoid/merge-bot/blob/v3.8.0-alpha.1/plugins/demo/plugin.wasm"
      allowed_hosts:
      - "api.host.com"

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugins
data:
  PLUGINS: "/app/demo-plugin.yaml"

```

merge-bot values.yaml

```yaml
mergebot:
  envFrom:
  - configMapRef:
      name: plugins

volumes:
  - name: demo-volume
    configMap:
      name: plugin-config


volumeMounts:
  - name: demo-volume
    mountPath: /app/demo-plugin.yaml
    subPath: demo-plugin.yaml

```
