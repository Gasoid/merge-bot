package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gasoid/merge-bot/handlers"
	"github.com/Gasoid/merge-bot/logger"
	"github.com/Gasoid/merge-bot/webhook"
)

func init() {
	handle("!merge", MergeCmd)
	handle("!check", CheckCmd)
	handle("!update", UpdateBranchCmd)
	handle("!rerun", RerunPipeline)
	handle(webhook.OnNewMR, NewMR)
	handle(webhook.OnMerge, MergeEvent)
}

func UpdateBranchCmd(command *handlers.Request, args string) error {
	if err := command.UpdateFromMaster(); err != nil {
		logger.Error("command.UpdateFromMaster failed", "error", err)
		return command.LeaveComment("❌ i couldn't update branch from master")
	}

	return nil
}

func MergeCmd(command *handlers.Request, args string) error {
	ok, text, err := command.Merge()
	if err != nil {
		return fmt.Errorf("command.Merge returns err: %w", err)
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(text)
	}
	return err
}

func CheckCmd(command *handlers.Request, args string) error {
	ok, text, err := command.IsValid()
	if err != nil {
		return fmt.Errorf("command.IsValid returns err: %w", err)
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(text)
	} else {
		return command.LeaveComment("You can merge, LGTM :D")
	}
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

func RerunPipeline(command *handlers.Request, args string) error {
	arg := strings.TrimPrefix(args, "#")
	pipelineId, err := strconv.Atoi(arg)
	if err != nil {
		logger.Debug("rerun", "args", args, "arg", arg)
		return command.LeaveComment("Pipeline ID is wrong")
	}
	if err := command.RerunPipeline(pipelineId); err != nil {
		return command.LeaveComment("Validate your pipeline")
	}
	return nil
}
