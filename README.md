# MergeBot: PR/MR bot

![screen](screen.webp)

### Features
- rule for title
- rule for approvals
- rule for approvers
- merge on command
- update branch
- delete stale branches


### Commands
- !merge
- !check
- !update

### Setup 
1. Invite bot ([@mergeapprovebot](https://gitlab.com/mergeapprovebot)) in your repository as **maintainer** (you can revoke permissions from usual developers in order to prevent merging)
2. Add webhook `https://mergebot.tools/mergebot/webhook/gitlab/your_username_or_company_name/repo-name/` (Comments and merge request events)
3. PROFIT: now you can create MR, leave commands: !check and then !merge (comment in MR)

### Quickstart on your env

Create personal/repo/org token in gitlab, copy it and set as env variable
```bash
export GITLAB_TOKEN="your_token"
export GITLAB_URL="" # if it is not public gitlab cloud
export TLS_ENABLED="true"
export TLS_DOMAIN="bot.domain.com"
```

Run bot
```
go run ./
```

### Build
```
go build ./
```



## Config file

Config file must be named `.mrbot.yaml`, placed in root directory, default branch (main/master)

```yaml
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

Example:

```yaml
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
