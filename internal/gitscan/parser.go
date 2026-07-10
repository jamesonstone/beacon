package gitscan

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func parseWorktrees(output []byte) []worktreeRecord {
	var records []worktreeRecord
	var current worktreeRecord
	flush := func() {
		if current.Path != "" {
			records = append(records, current)
			current = worktreeRecord{}
		}
	}
	for _, item := range bytes.Split(output, []byte{0}) {
		line := string(item)
		if line == "" {
			flush()
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			if current.Path != "" {
				flush()
			}
			current.Path = value
		case "HEAD":
			current.HeadOID = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "detached":
			current.Detached = true
		case "locked":
			current.Locked = true
		case "prunable":
			current.Prunable = true
		}
	}
	flush()
	return records
}

func parseStatus(output []byte) (statusRecord, error) {
	var status statusRecord
	for _, item := range bytes.Split(output, []byte{0}) {
		if len(item) == 0 {
			continue
		}
		line := string(item)
		if strings.HasPrefix(line, "# ") {
			key, value, _ := strings.Cut(strings.TrimPrefix(line, "# "), " ")
			switch key {
			case "branch.oid":
				status.HeadOID = value
			case "branch.head":
				status.Head = value
			case "branch.upstream":
				status.Upstream = value
			case "branch.ab":
				fields := strings.Fields(value)
				if len(fields) == 2 {
					status.Ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
					status.Behind, _ = strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
				}
			}
			continue
		}
		switch line[0] {
		case '1', '2':
			fields := strings.Fields(line)
			if len(fields) < 2 || len(fields[1]) != 2 {
				return statusRecord{}, fmt.Errorf("invalid porcelain status record: %q", line)
			}
			xy := fields[1]
			if xy[0] != '.' {
				status.Staged++
			}
			if xy[1] != '.' {
				status.Unstaged++
			}
		case 'u':
			status.Conflicted++
		case '?':
			status.Untracked++
		}
	}
	return status, nil
}

func parseCounts(output []byte) (int, int, error) {
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("invalid rev-list counts: %q", strings.TrimSpace(string(output)))
	}
	left, leftErr := strconv.Atoi(fields[0])
	right, rightErr := strconv.Atoi(fields[1])
	if leftErr != nil || rightErr != nil {
		return 0, 0, fmt.Errorf("invalid rev-list counts: %q", strings.TrimSpace(string(output)))
	}
	return left, right, nil
}

func shortOID(oid string) string {
	if len(oid) > 8 {
		return oid[:8]
	}
	return oid
}
