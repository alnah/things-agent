package app

import (
	"context"

	thingslib "github.com/alnah/things-agent/internal/things"
)

type scriptRunner interface {
	run(ctx context.Context, script string) (string, error)
}

type runner struct {
	inner *thingslib.Runner
}

var newRuntimeRunner = func(bundleID string) scriptRunner {
	return newRunner(bundleID)
}

func newRunner(bundleID string) *runner {
	return &runner{
		inner: thingslib.NewRunner(bundleID),
	}
}

func (r *runner) run(ctx context.Context, script string) (string, error) {
	return r.inner.Run(ctx, script)
}

func (r *runner) ensureReachable(ctx context.Context) error {
	return r.inner.EnsureReachable(ctx)
}
