package workset

import "github.com/jamesonstone/beacon/internal/model"

type projectFollowing map[string]bool

func followingProjects(projects []model.Project) projectFollowing {
	following := make(projectFollowing, len(projects))
	for _, project := range projects {
		isFollowing := project.FollowState == model.FollowFollowing
		if project.FollowState == "" {
			isFollowing = project.TrackingState != model.TrackingUntracked
		}
		following[project.GitHub] = isFollowing
	}
	return following
}

func (projects projectFollowing) includes(github string) bool {
	if github == "" {
		return true
	}
	following, known := projects[github]
	return !known || following
}
