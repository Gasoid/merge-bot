package main

import (
	"fmt"

	"github.com/Gasoid/mergebot/handlers"
	"github.com/Gasoid/mergebot/logger"
	"github.com/Gasoid/mergebot/webhook"
)

func init() {
	handle("!merge", MergeCmd)
	handle("!check", CheckCmd)
	handle("!update", UpdateBranchCmd)
	handle(webhook.OnNewMR, NewMR)
	handle(webhook.OnMerge, MergeEvent)
}

func UpdateBranchCmd(command *handlers.Request) error {
	if err := command.UpdateFromMaster(); err != nil {
		logger.Error("command.UpdateFromMaster failed", "error", err)
		return command.LeaveComment("❌ i couldn't update branch from master")
	}

	return nil
}

func MergeCmd(command *handlers.Request) error {
	ok, text, err := command.Merge()
	if err != nil {
		return fmt.Errorf("command.Merge returns err: %w", err)
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(text)
	}
	return err
}

func CheckCmd(command *handlers.Request) error {
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

func NewMR(command *handlers.Request) error {
	if err := command.Greetings(); err != nil {
		return fmt.Errorf("command.Greetings returns err: %w", err)
	}

	return nil
}

func MergeEvent(command *handlers.Request) error {
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
