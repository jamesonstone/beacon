package githubapi

import "context"

type freshEvidenceKey struct{}

// WithFreshEvidence bypasses a valid response-cache hit for an explicit user
// refresh. The rate-budget reserve still applies and may return stale evidence
// instead of spending protected GitHub capacity.
func WithFreshEvidence(ctx context.Context) context.Context {
	return context.WithValue(ctx, freshEvidenceKey{}, true)
}

func freshEvidenceRequested(ctx context.Context) bool {
	value, _ := ctx.Value(freshEvidenceKey{}).(bool)
	return value
}
