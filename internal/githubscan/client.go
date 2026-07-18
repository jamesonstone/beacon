package githubscan

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

const (
	githubTimeout = 20 * time.Second
	searchLimit   = 1000
)

type Client struct {
	Runner command.Runner
	Now    func() time.Time
}

func (c Client) Collect(ctx context.Context, repositories []config.Repository, scope, author string, maxParallel int) model.RemoteCollection {
	collection := model.RemoteCollection{
		Repositories: make(map[string]model.RemoteEvidence, len(repositories)),
		Errors:       []model.ScanError{}, Warnings: []model.ScanError{},
	}
	configured := make(map[string]config.Repository, len(repositories))
	for _, repository := range repositories {
		configured[repository.GitHub] = repository
		collection.Repositories[repository.GitHub] = emptyEvidence()
	}
	if scope == "all" {
		c.collectAll(ctx, repositories, maxParallel, &collection)
		return collection
	}
	c.collectMine(ctx, configured, author, maxParallel, &collection)
	return collection
}

func (c Client) ListOpen(ctx context.Context, repository, author string) ([]model.PullRequest, error) {
	output, err := c.run(ctx, "gh", "pr", "list", "--repo", repository, "--author", author, "--state", "open", "--limit", strconv.Itoa(searchLimit), "--json", pullRequestFields)
	if err != nil {
		return nil, err
	}
	return parsePullRequests(output)
}

func (c Client) collectMine(ctx context.Context, configured map[string]config.Repository, author string, maxParallel int, collection *model.RemoteCollection) {
	prOutput, err := c.run(ctx, "gh", "search", "prs", "--author", author, "--state", "open", "--limit", strconv.Itoa(searchLimit), "--json", "number,updatedAt,repository")
	if err != nil {
		collection.Errors = append(collection.Errors, model.ScanError{Stage: "github-search-prs", Message: err.Error()})
	} else {
		var matches []rawSearchItem
		if err := json.Unmarshal(prOutput, &matches); err != nil {
			collection.Errors = append(collection.Errors, model.ScanError{Stage: "github-search-prs", Message: "decode results: " + err.Error()})
		} else {
			if len(matches) == searchLimit {
				collection.Warnings = append(collection.Warnings, model.ScanError{Stage: "github-search-prs", Message: "result limit reached; pull requests may be truncated"})
			}
			c.enrichMinePullRequests(ctx, configured, matches, maxParallel, collection)
		}
	}

	issueOutput, err := c.run(ctx, "gh", "search", "issues", "--assignee", author, "--state", "open", "--limit", strconv.Itoa(searchLimit), "--json", "number,title,body,url,updatedAt,labels,assignees,repository")
	if err != nil {
		collection.Errors = append(collection.Errors, model.ScanError{Stage: "github-search-issues", Message: err.Error()})
		return
	}
	issues, resultCount, err := parseSearchIssues(issueOutput)
	if err != nil {
		collection.Errors = append(collection.Errors, model.ScanError{Stage: "github-search-issues", Message: err.Error()})
		return
	}
	if resultCount == searchLimit {
		collection.Warnings = append(collection.Warnings, model.ScanError{Stage: "github-search-issues", Message: "result limit reached; issues may be truncated"})
	}
	for repository, values := range issues {
		if _, ok := configured[repository]; !ok {
			continue
		}
		evidence := collection.Repositories[repository]
		evidence.Issues = append(evidence.Issues, values...)
		collection.Repositories[repository] = evidence
	}
}

func (c Client) enrichMinePullRequests(ctx context.Context, configured map[string]config.Repository, matches []rawSearchItem, maxParallel int, collection *model.RemoteCollection) {
	type task struct {
		repository string
		number     int
	}
	var tasks []task
	cutoff := c.now().Add(-6 * time.Hour)
	for _, match := range matches {
		repository := match.Repository.NameWithOwner
		if _, ok := configured[repository]; ok && (IncludeInactivePullRequestsFor(ctx, repository) || match.UpdatedAt.After(cutoff)) {
			tasks = append(tasks, task{repository: repository, number: match.Number})
		}
	}
	semaphore := make(chan struct{}, max(1, maxParallel))
	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	for _, current := range tasks {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			pullRequest, scanErrors, scanWarnings := c.pullRequestDetail(ctx, current.repository, current.number)
			mutex.Lock()
			defer mutex.Unlock()
			evidence := collection.Repositories[current.repository]
			if pullRequest != nil {
				evidence.PullRequests = append(evidence.PullRequests, *pullRequest)
			}
			for _, scanError := range scanErrors {
				scanError.Repository = configured[current.repository].Name
				evidence.Errors = append(evidence.Errors, scanError)
			}
			for _, scanWarning := range scanWarnings {
				scanWarning.Repository = configured[current.repository].Name
				evidence.Warnings = append(evidence.Warnings, scanWarning)
			}
			collection.Repositories[current.repository] = evidence
		}()
	}
	waitGroup.Wait()
	for repository, evidence := range collection.Repositories {
		sortPullRequests(evidence.PullRequests)
		sort.Slice(evidence.Errors, func(i, j int) bool {
			if evidence.Errors[i].Stage != evidence.Errors[j].Stage {
				return evidence.Errors[i].Stage < evidence.Errors[j].Stage
			}
			return evidence.Errors[i].Message < evidence.Errors[j].Message
		})
		sort.Slice(evidence.Warnings, func(i, j int) bool {
			if evidence.Warnings[i].Stage != evidence.Warnings[j].Stage {
				return evidence.Warnings[i].Stage < evidence.Warnings[j].Stage
			}
			return evidence.Warnings[i].Message < evidence.Warnings[j].Message
		})
		collection.Repositories[repository] = evidence
	}
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c Client) collectAll(ctx context.Context, repositories []config.Repository, maxParallel int, collection *model.RemoteCollection) {
	semaphore := make(chan struct{}, max(1, maxParallel))
	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	for _, repository := range repositories {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			evidence := c.collectRepository(ctx, repository, "all", "")
			mutex.Lock()
			collection.Repositories[repository.GitHub] = evidence
			mutex.Unlock()
		}()
	}
	waitGroup.Wait()
}

