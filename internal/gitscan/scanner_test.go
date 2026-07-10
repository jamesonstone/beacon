package gitscan

import (
	"context"
	"testing"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
)

func TestParseWorktrees(t *testing.T) {
	input := []byte("worktree /repo\x00HEAD abcdef\x00branch refs/heads/main\x00\x00worktree /repo-feature\x00HEAD 123456\x00branch refs/heads/feature/name\x00locked reason\x00\x00")
	records := parseWorktrees(input)
	if len(records) != 2 {
		t.Fatalf("records = %#v", records)
	}
	if records[1].Branch != "feature/name" || !records[1].Locked {
		t.Fatalf("second record = %#v", records[1])
	}
}

func TestParseStatus(t *testing.T) {
	input := []byte("# branch.oid abcdef\x00# branch.head feature\x00# branch.upstream origin/feature\x00# branch.ab +2 -1\x001 M. N... 100644 100644 100644 abc abc staged.txt\x001 .M N... 100644 100644 100644 abc abc modified.txt\x002 MM N... 100644 100644 100644 abc abc R100 renamed.txt\x00old name.txt\x00u UU N... 100644 100644 100644 100644 abc abc abc conflict.txt\x00? untracked.txt\x00")
	status, err := parseStatus(input)
	if err != nil {
		t.Fatal(err)
	}
	if status.Head != "feature" || status.Upstream != "origin/feature" || status.Ahead != 2 || status.Behind != 1 {
		t.Fatalf("branch status = %#v", status)
	}
	if status.Staged != 2 || status.Unstaged != 2 || status.Untracked != 1 || status.Conflicted != 1 {
		t.Fatalf("file counts = %#v", status)
	}
}

func TestPublication(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		detached bool
		worktree model.Worktree
		expected model.PublicationState
	}{
		{"base", "main", false, model.Worktree{}, model.PublicationBase},
		{"no upstream", "feature", false, model.Worktree{AheadBase: 1}, model.PublicationNoUpstream},
		{"unpushed", "feature", false, model.Worktree{Upstream: "origin/feature", Ahead: 1}, model.PublicationUnpushed},
		{"behind", "feature", false, model.Worktree{Upstream: "origin/feature", Behind: 1}, model.PublicationBehind},
		{"diverged", "feature", false, model.Worktree{Upstream: "origin/feature", Ahead: 1, Behind: 1}, model.PublicationDiverged},
		{"published", "feature", false, model.Worktree{Upstream: "origin/feature"}, model.PublicationPublished},
		{"detached", "feature", true, model.Worktree{}, model.PublicationUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := publication("main", test.branch, test.detached, test.worktree); actual != test.expected {
				t.Fatalf("publication = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestParseCounts(t *testing.T) {
	left, right, err := parseCounts([]byte("3\t7\n"))
	if err != nil || left != 3 || right != 7 {
		t.Fatalf("counts = %d, %d, %v", left, right, err)
	}
}

func TestPrunableWorktreeIsWarningNotError(t *testing.T) {
	lane, errors, warnings := (Scanner{}).scanWorktree(context.Background(), config.Repository{GitHub: "owner/repo"}, worktreeRecord{
		Path: "/missing/worktree", HeadOID: "abcdef", Branch: "feature", Prunable: true,
	})
	if !lane.Worktree.Prunable || len(errors) != 0 || len(warnings) != 1 || warnings[0].Stage != "worktree" {
		t.Fatalf("lane=%#v errors=%#v warnings=%#v", lane, errors, warnings)
	}
}
