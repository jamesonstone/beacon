package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

type ProbeResult struct {
	Combined string
	Local    string
	Remote   string
}

type Prober struct {
	Runner command.Runner
}

func (p Prober) Probe(ctx context.Context, repository config.Repository, author string) (ProbeResult, error) {
	localContext, cancelLocal := context.WithTimeout(ctx, 5*time.Second)
	defer cancelLocal()
	head, err := p.Runner.Run(localContext, repository.Path, "git", "rev-parse", "HEAD")
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe local HEAD for %s: %w", repository.Name, err)
	}
	status, err := p.Runner.Run(localContext, repository.Path, "git", "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe local status for %s: %w", repository.Name, err)
	}
	branches, err := p.Runner.Run(localContext, repository.Path, "git", "for-each-ref", "--format=%(refname:short)%00%(objectname)", "refs/heads")
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe local branches for %s: %w", repository.Name, err)
	}
	local := digest(head, status, branches)

	remoteContext, cancelRemote := context.WithTimeout(ctx, 20*time.Second)
	defer cancelRemote()
	pullRequestArguments := []string{"pr", "list", "--repo", repository.GitHub, "--state", "open", "--limit", "100"}
	if author != "" {
		pullRequestArguments = append(pullRequestArguments, "--author", author)
	}
	pullRequestArguments = append(pullRequestArguments, "--json", "number,headRefOid,isDraft,updatedAt,reviewDecision,statusCheckRollup,mergeStateStatus")
	pullRequests, err := p.Runner.Run(remoteContext, "", "gh", pullRequestArguments...)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe pull requests for %s: %w", repository.Name, err)
	}
	issueArguments := []string{"issue", "list", "--repo", repository.GitHub, "--state", "open", "--limit", "100"}
	if author != "" {
		issueArguments = append(issueArguments, "--assignee", author)
	}
	issueArguments = append(issueArguments, "--json", "number,updatedAt,state")
	issues, err := p.Runner.Run(remoteContext, "", "gh", issueArguments...)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe issues for %s: %w", repository.Name, err)
	}
	remote := digest(pullRequests, issues)
	return ProbeResult{Combined: digest([]byte(local), []byte(remote)), Local: local, Remote: remote}, nil
}

func digest(parts ...[]byte) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:", len(part))
		hash.Write(part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
