package command

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

type AppController interface {
	IsRunning(ctx context.Context, bundleID string) (bool, error)
	Quit(ctx context.Context, bundleID string) error
	Activate(ctx context.Context, bundleID string) error
}

func NewOpenCmd(runE func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open Things",
		RunE:  runE,
	}
}

func NewCloseCmd(runE func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Close Things",
		RunE:  runE,
	}
}

func WaitForAppState(ctx context.Context, app AppController, bundleID string, wantRunning bool, timeout, poll time.Duration, sleep func(time.Duration)) error {
	if timeout <= 0 {
		timeout = time.Second
	}
	if poll <= 0 {
		poll = 10 * time.Millisecond
	}
	if sleep == nil {
		sleep = time.Sleep
	}

	deadline := time.Now().Add(timeout)
	for {
		running, err := app.IsRunning(ctx, bundleID)
		if err != nil {
			return err
		}
		if running == wantRunning {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if wantRunning {
				return fmt.Errorf("Things did not open within %s", timeout)
			}
			return fmt.Errorf("Things did not close within %s", timeout)
		}
		sleep(poll)
	}
}
