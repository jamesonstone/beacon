package workset

import (
	"sort"

	"github.com/jamesonstone/beacon/internal/model"
)

func sortWorking(groups *model.WorkingSet, order []string) {
	ranks := make(map[string]int, len(order))
	for index, id := range order {
		ranks[id] = index
	}
	sortGroup := func(values []string) {
		sort.SliceStable(values, func(i, j int) bool {
			left, leftFound := ranks[values[i]]
			right, rightFound := ranks[values[j]]
			if leftFound && rightFound {
				return left < right
			}
			if leftFound != rightFound {
				return leftFound
			}
			return values[i] < values[j]
		})
	}
	sortGroup(groups.Active)
	sortGroup(groups.Waiting)
	sortGroup(groups.Recent)
	sortGroup(groups.Parked)
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
