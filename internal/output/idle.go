package output

import "github.com/jamesonstone/beacon/internal/model"

func idleFollowingInventory(snapshot model.Snapshot) ([]string, int) {
	byID := make(map[string]model.Lane, len(snapshot.Lanes))
	for _, lane := range snapshot.Lanes {
		byID[lane.ID] = lane
	}

	activeProjects := make(map[string]struct{})
	for _, group := range [][]string{snapshot.Groups.Ready, snapshot.Groups.Action, snapshot.Groups.Waiting} {
		for _, id := range group {
			if lane, ok := byID[id]; ok {
				activeProjects[projectKey(lane)] = struct{}{}
			}
		}
	}

	quietProjects := make(map[string]struct{})
	quietIDs := make([]string, 0, len(snapshot.Groups.Idle))
	for _, id := range snapshot.Groups.Idle {
		lane, ok := byID[id]
		if !ok {
			continue
		}
		key := projectKey(lane)
		if _, active := activeProjects[key]; active {
			continue
		}
		quietIDs = append(quietIDs, id)
		quietProjects[key] = struct{}{}
	}
	return quietIDs, len(quietProjects)
}

func projectKey(lane model.Lane) string {
	if lane.GitHub != "" {
		return lane.GitHub
	}
	return lane.Repository
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
