# MergeBot: MR bot for Gitlab

[![Docker Image Version](https://img.shields.io/docker/v/gasoid/merge-bot?style=flat-square&label=docker&sort=semver)](https://hub.docker.com/r/gasoid/merge-bot)
[![Helm Chart Version](https://img.shields.io/github/v/release/gasoid/helm-charts?style=flat-square&label=helm&filter=merge-bot-*)](https://github.com/gasoid/helm-charts/releases)
[![Latest Release](https://img.shields.io/github/v/release/gasoid/merge-bot?style=flat-square)](https://github.com/gasoid/merge-bot/releases/latest)

![screen](screen.webp)

## Features
- [x] rule for title
- [x] rule for approvals
- [x] rule for approvers
- [x] merge on command
- [x] update branch (pull changes from master)
- [x] delete stale branches


## Table of Contents

- [Installation](#installation)
  - [Gitlab Cloud](#gitlab-cloud)
  - [Docker-compose](#Docker-compose)
  - [Helm](#helm)
  - [CLI](#cli)
- [Required bot permissions](#required-bot-permissions)
- [Webhook secret](#webhook-secret)
- [Stale branches](#stale-branches)
- [Config file](#config-file)
  - [Example](#example)
  - [Demo project on gitlab](https://gitlab.com/Gasoid/sugar-test)


### Demo repo

https://gitlab.com/Gasoid/sugar-test

### Commands
- `!merge`: merges MR if the MR meets rules of the repo
- `!check`: checks whether the MR meets rules of the repo
- `!update`: updates the branch from destination branch (e.g. master) changes

### Use-cases
- Given a lot of repos,  therefore we require to set up various rules for each of them. It is complicated and tedious to run as many bot instances as teams. The Merge-bot checks whether MRs meet rules of the repository (.mrbot.yaml file). Owner of repo can create his own set of rules.

- Opensource solution to get premium features

## Installation
The Bot could be run within your infrastructure as container.
In case you want to test the bot you can use gitlab cloud bot.


### Gitlab Cloud
1. Invite bot ([@mergeapprovebot](https://gitlab.com/mergeapprovebot)) in your repository as **maintainer** (you can revoke permissions from usual developers in order to prevent merging)
2. Add webhook `https://mergebot.tools/mergebot/webhook/gitlab/` (Comments and merge request events)
3. PROFIT: now you can create MR, leave commands: !check and then !merge (comment in MR)

You can test bot on gitlab public repo: https://gitlab.com/Gasoid/sugar-test

### Docker-compose

1. bot.env:
```
GITLAB_TOKEN="your_token"
#TLS_ENABLED="false"
#TLS_DOMAIN="domain.your-example.com"
#GITLAB_URL=""
```

2. run docker-compose
```
docker-compose up -d
```

3. set up webhook please follow these instruction [Gitlab Cloud](#gitlab-cloud)

### Helm

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add merge-bot https://gasoid.github.io/helm-charts

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo merge-bot` to see the charts.

To install the merge-bot chart:

    helm install my-merge-bot merge-bot/merge-bot

To uninstall the chart:

    helm uninstall my-merge-bot

In order to set up webhook, please read [Gitlab Cloud](#gitlab-cloud)

### CLI

Create personal/repo/org token in gitlab, copy it and set as env variable
```bash
export GITLAB_TOKEN="your_token"
export GITLAB_URL="" # if it is not public gitlab cloud
export TLS_ENABLED="true"
export TLS_DOMAIN="bot.domain.com"
```

you can configure bot using cli args as well:
```bash
Usage of merge-bot:
  -debug
    	whether debug logging is enabled, default is false (also via DEBUG)
  -gitlab-token string
    	in order to communicate with gitlab api, bot needs token (also via GITLAB_TOKEN)
  -gitlab-url string
    	in case of self-hosted gitlab, you need to set this var up (also via GITLAB_URL)
  -gitlab-max-repo-size string
      max size of repo in in Gb/Mb/Kb, default is 500Mb (also via GITLAB_MAX_REPO_SIZE)
  -tls-domain string
    	which domain is used for ssl certificate (also via TLS_DOMAIN)
  -tls-enabled
    	whether tls enabled or not, bot will use Letsencrypt, default is false (also via TLS_ENABLED)
  -sentry-enabled
      whether sentry enabled or not, default is true (also via SENTRY_ENABLED)
```

Run bot
```
go run ./
```

### Required bot permissions
- Bot must have __Maintainer__ role in order to comment, merge and delete branches
- Access Token must have following permissions: api, read_repository, write_repository

### Webhook secret
You can enforce security by using `secret`.

1. You need to create CI/CD var `MERGE_BOT_SECRET` in your project with your secure/random value. This var will be compared with webhook secret.
2. Set up the same webhook secret as `MERGE_BOT_SECRET` value.

The bot will read `MERGE_BOT_SECRET` value, if it doesn't exist, it will be considered as empty string ("").

### Stale branches
If `stale branches deletion` feature is enabled, deletion of stale branches will work.
The bot deletes stale branches once a MR is merged.



## Config file

Config file must be named `.mrbot.yaml`, placed in root directory, default branch (main/master)

```yaml
rules:
  approvers: [] # list of users who must approve MR/PR, default is empty ([])
  min_approvals: 1 # minimum number of required approvals, default is 1
  allow_empty_description: true # whether MR description is allowed to be empty or not, default is true
  allow_failing_pipelines: true # whether pipelines are allowed to fail, default is true
  title_regex: ".*" # pattern of title, default is ".*"

greetings:
  enabled: false # enable message for new MR, default is false
  template: "" # template of message for new MR, default is "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nOnce you've done, send **!merge** command and i will merge it!"

auto_master_merge: false # the bot tries to update branch from master, default is false

stale_branches_deletion:
  enabled: false # enable deletion of stale branches after every merge, default is false
  days: 90 # branch is staled after int days, default is 90
```

#### Example:

```yaml
rules:
  approvers:
    - user1
    - user2
  min_approvals: 1
  allow_empty_description: true
  allow_failing_pipelines: true
  allow_failing_tests: true
  title_regex: "^[A-Z]+-[0-9]+" # title begins with jira key prefix, e.g. SCO-123 My cool Title

greetings:
  enabled: true
  template: "Requirements:\n - Min approvals: {{ .MinApprovals }}\n - Title regex: {{ .TitleRegex }}\n\nOnce you've done, send **!merge** command and i will merge it!"

auto_master_merge: true

stale_branches_deletion:
  enabled: true
  days: 90
```

place it in root of your repo and name it `.mrbot.yaml`