func (c Client) collectRepository(ctx context.Context, repository config.Repository, scope, author string) model.RemoteEvidence {
	evidence := emptyEvidence()
	prArguments := []string{"pr", "list", "--repo", repository.GitHub, "--state", "open", "--limit", strconv.Itoa(searchLimit), "--json", pullRequestFields}
	if scope != "all" {
		prArguments = append(prArguments, "--author", author)
	}
	prOutput, err := c.run(ctx, "gh", prArguments...)
	if err != nil {
		evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github-prs", Message: err.Error()})
	} else if pullRequests, parseErr := parsePullRequests(prOutput); parseErr != nil {
		evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github-prs", Message: parseErr.Error()})
	} else {
		evidence.PullRequests = pullRequests
		if len(pullRequests) == searchLimit {
			evidence.Warnings = append(evidence.Warnings, model.ScanError{Repository: repository.Name, Stage: "github-prs", Message: "result limit reached; pull requests may be truncated"})
		}
		for index := range evidence.PullRequests {
			threads, count, truncated, threadErr := c.reviewThreadDetails(ctx, repository.GitHub, evidence.PullRequests[index].Number)
			if threadErr != nil {
				evidence.PullRequests[index].ReviewDecision = "UNKNOWN"
				evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github-feedback", Message: threadErr.Error()})
				continue
			}
			evidence.PullRequests[index].Feedback.UnresolvedThreads = count
			evidence.PullRequests[index].Feedback.Threads = threads
			evidence.PullRequests[index].Feedback.ThreadsTruncated = truncated
			if truncated {
				evidence.Warnings = append(evidence.Warnings, model.ScanError{Repository: repository.Name, Stage: "github-feedback", Message: fmt.Sprintf("PR #%d review threads may be truncated", evidence.PullRequests[index].Number)})
			}
		}
	}
	issueArguments := []string{"issue", "list", "--repo", repository.GitHub, "--state", "open", "--limit", strconv.Itoa(searchLimit), "--json", "number,title,body,url,updatedAt,labels,assignees"}
	if scope != "all" {
		issueArguments = append(issueArguments, "--assignee", author)
	}
	issueOutput, err := c.run(ctx, "gh", issueArguments...)
	if err != nil {
		evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github-issues", Message: err.Error()})
	} else if issues, parseErr := parseIssues(issueOutput); parseErr != nil {
		evidence.Errors = append(evidence.Errors, model.ScanError{Repository: repository.Name, Stage: "github-issues", Message: parseErr.Error()})
	} else {
		evidence.Issues = issues
		if len(issues) == searchLimit {
			evidence.Warnings = append(evidence.Warnings, model.ScanError{Repository: repository.Name, Stage: "github-issues", Message: "result limit reached; issues may be truncated"})
		}
	}
	return evidence
}

func (c Client) pullRequestDetail(ctx context.Context, repository string, number int) (*model.PullRequest, []model.ScanError, []model.ScanError) {
	output, err := c.run(ctx, "gh", "pr", "view", strconv.Itoa(number), "--repo", repository, "--json", pullRequestFields)
	if err != nil {
		return nil, []model.ScanError{{Stage: "github-pr", Message: err.Error()}}, nil
	}
	var raw rawPullRequest
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, []model.ScanError{{Stage: "github-pr", Message: "decode detail: " + err.Error()}}, nil
	}
	pullRequest := normalizePullRequest(raw)
	threads, count, truncated, err := c.reviewThreadDetails(ctx, repository, number)
	if err != nil {
		pullRequest.ReviewDecision = "UNKNOWN"
		return &pullRequest, []model.ScanError{{Stage: "github-feedback", Message: err.Error()}}, nil
	}
	pullRequest.Feedback.UnresolvedThreads = count
	pullRequest.Feedback.Threads = threads
	pullRequest.Feedback.ThreadsTruncated = truncated
	if truncated {
		return &pullRequest, nil, []model.ScanError{{Stage: "github-feedback", Message: fmt.Sprintf("PR #%d review threads may be truncated", number)}}
	}
	return &pullRequest, nil, nil
}

func (c Client) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	commandContext, cancel := context.WithTimeout(ctx, githubTimeout)
	defer cancel()
	return c.Runner.Run(commandContext, "", name, args...)
}

func emptyEvidence() model.RemoteEvidence {
	return model.RemoteEvidence{
		PullRequests: []model.PullRequest{}, Issues: []model.Issue{},
		Errors: []model.ScanError{}, Warnings: []model.ScanError{},
	}
}

func sortPullRequests(pullRequests []model.PullRequest) {
	sort.Slice(pullRequests, func(i, j int) bool { return pullRequests[i].Number < pullRequests[j].Number })
}
