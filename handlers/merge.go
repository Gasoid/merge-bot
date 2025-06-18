package handlers

import (
	"fmt"
	"mergebot/logger"
	"net/url"
	"os"

	"github.com/ldez/go-git-cmd-wrapper/v2/checkout"
	"github.com/ldez/go-git-cmd-wrapper/v2/clone"
	"github.com/ldez/go-git-cmd-wrapper/v2/config"
	"github.com/ldez/go-git-cmd-wrapper/v2/git"
	"github.com/ldez/go-git-cmd-wrapper/v2/merge"
	"github.com/ldez/go-git-cmd-wrapper/v2/push"
)

const (
	defaultRemote = "origin"
)

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
		logger.Debug("temp dir error")
		return fmt.Errorf("temp dir error: %w", err)
	}

	defer os.RemoveAll(dir)

	if output, err := git.Clone(clone.Repository(repoUrl), clone.Directory(dir)); err != nil {
		logger.Debug("git clone error", "dir", dir, "output", output)
		return fmt.Errorf("git clone error: %w, output: %s", err, output)
	}

	if err := os.Chdir(dir); err != nil {
		logger.Debug("chdir error")
		return fmt.Errorf("chdir error: %w", err)
	}

	if output, err := git.Config(config.Entry("user.email", fmt.Sprintf("%s@localhost", username))); err != nil {
		logger.Debug("git config error", "user.email", fmt.Sprintf("%s@localhost", username), "output", output)
		return fmt.Errorf("git config error: %w, output: %s", err, output)
	}

	if output, err := git.Config(config.Entry("user.name", username)); err != nil {
		logger.Debug("git config error", "user.name", username, "output", output)
		return fmt.Errorf("git config error: %w, output: %s", err, output)
	}

	if output, err := git.Checkout(checkout.Branch(branchName)); err != nil {
		logger.Debug("git checkout error", "branch", branchName, "output", output)
		return fmt.Errorf("git checkout error: %w, output: %s", err, output)
	}

	if output, err := git.Merge(merge.Commits(master), merge.M("update from master")); err != nil {
		logger.Debug("git merge error", "output", output)
		if output, err := git.Merge(merge.NoFf, merge.Commits(master), merge.M("update from master")); err != nil {
			logger.Debug("git merge --noff error", "output", output)
			return fmt.Errorf("git merge --noff error: %w, output: %s", err, output)
		}
	}

	if output, err := git.Push(push.Remote(defaultRemote), push.RefSpec(branchName)); err != nil {
		logger.Debug("git push error", "output", output)
		return fmt.Errorf("git push error: %w, output: %s", err, output)
	}

	return nil
}
