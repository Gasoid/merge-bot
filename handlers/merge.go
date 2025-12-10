package handlers

import (
	"fmt"
	"net/url"
	"os"

	"github.com/gasoid/merge-bot/logger"

	"github.com/ldez/go-git-cmd-wrapper/v2/checkout"
	"github.com/ldez/go-git-cmd-wrapper/v2/clone"
	"github.com/ldez/go-git-cmd-wrapper/v2/config"
	"github.com/ldez/go-git-cmd-wrapper/v2/fetch"
	"github.com/ldez/go-git-cmd-wrapper/v2/git"
	"github.com/ldez/go-git-cmd-wrapper/v2/global"
	"github.com/ldez/go-git-cmd-wrapper/v2/merge"
	"github.com/ldez/go-git-cmd-wrapper/v2/push"
)

const (
	defaultRemote = "origin"
)

type MergeError struct {
	SourceBranch      string
	DestinationBranch string
}

func (e *MergeError) Error() string {
	return "you have to merge your destination branch manually"
}

//nolint:errcheck
func MergeMaster(username, password, repoUrl, branchName, master string) error {

	if username != "" && password != "" {
		parsedUrl, err := url.Parse(repoUrl)
		if err != nil {
			return err
		}
		parsedUrl.User = url.UserPassword(username, password)
		repoUrl = parsedUrl.String()
	}

	dir, err := os.MkdirTemp("", "merge-bot")
	if err != nil {
		logger.Debug("temp dir error", "error", err)
		return fmt.Errorf("temp dir error: %w", err)
	}

	workingDir := global.UpperC(dir)

	defer os.RemoveAll(dir)

	if output, err := git.Clone(clone.Repository(repoUrl), clone.Directory(dir)); err != nil {
		logger.Debug("git clone error", "dir", dir, "output", output)
		return fmt.Errorf("git clone error: %w, output: %s", err, output)
	}

	if output, err := git.Config(workingDir, config.Entry("user.email", fmt.Sprintf("%s@localhost", username))); err != nil {
		logger.Debug("git config error", "user.email", fmt.Sprintf("%s@localhost", username), "output", output)
		return fmt.Errorf("git config error: %w, output: %s", err, output)
	}

	if output, err := git.Config(workingDir, config.Entry("user.name", username)); err != nil {
		logger.Debug("git config error", "user.name", username, "output", output)
		return fmt.Errorf("git config error: %w, output: %s", err, output)
	}

	if output, err := git.Fetch(workingDir, fetch.All); err != nil {
		logger.Debug("git fetch error", "branch", branchName, "output", output)
		return fmt.Errorf("git fetch error: %w, output: %s", err, output)
	}

	if output, err := git.Checkout(workingDir, checkout.Branch(branchName)); err != nil {
		logger.Debug("git checkout error", "branch", branchName, "output", output)
		return fmt.Errorf("git checkout error: %w, output: %s", err, output)
	}

	remoteTargetBranch := fmt.Sprintf("%s/%s", defaultRemote, master)
	if output, err := git.Merge(workingDir, merge.Commits(remoteTargetBranch), merge.M(fmt.Sprintf("✨ merged %s", master))); err != nil {
		logger.Debug("git merge error", "output", output)
		if output, err := git.Merge(workingDir, merge.NoFf, merge.Commits(remoteTargetBranch), merge.M(fmt.Sprintf("✨ merged %s", master))); err != nil {
			logger.Debug("git merge --no-ff error", "output", output)

			mergeError := &MergeError{
				DestinationBranch: master,
				SourceBranch:      branchName,
			}

			return fmt.Errorf("git merge --no-ff error: %w, output: %s", mergeError, output)
		}
	}

	if output, err := git.Push(workingDir, push.Remote(defaultRemote), push.RefSpec(branchName)); err != nil {
		logger.Debug("git push error", "output", output)
		return fmt.Errorf("git push error: %w, output: %s", err, output)
	}

	return nil
}
