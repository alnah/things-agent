package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open Things",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				app := scriptAppController{runner: cfg.runner}
				if err := app.Activate(ctx, cfg.bundleID); err != nil {
					return err
				}
				if err := waitForAppState(ctx, app, cfg.bundleID, true, restoreLaunchTimeout, restorePollInterval, time.Sleep); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "ok")
				return nil
			})
		},
	}
}

func newCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Close Things",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
				app := scriptAppController{runner: cfg.runner}
				if err := app.Quit(ctx, cfg.bundleID); err != nil {
					return err
				}
				if err := waitForAppState(ctx, app, cfg.bundleID, false, restoreStopTimeout, restorePollInterval, time.Sleep); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "ok")
				return nil
			})
		},
	}
}

func waitForAppState(ctx context.Context, app appController, bundleID string, wantRunning bool, timeout, poll time.Duration, sleep func(time.Duration)) error {
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
