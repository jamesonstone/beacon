package githubscan

import "context"

type includeInactiveKey struct{}

// WithInactivePullRequests marks an explicit diagnostic or lane refresh. The
// default background path only enriches recently active authored pull requests.
func WithInactivePullRequests(ctx context.Context) context.Context {
	return context.WithValue(ctx, includeInactiveKey{}, true)
}

func IncludeInactivePullRequests(ctx context.Context) bool {
	value, _ := ctx.Value(includeInactiveKey{}).(bool)
	return value
}
