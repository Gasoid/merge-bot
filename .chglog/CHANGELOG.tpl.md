{{ range .Versions }}

MergeBot is an automated merge request bot for GitLab.

![screen](screen.webp)

## Installation

### Docker Compose

1. **Create environment file** (`bot.env`):
```env
GITLAB_TOKEN=your_personal_access_token
# Optional: Enable TLS
# TLS_ENABLED=true
# TLS_DOMAIN=your-domain.com
# GITLAB_URL=https://your-gitlab-instance.com
```

2. **Run the container**:
```bash
docker-compose up -d
```

### Helm

Add the Helm repository:
```bash
helm repo add merge-bot https://gasoid.github.io/helm-charts
helm repo update
```

Install the chart:
```bash
helm install my-merge-bot merge-bot/merge-bot
```


## Available Commands

- `!merge` - Merges MR if all repository rules are satisfied
- `!check` - Validates whether the MR meets all rules
- `!update` - Updates the branch from the target branch (e.g., main/master)
- `!rerun` - Re-run pipeline, e.g. `!rerun #123123333` or `!rerun 123123333`, command will run pipeline against the branch of the merge request with variables of provided pipeline (e.g. 123123333)

## Plugin support
You can extend the functionality of the Merge-Bot by creating and using plugins. Plugins are WebAssembly (WASM) modules that can be loaded at runtime to provide additional features.

See [docs](plugins.md)


{{ range .CommitGroups -}}
### {{ .Title }}

{{ range .Commits -}}
* {{ .Subject }}
{{ end }}
{{ end -}}

{{- if .RevertCommits -}}
### Reverts

{{ range .RevertCommits -}}
* {{ .Revert.Header }}
{{ end }}
{{ end -}}

{{- if .MergeCommits -}}
### Pull Requests

{{ range .MergeCommits -}}
* {{ .Header }}
{{ end }}
{{ end -}}

{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}

{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}