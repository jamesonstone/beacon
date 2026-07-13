package githubscan

import "context"

type includeInactiveKey struct{}
type inactiveRepositoriesKey struct{}

// WithInactivePullRequests marks an explicit diagnostic or lane refresh. The
// default background path only enriches recently active authored pull requests.
func WithInactivePullRequests(ctx context.Context) context.Context {
	return context.WithValue(ctx, includeInactiveKey{}, true)
}

func IncludeInactivePullRequests(ctx context.Context) bool {
	value, _ := ctx.Value(includeInactiveKey{}).(bool)
	return value
}

// WithInactivePullRequestRepositories limits inactive pull-request enrichment
// to the supplied followed repositories. Global PR discovery still runs once
// for the full batch, but old PRs outside this set remain cached/quiet.
func WithInactivePullRequestRepositories(ctx context.Context, repositories []string) context.Context {
	values := make(map[string]struct{}, len(repositories))
	for _, repository := range repositories {
		values[repository] = struct{}{}
	}
	return context.WithValue(ctx, inactiveRepositoriesKey{}, values)
}

func IncludeInactivePullRequestsFor(ctx context.Context, repository string) bool {
	if IncludeInactivePullRequests(ctx) {
		return true
	}
	values, _ := ctx.Value(inactiveRepositoriesKey{}).(map[string]struct{})
	_, found := values[repository]
	return found
}
