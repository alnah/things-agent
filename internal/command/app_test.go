package command

import (
	"context"
	"testing"
	"time"
)

type fakeAppController struct {
	running []bool
}

func (f *fakeAppController) IsRunning(_ context.Context, _ string) (bool, error) {
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
}
