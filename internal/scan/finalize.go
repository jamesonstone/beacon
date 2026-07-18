package scan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/progress"
)

func Finalize(snapshot *model.Snapshot) {
	snapshot.Groups = model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}, Untracked: []string{}}
	snapshot.WorkingSet = model.WorkingSet{Order: []string{}, Active: []string{}, Waiting: []string{}, Recent: []string{}, Parked: []string{}}
	snapshot.Summary = model.Summary{}
	normalizeRichEvidence(snapshot.Lanes)
	orderLanes(snapshot.Lanes)
	orderProjectLanes(snapshot.Projects, snapshot.Lanes)
	group(snapshot)
}

func normalizeRichEvidence(lanes []model.Lane) {
	for laneIndex := range lanes {
		pullRequest := lanes[laneIndex].PullRequest
		if pullRequest == nil {
			continue
		}
		if pullRequest.Feedback.Threads == nil {
			pullRequest.Feedback.Threads = []model.ReviewThread{}
		}
		for threadIndex := range pullRequest.Feedback.Threads {
			if pullRequest.Feedback.Threads[threadIndex].Comments == nil {
				pullRequest.Feedback.Threads[threadIndex].Comments = []model.ReviewComment{}
			}
		}
	}
}

func correlateProgress(repository config.Repository, result progress.Result) (*model.Progress, map[int]model.Progress) {
	byIssue := make(map[int]model.Progress)
	prefix := fmt.Sprintf("https://github.com/%s/issues/", repository.GitHub)
	for _, feature := range result.Features {
		for _, issueURL := range feature.IssueURLs {
			if !strings.HasPrefix(issueURL, prefix) {
				continue
			}
			number, err := strconv.Atoi(strings.TrimPrefix(issueURL, prefix))
			if err == nil && number > 0 {
				// Features are ordered by numeric ID, so the newest exact
				// reference deterministically wins shared delivery issues.
				byIssue[number] = progressModel(feature)
			}
		}
	}
	if result.Selected == nil {
		return nil, byIssue
	}
	selected := progressModel(*result.Selected)
	return &selected, byIssue
}

func progressModel(feature progress.Feature) model.Progress {
	summary := feature.Summary
	if summary == "" {
		summary = feature.OpenItems
	}
	if summary == "" {
		summary = feature.Intent
	}
	return model.Progress{
		Source: "kit", FeatureID: feature.ID, Feature: feature.Slug,
		Phase: feature.Phase, Summary: summary, Path: feature.SpecPath,
	}
}

func orderLanes(lanes []model.Lane) {
	sort.SliceStable(lanes, func(left, right int) bool {
		leftLane, rightLane := lanes[left], lanes[right]
		if leftLane.ReviewReady != rightLane.ReviewReady {
			return leftLane.ReviewReady
		}
		if !leftLane.ReviewReady {
			leftPriority, rightPriority := actionPriority(leftLane.NextAction), actionPriority(rightLane.NextAction)
			if leftPriority != rightPriority {
				return leftPriority < rightPriority
			}
		}
		if !leftLane.UpdatedAt.Equal(rightLane.UpdatedAt) {
			return leftLane.UpdatedAt.Before(rightLane.UpdatedAt)
		}
		if leftLane.Repository != rightLane.Repository {
			return leftLane.Repository < rightLane.Repository
		}
		if leftLane.Branch != rightLane.Branch {
			return leftLane.Branch < rightLane.Branch
		}
		return leftLane.ID < rightLane.ID
	})
}

func actionPriority(action model.Action) int {
	switch action {
	case model.ActionResolveConflict:
		return 1
	case model.ActionFixCI:
		return 2
	case model.ActionAddressReview:
		return 3
	case model.ActionInspectLocal:
		return 4
	case model.ActionPushBranch:
		return 5
	case model.ActionRefreshState:
		return 6
	case model.ActionCreatePR:
		return 7
	case model.ActionMarkReady:
		return 8
	case model.ActionWaitForCI:
		return 9
	case model.ActionMergePR:
		return 10
	case model.ActionManualTestMerge:
		return 11
	case model.ActionReviewPR:
		return 12
	case model.ActionResumeOrClose:
		return 13
	case model.ActionStartIssue:
		return 14
	default:
		return 15
	}
}

func group(snapshot *model.Snapshot) {
	snapshot.Summary.Projects = len(snapshot.Projects)
	for _, lane := range snapshot.Lanes {
		snapshot.Summary.Total++
		if lane.Issue != nil {
			snapshot.Summary.OpenIssues++
		}
		if lane.PullRequest != nil {
			snapshot.Summary.UnresolvedFeedback += lane.PullRequest.Feedback.UnresolvedThreads
		}
		switch {
		case lane.ReviewReady:
			snapshot.Groups.Ready = append(snapshot.Groups.Ready, lane.ID)
			snapshot.Summary.ReviewReady++
		case lane.NextAction != model.ActionNone:
			snapshot.Groups.Action = append(snapshot.Groups.Action, lane.ID)
			snapshot.Summary.NeedsAction++
		case lane.PullRequest != nil || lane.Issue != nil || lane.Branch != lane.Base:
			snapshot.Groups.Waiting = append(snapshot.Groups.Waiting, lane.ID)
			snapshot.Summary.Waiting++
		default:
			snapshot.Groups.Idle = append(snapshot.Groups.Idle, lane.ID)
			snapshot.Summary.Idle++
		}
	}
	snapshot.Summary.Errors = len(snapshot.Errors)
	snapshot.Summary.Warnings = len(snapshot.Warnings)
}
