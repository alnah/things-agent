package app

import (
	"context"

	thingslib "github.com/alnah/things-agent/internal/things"
)

const (
	networkIsolationNone             = thingslib.NetworkIsolationNone
	networkIsolationSandboxNoNetwork = thingslib.NetworkIsolationSandboxNoNetwork
)

type offlineAppLaunchFunc func(context.Context, string) error

var newOfflineAppLaunch = func(mode string) (offlineAppLaunchFunc, error) {
	launch, err := thingslib.NewOfflineAppLaunch(mode)
	if err != nil {
		return nil, err
	}
	if launch == nil {
		return nil, nil
	}
	return func(ctx context.Context, bundleID string) error {
		return launch(ctx, bundleID)
	}, nil
}
