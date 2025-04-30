package handlers

import (
	"fmt"
	"log/slog"
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
		slog.Debug("temp dir error")
		return err
	}

	defer os.RemoveAll(dir)

	if _, err := git.Clone(clone.Repository(repoUrl), clone.Directory(dir)); err != nil {
		slog.Debug("git clone error", "dir", dir)
		return err
	}

	if err := os.Chdir(dir); err != nil {
		slog.Debug("chdir error")
		return err
	}

	if _, err := git.Config(config.Entry("user.email", fmt.Sprintf("%s@localhost", username))); err != nil {
		slog.Debug("git config error", "user.email", fmt.Sprintf("%s@localhost", username))
		return err
	}

	if _, err := git.Config(config.Entry("user.name", username)); err != nil {
		slog.Debug("git config error", "user.name", username)
		return err
	}

	if _, err := git.Checkout(checkout.Branch(branchName)); err != nil {
		slog.Debug("git checkout error", "branch", branchName)
		return err
	}

	if _, err := git.Merge(merge.Commits(master), merge.M("update from master")); err != nil {
		slog.Debug("git merge error")
		if _, err := git.Merge(merge.NoFf, merge.Commits(master), merge.M("update from master")); err != nil {
			slog.Debug("git merge --noff error")
			return err
		}
	}

	if _, err := git.Push(push.Remote(defaultRemote), push.RefSpec(branchName)); err != nil {
		slog.Debug("git push error")
		return err
	}

	return nil
}
