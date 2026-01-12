# MergeBot: Merge Request Bot for GitLab

[![Docker Image Version](https://img.shields.io/docker/v/gasoid/merge-bot?style=flat-square&label=docker&sort=semver)](https://hub.docker.com/r/gasoid/merge-bot)
[![Helm Chart Version](https://img.shields.io/github/v/release/gasoid/helm-charts?style=flat-square&label=helm&filter=merge-bot-*)](https://github.com/gasoid/helm-charts/releases)
[![Latest Release](https://img.shields.io/github/v/release/gasoid/merge-bot?style=flat-square)](https://github.com/gasoid/merge-bot/releases/latest)

![screen](screen.webp)

MergeBot is an automated merge request bot for GitLab that enforces repository-specific rules and helps streamline your code review process.

## Features

- ✅ **Title validation** - Enforce naming conventions with regex patterns
- ✅ **Approval rules** - Set minimum approvals and specific approvers
- ✅ **Automated merging** - Merge on command when rules are met
- ✅ **Branch updates** - Automatically pull changes from target branch
- ✅ **Stale branch cleanup** - Remove outdated branches automatically
- ✅ **Customizable rules** - Per-repository configuration via `.mrbot.yaml`

## Quick Start

Try the bot on our [demo repository](https://gitlab.com/Gasoid/sugar-test) or invite [@mergeapprovebot](https://gitlab.com/mergeapprovebot) to your project.

### Available Commands

- `!merge` - Merges MR if all repository rules are satisfied
- `!check` - Validates whether the MR meets all rules
- `!update` - Updates the branch from the target branch (e.g., main/master)
- `!rerun` - Re-run pipeline, e.g. `!rerun #123123333` or `!rerun 123123333`, command will run pipeline against the branch of the merge request with variables of provided pipeline (e.g. 123123333)

## Table of Contents

- [Installation](#installation)
  - [GitLab Cloud](#gitlab-cloud)
  - [Docker Compose](#docker-compose)
  - [Helm](#helm)
  - [CLI](#cli)
- [Configuration](#configuration)
  - [Required Bot Permissions](#required-bot-permissions)
  - [Webhook Secret](#webhook-secret)
  - [Config File](#config-file)
- [Features](#features-1)
  - [Stale Branches](#stale-branches)
  - [Greetings](#greetings)
- [Demo](#demo)

## Installation

### GitLab Cloud

1. **Invite the bot**: Add [@mergeapprovebot](https://gitlab.com/mergeapprovebot) to your repository with **Maintainer** role
2. **Configure webhook**: 
   - URL: `https://mergebot.tools/mergebot/webhook/gitlab/`
   - Trigger events: Comments and Merge Request events
3. **Create configuration**: Add `.mrbot.yaml` to your repository root (see [Config File](#config-file))
4. **Start using**: Create an MR and use commands like `!check` and `!merge`

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

3. **Configure webhook**: Follow the [GitLab Cloud](#gitlab-cloud) instructions, but use your own bot URL

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

For webhook configuration, follow the [GitLab Cloud](#gitlab-cloud) instructions.

### CLI

1. **Set environment variables**:
```bash
export GITLAB_TOKEN="your_token"
export GITLAB_URL=""  # Optional: for self-hosted GitLab
export TLS_ENABLED="true"  # Optional
export TLS_DOMAIN="bot.yourdomain.com"  # Optional
```

2. **Run the bot**:
```bash
go run ./
```

**Available CLI flags**:
```
  -debug
        Enable debug logging (also via DEBUG)
  -gitlab-token string
        GitLab personal access token (also via GITLAB_TOKEN)
  -gitlab-url string
        GitLab instance URL for self-hosted (also via GITLAB_URL)
  -gitlab-max-repo-size string
        Maximum repository size (default: 500Mb, also via GITLAB_MAX_REPO_SIZE)
  -tls-domain string
        Domain for SSL certificate (also via TLS_DOMAIN)
  -tls-enabled
        Enable TLS with Let's Encrypt (also via TLS_ENABLED)
  -sentry-enabled
        Enable Sentry error reporting (default: true, also via SENTRY_ENABLED)
  -plugins string
        Comma-separated list of plugin config URLs or paths (also via PLUGINS)
  -version
      	Shows version and build time
```

## Configuration

### Required Bot Permissions

- **Bot role**: Maintainer (required for commenting, merging, and deleting branches)
- **Access token scopes**: `api`, `read_repository`, `write_repository`

### Webhook Secret

Enhance security by using webhook secrets:

1. **Create CI/CD variable**: Add `MERGE_BOT_SECRET` to your GitLab project variables
2. **Configure webhook**: Set the same secret value in your webhook configuration
3. **Verification**: The bot will validate incoming webhooks against this secret

### Config File

Create `.mrbot.yaml` in your repository root on the default branch:

```yaml
# all settings are optional, defaults are shown below

rules:
  approvers: []  # Specific users who must approve (empty = any approver)
  min_approvals: 1  # Minimum number of approvals required
  allow_empty_description: true  # Allow empty MR descriptions
  allow_failing_pipelines: true  # Allow merging with failed pipelines
  title_regex: ".*"  # Title validation regex pattern
  reset_approvals_on_push:
    enabled: false  # Reset approvals on new commits
    issue_token: true # Whether token will be created or current GITLAB_TOKEN will be used
    project_var_name: MergeBot

greetings:
  enabled: false  # Send welcome message on new MRs
  resolvable: false # Whether greeting message can be updated and resolved
  template: "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nSend **!merge** when ready!"

auto_master_merge: false  # Auto-update branch from target branch

stale_branches_deletion:
  enabled: false  # Clean up stale branches after merge
  protected: false # Whether to consider protected branches for deletion
  days: 90  # Consider branches stale after N days
  batch_size: 5 # Number of branches can be deleted at once
  wait_days: 1 # Wait N days before MR/branch deletion, merge-bot:stale label is set

plugin_vars: {}  # Custom variables for plugins
```

#### Example Configuration

```yaml
rules:
  approvers:
    - alice
    - bob
  min_approvals: 2
  allow_empty_description: false
  allow_failing_pipelines: false
  title_regex: "^(feat|fix|docs|style|refactor|test|chore):"  # Conventional commits
  reset_approvals_on_push:
    enabled: true  # Reset approvals on new commits

greetings:
  enabled: true
  resolvable: true
  template: |
    ## 🤖 MergeBot Requirements
    
    - **Min approvals**: {{ .MinApprovals }}
    - **Title format**: {{ .TitleRegex }}
    - **Approvers**: {{ .Approvers }}
    
    Use `!check` to validate and `!merge` when ready!

auto_master_merge: true

stale_branches_deletion:
  enabled: true
  protected: true
  days: 30
  batch_size: 2
  wait_days: 1

plugin_vars:
  var_name: value  # Custom variables for plugins
```

## Features

### Plugin support
Please read [docs](plugins.md) for more information.

### Stale Branches

When enabled, the bot automatically deletes stale branches after each successful merge/update. Branches are considered stale based on the configured number of days since their last activity.

### Greetings

Customize welcome messages for new merge requests using Go templates. Available variables:
- `{{ .MinApprovals }}`
- `{{ .TitleRegex }}`
- `{{ .Approvers }}`

You can also enable the `resolvable` option to allow the bot to update and resolve the greeting message once all requirements are met. Merge Request will be blocked until requirements are met. (You need to enable "All threads must be resolved" in project settings for this feature to work.)

### Labels

The bot creates 2 labels:
- merge-bot:stale
- merge-bot:auto-update

Use `merge-bot:auto-update` label if you need to update merge request when target branch (master) is updated.

## Demo

Test the bot on our public demo repository: [https://gitlab.com/Gasoid/sugar-test](https://gitlab.com/Gasoid/sugar-test)

## Use Cases

- **Multi-repository management**: Configure different rules per repository without running multiple bot instances
- **Open-source alternative**: Get premium GitLab features without the cost
- **Automated compliance**: Enforce consistent review processes across teams
- **Branch hygiene**: Automatically clean up stale branches

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
