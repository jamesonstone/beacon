package checkoutwarn

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/config"
)

func TestConfirmChecksRemoteBeforeExactMergedPullRequest(t *testing.T) {
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	git := &confirmationRunner{outputs: [][]byte{nil}}
	github := &confirmationRunner{outputs: [][]byte{[]byte(`{
        "number": 32,
        "state": "MERGED",
        "mergedAt": "2026-07-15T15:00:00Z",
        "headRefName": "GH-31",
        "headRefOid": "merged-head",
        "baseRefName": "main",
        "url": "https://github.com/owner/repo/pull/32"
    }`)}}
	service := Service{GitRunner: git, GitHubRunner: github, Now: func() time.Time { return now }}

	confirmation, err := service.Confirm(context.Background(), testRepository(), testConfirmation())
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != StatusConfirmed || confirmation.HeadOID != "merged-head" || confirmation.CheckedAt != now {
		t.Fatalf("confirmation = %#v", confirmation)
	}
	if len(git.calls) != 1 || git.calls[0] != "git ls-remote --heads origin refs/heads/GH-31" {
		t.Fatalf("git calls = %#v", git.calls)
	}
	expectedGitHub := "gh pr view 32 --repo owner/repo --json number,state,mergedAt,headRefName,headRefOid,baseRefName,url"
	if len(github.calls) != 1 || github.calls[0] != expectedGitHub {
		t.Fatalf("github calls = %#v", github.calls)
	}
}

func TestConfirmSkipsGitHubWhileRemoteBranchExists(t *testing.T) {
	git := &confirmationRunner{outputs: [][]byte{[]byte("abc\trefs/heads/GH-31\n")}}
	github := &confirmationRunner{}
	confirmation, err := (Service{GitRunner: git, GitHubRunner: github}).Confirm(
		context.Background(), testRepository(), testConfirmation(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if confirmation.Status != StatusRemotePresent || len(github.calls) != 0 {
		t.Fatalf("confirmation=%#v github calls=%#v", confirmation, github.calls)
	}
}

func TestConfirmRefusesClosedOrMismatchedPullRequest(t *testing.T) {
	for name, response := range map[string]string{
		"closed":       `{"number":32,"state":"CLOSED","headRefName":"GH-31","headRefOid":"head","baseRefName":"main","url":"url"}`,
		"wrong branch": `{"number":32,"state":"MERGED","mergedAt":"2026-07-15T15:00:00Z","headRefName":"other","headRefOid":"head","baseRefName":"main","url":"url"}`,
	} {
		t.Run(name, func(t *testing.T) {
			service := Service{
				GitRunner:    &confirmationRunner{outputs: [][]byte{nil}},
				GitHubRunner: &confirmationRunner{outputs: [][]byte{[]byte(response)}},
			}
			confirmation, err := service.Confirm(context.Background(), testRepository(), testConfirmation())
			if err != nil {
				t.Fatal(err)
			}
			if confirmation.Status != StatusNotMerged {
				t.Fatalf("confirmation = %#v", confirmation)
			}
		})
	}
}

func TestConfirmReturnsRemoteAndGitHubFailuresWithoutGuessing(t *testing.T) {
	remoteFailure := Service{
		GitRunner:    &confirmationRunner{errs: []error{errors.New("offline")}},
		GitHubRunner: &confirmationRunner{},
	}
	if _, err := remoteFailure.Confirm(context.Background(), testRepository(), testConfirmation()); err == nil || !strings.Contains(err.Error(), "inspect remote branch") {
		t.Fatalf("remote error = %v", err)
	}

	githubFailure := Service{
		GitRunner:    &confirmationRunner{outputs: [][]byte{nil}},
		GitHubRunner: &confirmationRunner{errs: []error{errors.New("rate limited")}},
	}
	if _, err := githubFailure.Confirm(context.Background(), testRepository(), testConfirmation()); err == nil || !strings.Contains(err.Error(), "confirm pull request") {
		t.Fatalf("github error = %v", err)
	}
}

type confirmationRunner struct {
	calls   []string
	outputs [][]byte
	errs    []error
}

func (r *confirmationRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	index := len(r.calls) - 1
	var output []byte
	if index < len(r.outputs) {
		output = r.outputs[index]
	}
	if index < len(r.errs) {
		return output, r.errs[index]
	}
	return output, nil
}

func testRepository() config.Repository {
	return config.Repository{Name: "repo", Path: "/repo", GitHub: "owner/repo", Base: "main", Remote: "origin"}
}

func testConfirmation() Confirmation {
	return Confirmation{
		PullRequestNumber: 32, PullRequestURL: "https://github.com/owner/repo/pull/32",
		Branch: "GH-31", Base: "main", HeadOID: "head", Status: StatusPending,
	}
}
