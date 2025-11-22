package main

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gasoid/merge-bot/handlers"
	"github.com/gasoid/merge-bot/logger"
	"github.com/gasoid/merge-bot/webhook"
)

func init() {
	handle("!merge", MergeCmd)
	handle("!check", CheckCmd)
	handle("!update", UpdateBranchCmd)
	handle("!rerun", RerunPipeline)
	handle(webhook.OnNewMR, NewMR)
	handle(webhook.OnMerge, MergeEvent)
	handle(webhook.OnUpdate, UpdateEvent)
}

func UpdateBranchCmd(command *handlers.Request, args string) error {
	const (
		mergeText = `
🛠️ I failed to merge %s into your branch, you have to resolve conflicts manually

<details>
<summary>
How to merge branch manually:
</summary>
<pre><code>git checkout %s
git pull origin
git checkout %s
git merge %s
# resolve conflicts
git push origin</code></pre>

</details>
`
	)
	if err := command.UpdateFromMaster(); err != nil {
		logger.Info("command.UpdateFromMaster failed", "error", err)
		mergeError := &handlers.MergeError{}
		if errors.As(err, &mergeError) {
			text := fmt.Sprintf(
				mergeText,
				mergeError.DestinationBranch,
				mergeError.DestinationBranch,
				mergeError.SourceBranch,
				mergeError.DestinationBranch,
			)

			return command.LeaveComment(text)
		}

		return command.LeaveComment("❌ i couldn't update the branch from the destination")
	}

	return nil
}

func MergeCmd(command *handlers.Request, args string) error {
	ok, text, err := command.Merge()
	if err != nil {
		return fmt.Errorf("command.Merge returns err: %w", err)
	}

	if !ok {
		return command.LeaveComment(text)
	}
	return err
}

func CheckCmd(command *handlers.Request, args string) error {
	_, text, err := command.IsValid()
	if err != nil {
		return fmt.Errorf("command.IsValid returns err: %w", err)
	}

	return command.LeaveNote(text)
}

func NewMR(command *handlers.Request, args string) error {
	if err := command.Greetings(); err != nil {
		return fmt.Errorf("command.Greetings returns err: %w", err)
	}

	return nil
}

func MergeEvent(command *handlers.Request, args string) error {
	if err := command.CreateLabels(); err != nil {
		return fmt.Errorf("command.CreateLabels returns err: %w", err)
	}

	if err := command.UpdateBranches(); err != nil {
		return fmt.Errorf("command.UpdateBranchesWithLabel returns err: %w", err)
	}

	if err := command.DeleteStaleBranches(); err != nil {
		return fmt.Errorf("command.DeleteStaleBranches returns err: %w", err)
	}
	return nil
}

func UpdateEvent(command *handlers.Request, args string) error {
	if err := command.UnresolveDiscussion(); err != nil {
		return err
	}

	parsedTime, err := time.Parse("2006-01-02 15:04:05 UTC", args)
	if err != nil {
		return fmt.Errorf("time.Parse returns err: %w", err)
	}

	if err := command.ResetApprovals(parsedTime); err != nil {
		return fmt.Errorf("command.ResetApprovals returns err: %w", err)
	}
	return nil
}

func RerunPipeline(command *handlers.Request, args string) error {
	arg := strings.TrimPrefix(args, "#")
	pipelineId, err := strconv.Atoi(arg)
	if err != nil {
		logger.Debug("rerun", "args", args, "arg", arg)
		return command.LeaveComment("> [!important]\n> Pipeline ID is invalid or wrong")
	}

	logger.Debug("rerun", "args", args, "arg", arg)
	pipelineURL, err := command.RerunPipeline(pipelineId)
	if err != nil {
		if errors.Is(err, handlers.NotFoundError) {
			return command.LeaveComment("> [!important]\n> Provided pipeline was not found")
		}

		return command.LeaveComment("> [!important]\n> Validate your pipeline syntax")
	}

	parsedUrl, err := url.Parse(pipelineURL)
	if err != nil {
		return command.LeaveComment("> [!important]\n> pipeline created, but provided url is wrong")
	}

	return command.LeaveComment(fmt.Sprintf("🤖 pipeline created: [%s](%s)", path.Base(parsedUrl.Path), pipelineURL))
}
