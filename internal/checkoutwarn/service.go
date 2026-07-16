package checkoutwarn

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

const confirmationTimeout = 20 * time.Second

type Status string

const (
	StatusPending       Status = "pending"
	StatusRemotePresent Status = "remote_present"
	StatusNotMerged     Status = "not_merged"
	StatusConfirmed     Status = "confirmed"
)

type Confirmation struct {
	PullRequestNumber int       `json:"pull_request_number"`
	PullRequestURL    string    `json:"pull_request_url,omitempty"`
	Branch            string    `json:"branch"`
	Base              string    `json:"base"`
	HeadOID           string    `json:"head_oid"`
	Status            Status    `json:"status"`
	MergedAt          time.Time `json:"merged_at,omitempty"`
	CheckedAt         time.Time `json:"checked_at,omitempty"`
	RetryAfter        time.Time `json:"retry_after,omitempty"`
}

type Confirmer interface {
	Confirm(context.Context, config.Repository, Confirmation) (Confirmation, error)
}

type Service struct {
	GitRunner    command.Runner
	GitHubRunner command.Runner
	Now          func() time.Time
}

type pullRequestDetail struct {
	Number      int        `json:"number"`
	State       string     `json:"state"`
	MergedAt    *time.Time `json:"mergedAt"`
	HeadRefName string     `json:"headRefName"`
	HeadRefOID  string     `json:"headRefOid"`
	BaseRefName string     `json:"baseRefName"`
	URL         string     `json:"url"`
}

func (s Service) Confirm(ctx context.Context, repository config.Repository, confirmation Confirmation) (Confirmation, error) {
	if s.GitRunner == nil || s.GitHubRunner == nil {
		return confirmation, fmt.Errorf("merged-checkout confirmation is unavailable")
	}
	if confirmation.PullRequestNumber <= 0 || confirmation.Branch == "" || confirmation.Base == "" {
		return confirmation, fmt.Errorf("merged-checkout confirmation is incomplete")
	}
	commandContext, cancel := context.WithTimeout(ctx, confirmationTimeout)
	defer cancel()
	remoteRef := "refs/heads/" + confirmation.Branch
	remote, err := s.GitRunner.Run(
		commandContext, repository.Path, "git", "ls-remote", "--heads", repository.Remote, remoteRef,
	)
	if err != nil {
		return confirmation, fmt.Errorf("inspect remote branch %s: %w", confirmation.Branch, err)
	}
	confirmation.CheckedAt = s.now()
	if strings.TrimSpace(string(remote)) != "" {
		confirmation.Status = StatusRemotePresent
		return confirmation, nil
	}

	fields := "number,state,mergedAt,headRefName,headRefOid,baseRefName,url"
	output, err := s.GitHubRunner.Run(
		commandContext, repository.Path, "gh", "pr", "view", strconv.Itoa(confirmation.PullRequestNumber),
		"--repo", repository.GitHub, "--json", fields,
	)
	if err != nil {
		return confirmation, fmt.Errorf("confirm pull request #%d: %w", confirmation.PullRequestNumber, err)
	}
	var pullRequest pullRequestDetail
	if err := json.Unmarshal(output, &pullRequest); err != nil {
		return confirmation, fmt.Errorf("decode pull request #%d: %w", confirmation.PullRequestNumber, err)
	}
	if pullRequest.Number != confirmation.PullRequestNumber ||
		pullRequest.HeadRefName != confirmation.Branch ||
		pullRequest.BaseRefName != confirmation.Base ||
		strings.ToUpper(pullRequest.State) != "MERGED" || pullRequest.MergedAt == nil {
		confirmation.Status = StatusNotMerged
		return confirmation, nil
	}
	confirmation.Status = StatusConfirmed
	confirmation.PullRequestURL = pullRequest.URL
	confirmation.MergedAt = pullRequest.MergedAt.UTC()
	if pullRequest.HeadRefOID != "" {
		confirmation.HeadOID = pullRequest.HeadRefOID
	}
	return confirmation, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
