package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeAppController struct {
	running []bool
	err     error
}

func (f *fakeAppController) IsRunning(_ context.Context, _ string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if len(f.running) == 0 {
		return false, nil
	}
	state := f.running[0]
	if len(f.running) > 1 {
		f.running = f.running[1:]
	}
	return state, nil
}

func (f *fakeAppController) Quit(_ context.Context, _ string) error {
	return nil
}

func (f *fakeAppController) Activate(_ context.Context, _ string) error {
	return nil
}

func TestWaitForAppState(t *testing.T) {
	t.Run("waits for open", func(t *testing.T) {
		app := &fakeAppController{running: []bool{false, true}}
		if err := WaitForAppState(context.Background(), app, "bundle.id", true, 100*time.Millisecond, time.Millisecond, func(time.Duration) {}); err != nil {
			t.Fatalf("WaitForAppState open failed: %v", err)
		}
	})

	t.Run("waits for close", func(t *testing.T) {
		app := &fakeAppController{running: []bool{true, false}}
		if err := WaitForAppState(context.Background(), app, "bundle.id", false, 100*time.Millisecond, time.Millisecond, func(time.Duration) {}); err != nil {
			t.Fatalf("WaitForAppState close failed: %v", err)
		}
	})

	t.Run("returns controller errors", func(t *testing.T) {
		app := &fakeAppController{err: errors.New("boom")}
		if err := WaitForAppState(context.Background(), app, "bundle.id", true, 100*time.Millisecond, time.Millisecond, func(time.Duration) {}); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("expected controller error, got %v", err)
		}
	})

	t.Run("times out waiting for open", func(t *testing.T) {
		app := &fakeAppController{running: []bool{false, false, false}}
		err := WaitForAppState(context.Background(), app, "bundle.id", true, time.Nanosecond, time.Millisecond, func(time.Duration) {})
		if err == nil || !strings.Contains(err.Error(), "did not open") {
			t.Fatalf("expected open timeout, got %v", err)
		}
	})

	t.Run("times out waiting for close", func(t *testing.T) {
		app := &fakeAppController{running: []bool{true, true, true}}
		err := WaitForAppState(context.Background(), app, "bundle.id", false, time.Nanosecond, time.Millisecond, func(time.Duration) {})
		if err == nil || !strings.Contains(err.Error(), "did not close") {
			t.Fatalf("expected close timeout, got %v", err)
		}
	})

	t.Run("returns context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		app := &fakeAppController{running: []bool{false}}
		err := WaitForAppState(ctx, app, "bundle.id", true, 100*time.Millisecond, time.Millisecond, func(time.Duration) {})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	})

	t.Run("uses default timeout and poll when unset", func(t *testing.T) {
		app := &fakeAppController{running: []bool{false, true}}
		if err := WaitForAppState(context.Background(), app, "bundle.id", true, 0, 0, nil); err != nil {
			t.Fatalf("expected default timeout/poll path to succeed, got %v", err)
		}
	})
}
