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

func UpdateBranchCmd(command *handlers.Request, hook *webhook.Webhook) error {
	if err := command.UpdateFromMaster(hook.GetProjectID(), hook.GetID()); err != nil {
		logger.Error("command.UpdateFromMaster failed", "error", err)
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), "❌ i couldn't update branch from master")
	}

	return nil
}

func MergeCmd(command *handlers.Request, hook *webhook.Webhook) error {
	ok, text, err := command.Merge(hook.GetProjectID(), hook.GetID())
	if err != nil {
		return fmt.Errorf("command.Merge returns err: %w", err)
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), text)
	}
	return err
}

func CheckCmd(command *handlers.Request, hook *webhook.Webhook) error {
	ok, text, err := command.IsValid(hook.GetProjectID(), hook.GetID())
	if err != nil {
		return fmt.Errorf("command.IsValid returns err: %w", err)
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), text)
	} else {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), "You can merge, LGTM :D")
	}
}

func NewMR(command *handlers.Request, hook *webhook.Webhook) error {
	if err := command.Greetings(hook.GetProjectID(), hook.GetID()); err != nil {
		return fmt.Errorf("command.Greetings returns err: %w", err)
	}

	return nil
}

func MergeEvent(command *handlers.Request, hook *webhook.Webhook) error {
	if err := command.DeleteStaleBranches(hook.GetProjectID(), hook.GetID()); err != nil {
		return fmt.Errorf("command.MergeEvent returns err: %w", err)
	}

	return nil
}
