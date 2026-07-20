package workset

import (
	"sort"

	"github.com/jamesonstone/beacon/internal/model"
)

func sortWorking(groups *model.WorkingSet, order []string, projects []model.Project, lanes []model.Lane) {
	ranks := make(map[string]int, len(order))
	for index, id := range order {
		ranks[id] = index
	}
	lanesByID := make(map[string]model.Lane, len(lanes))
	for _, lane := range lanes {
		lanesByID[lane.ID] = lane
	}
	projectRanks := make(map[string]int, len(projects))
	for index, project := range projects {
		projectRanks[project.GitHub] = index
	}
	userOrderLess := func(leftID, rightID string) bool {
		left, leftFound := ranks[leftID]
		right, rightFound := ranks[rightID]
		if leftFound && rightFound {
			return left < right
		}
		if leftFound != rightFound {
			return leftFound
		}
		return leftID < rightID
	}
	sortUserOrder := func(values []string) {
		sort.SliceStable(values, func(i, j int) bool {
			return userOrderLess(values[i], values[j])
		})
	}
	sortFollowing := func(values []string) {
		sort.SliceStable(values, func(i, j int) bool {
			left, leftFound := lanesByID[values[i]]
			right, rightFound := lanesByID[values[j]]
			if leftFound != rightFound {
				return leftFound
			}
			if !leftFound {
				return userOrderLess(values[i], values[j])
			}

			leftProject, leftProjectFound := projectRanks[left.GitHub]
			rightProject, rightProjectFound := projectRanks[right.GitHub]
			if leftProjectFound && rightProjectFound && leftProject != rightProject {
				return leftProject < rightProject
			}
			if leftProjectFound != rightProjectFound {
				return leftProjectFound
			}
			if !leftProjectFound && left.GitHub != right.GitHub {
				return left.GitHub < right.GitHub
			}

			leftType, rightType := followingTypePriority(left), followingTypePriority(right)
			if leftType != rightType {
				return leftType < rightType
			}
			return userOrderLess(values[i], values[j])
		})
	}
	sortFollowing(groups.Active)
	sortFollowing(groups.Waiting)
	sortFollowing(groups.Recent)
	sortUserOrder(groups.Parked)
}

func followingTypePriority(lane model.Lane) int {
	if lane.PullRequest != nil {
		return 0
	}
	if lane.Issue != nil {
		return 1
	}
	return 2
}

func normalizeLaneOrder(order []string, entries map[string]Entry, leading []string) []string {
	result := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	appendKnown := func(id string) {
		if _, found := entries[id]; !found {
			return
		}
		if _, found := seen[id]; found {
			return
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	for _, id := range leading {
		appendKnown(id)
	}
	for _, id := range order {
		appendKnown(id)
	}
	missing := make([]string, 0, len(entries)-len(result))
	for id := range entries {
		if _, found := seen[id]; !found {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return append(result, missing...)
}

func visibleLaneOrder(order []string, working model.WorkingSet) []string {
	visible := make(map[string]struct{}, len(working.Active)+len(working.Waiting)+len(working.Recent)+len(working.Parked))
	for _, group := range [][]string{working.Active, working.Waiting, working.Recent, working.Parked} {
		for _, id := range group {
			visible[id] = struct{}{}
		}
	}
	result := make([]string, 0, len(visible))
	for _, id := range order {
		if _, found := visible[id]; found {
			result = append(result, id)
		}
	}
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func entrySlice(values map[string]Entry) []Entry {
	result := make([]Entry, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
