package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

const (
	ProbeFormatRepository = "repository-v1"
	ProbeFormatBatch      = "batch-v1"
)

type ProbeResult struct {
	Combined string
	Local    string
	Remote   string
	Format   string
}

type RemoteCollector interface {
	Collect(context.Context, []config.Repository, string, string, int) model.RemoteCollection
}

type Prober struct {
	Runner command.Runner
	Remote RemoteCollector
}

func (p Prober) Probe(ctx context.Context, repository config.Repository, author string) (ProbeResult, error) {
	local, err := p.probeLocal(ctx, repository)
	if err != nil {
		return ProbeResult{}, err
	}

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
	return ProbeResult{Combined: digest([]byte(local), []byte(remote)), Local: local, Remote: remote, Format: ProbeFormatRepository}, nil
}

// ProbeMany collects GitHub evidence once for the complete due-project set.
// Local fingerprints remain repository-specific and never fetch remotes.
func (p Prober) ProbeMany(
	ctx context.Context,
	repositories []config.Repository,
	scope, author string,
	maxParallel int,
) (map[string]ProbeResult, map[string]error) {
	results := make(map[string]ProbeResult, len(repositories))
	failures := make(map[string]error)
	if len(repositories) == 0 {
		return results, failures
	}
	if p.Remote == nil {
		for _, repository := range repositories {
			result, err := p.Probe(ctx, repository, author)
			if err != nil {
				failures[repository.GitHub] = err
				continue
			}
			results[repository.GitHub] = result
		}
		return results, failures
	}

	remote := p.Remote.Collect(ctx, repositories, scope, author, maxParallel)
	type localResult struct {
		repository  config.Repository
		fingerprint string
		err         error
	}
	localResults := make(chan localResult, len(repositories))
	semaphore := make(chan struct{}, max(1, maxParallel))
	var waitGroup sync.WaitGroup
	for _, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			fingerprint, err := p.probeLocal(ctx, repository)
			localResults <- localResult{repository: repository, fingerprint: fingerprint, err: err}
		}()
	}
	waitGroup.Wait()
	close(localResults)

	for value := range localResults {
		repository := value.repository
		if value.err != nil {
			failures[repository.GitHub] = value.err
			continue
		}
		if len(remote.Errors) > 0 {
			failures[repository.GitHub] = fmt.Errorf("batch GitHub probe: %s", remote.Errors[0].Message)
			continue
		}
		evidence := remote.Repositories[repository.GitHub]
		if len(evidence.Errors) > 0 {
			failures[repository.GitHub] = fmt.Errorf("batch GitHub probe for %s: %s", repository.Name, evidence.Errors[0].Message)
			continue
		}
		encoded, err := json.Marshal(struct {
			PullRequests []model.PullRequest `json:"pull_requests"`
			Issues       []model.Issue       `json:"issues"`
		}{PullRequests: evidence.PullRequests, Issues: evidence.Issues})
		if err != nil {
			failures[repository.GitHub] = fmt.Errorf("encode batch GitHub probe for %s: %w", repository.Name, err)
			continue
		}
		remoteFingerprint := digest(encoded)
		results[repository.GitHub] = ProbeResult{
			Combined: digest([]byte(value.fingerprint), []byte(remoteFingerprint)),
			Local:    value.fingerprint, Remote: remoteFingerprint, Format: ProbeFormatBatch,
		}
	}
	return results, failures
}

func (p Prober) probeLocal(ctx context.Context, repository config.Repository) (string, error) {
	localContext, cancelLocal := context.WithTimeout(ctx, 5*time.Second)
	defer cancelLocal()
	head, err := p.Runner.Run(localContext, repository.Path, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("probe local HEAD for %s: %w", repository.Name, err)
	}
	status, err := p.Runner.Run(localContext, repository.Path, "git", "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return "", fmt.Errorf("probe local status for %s: %w", repository.Name, err)
	}
	branches, err := p.Runner.Run(localContext, repository.Path, "git", "for-each-ref", "--format=%(refname:short)%00%(objectname)", "refs/heads")
	if err != nil {
		return "", fmt.Errorf("probe local branches for %s: %w", repository.Name, err)
	}
	return digest(head, status, branches), nil
}

func digest(parts ...[]byte) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:", len(part))
		hash.Write(part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
