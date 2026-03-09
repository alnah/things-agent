package main

import (
	"context"
	"fmt"
	"time"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return commandlib.NewOpenCmd(func(cmd *cobra.Command, args []string) error {
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
	})
}

func newCloseCmd() *cobra.Command {
	return commandlib.NewCloseCmd(func(cmd *cobra.Command, args []string) error {
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
	})
}

func waitForAppState(ctx context.Context, app appController, bundleID string, wantRunning bool, timeout, poll time.Duration, sleep func(time.Duration)) error {
	return commandlib.WaitForAppState(ctx, app, bundleID, wantRunning, timeout, poll, sleep)
}
