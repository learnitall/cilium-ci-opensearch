package github

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v60/github"
)

func GetCommitBySHA(
	ctx context.Context,
	logger *slog.Logger,
	repoOwner string,
	repoName string,
	client *github.Client,
	sha string,
) (*Commit, error) {
	c, _, err := client.Git.GetCommit(
		ctx, repoOwner, repoName, sha,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"unable to get commit data for sha %s: %v", sha, err,
		)
	}

	if c == nil {
		return nil, fmt.Errorf(
			"unable to get commit data for sha %s: unexpected nil", sha,
		)
	}

	author := User{}

	a := c.GetAuthor()
	if a != nil {
		author.Email = a.GetEmail()
		author.Login = a.GetLogin()
		author.Name = a.GetName()
	}

	commit := &Commit{
		Message: c.GetMessage(),
		Author:  author,
		URL:     c.GetURL(),
	}

	logger.Debug("Got commit for sha", "sha", sha, "commit", commit)

	return commit, nil
}
