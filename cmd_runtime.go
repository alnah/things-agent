package main

import (
	"context"

	"github.com/spf13/cobra"
)

type commandRuntimeFunc func(context.Context, *runtimeConfig) error

func withRuntimeConfig(cmd *cobra.Command, run commandRuntimeFunc) error {
	ctx := cmd.Context()
	cfg, err := resolveRuntimeConfig(ctx)
	if err != nil {
		return err
	}
	return run(ctx, cfg)
}

func withWriteBackup(cmd *cobra.Command, destructive bool, run commandRuntimeFunc) error {
	return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
		var err error
		if destructive {
			err = backupIfDestructive(ctx, cfg)
		} else {
			err = backupIfNeeded(ctx, cfg)
		}
		if err != nil {
			return err
		}
		return run(ctx, cfg)
	})
}
