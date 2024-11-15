package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/url"
	"os"

	"github.com/ldez/go-git-cmd-wrapper/v2/clone"
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
			slog.Error("url.Parse", "err", err)
			return err
		}
		parsedUrl.User = url.UserPassword(username, password)
		repoUrl = parsedUrl.String()
	}

	hasher := md5.New()
	hasher.Write([]byte(repoUrl))
	dir := hex.EncodeToString(hasher.Sum([]byte(branchName)))

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return errors.New("directory exists")
	}

	defer os.RemoveAll(dir)

	_, err := git.Clone(clone.Repository(repoUrl), clone.Branch(branchName), clone.Directory(dir))
	if err != nil {
		slog.Error("git.Clone", "err", err)
		return err
	}

	_, err = git.Merge(merge.NoFf, merge.Commits(master))
	if err != nil {
		slog.Error("git.Merge", "err", err)
		return err
	}

	_, err = git.Push(push.Remote(defaultRemote))
	if err != nil {
		slog.Error("git.Push", "err", err)
		return err
	}

	return nil
}
