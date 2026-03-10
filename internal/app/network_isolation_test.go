package app

import (
	"context"
	"errors"
	"testing"

	thingslib "github.com/alnah/things-agent/internal/things"
)

func TestNewOfflineAppLaunchWrapper(t *testing.T) {
	orig := thingsNewOfflineAppLaunch
	t.Cleanup(func() {
		thingsNewOfflineAppLaunch = orig
	})

	thingsNewOfflineAppLaunch = func(string) (thingslib.OfflineAppLaunchFunc, error) {
		return nil, nil
	}
	launch, err := newOfflineAppLaunch(networkIsolationNone)
	if err != nil || launch != nil {
		t.Fatalf("expected none network isolation to return nil launch, got launch=%v err=%v", launch, err)
	}

	called := false
	thingsNewOfflineAppLaunch = func(string) (thingslib.OfflineAppLaunchFunc, error) {
		return func(context.Context, string) error {
			called = true
			return nil
		}, nil
	}
	launch, err = newOfflineAppLaunch(networkIsolationSandboxNoNetwork)
	if err != nil || launch == nil {
		t.Fatalf("expected sandbox network isolation launcher, got launch=%v err=%v", launch, err)
	}
	if err := launch(context.Background(), defaultBundleID); err != nil {
		t.Fatalf("wrapped launch failed: %v", err)
	}
	if !called {
		t.Fatal("expected wrapped launch function to be invoked")
	}

	thingsNewOfflineAppLaunch = func(string) (thingslib.OfflineAppLaunchFunc, error) {
		return nil, errors.New("boom")
	}
	if _, err := newOfflineAppLaunch("bogus"); err == nil {
		t.Fatal("expected unsupported network isolation mode error")
	}
}
