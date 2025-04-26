package main

import (
	"log/slog"
	"mergebot/handlers"
	"mergebot/webhook"
)

func init() {
	handle("!merge", MergeCmd)
	handle("!check", CheckCmd)
	handle("!update", UpdateBranchCmd)
	handle(webhook.OnNewMR, NewMR)
}

func UpdateBranchCmd(providerName string, hook *webhook.Webhook) error {
	command, err := handlers.New(providerName)
	if err != nil {
		return err
	}

	if err := command.UpdateFromMaster(hook.GetProjectID(), hook.GetID()); err != nil {
		slog.Error("updateBranchCmd", "error", err)
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), "❌ i couldn't update branch from master")
	}

	return err
}

func MergeCmd(providerName string, hook *webhook.Webhook) error {
	command, err := handlers.New(providerName)
	if err != nil {
		return err
	}

	ok, text, err := command.Merge(hook.GetProjectID(), hook.GetID())
	if err != nil {
		return err
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), text)
	}
	return err
}

func CheckCmd(providerName string, hook *webhook.Webhook) error {
	command, err := handlers.New(providerName)
	if err != nil {
		return err
	}

	ok, text, err := command.IsValid(hook.GetProjectID(), hook.GetID())
	if err != nil {
		return err
	}

	if !ok && len(text) > 0 {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), text)
	} else {
		return command.LeaveComment(hook.GetProjectID(), hook.GetID(), "You can merge, LGTM :D")
	}
}

func NewMR(providerName string, hook *webhook.Webhook) error {
	command, err := handlers.New(providerName)
	if err != nil {
		return err
	}

	if err = command.Greetings(hook.GetProjectID(), hook.GetID()); err != nil {
		return err
	}

	return nil
}
