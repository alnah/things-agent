package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newTagsCmd() *cobra.Command {
	return commandlib.NewTagsRootCmd(
		newTagsListCmd(),
		newTagsSearchCmd(),
		newTagsAddCmd(),
		newTagsEditCmd(),
		newTagsDeleteCmd(),
	)
}

func newTagsListCmd() *cobra.Command {
	return commandlib.NewTagsListCmd(func(cmd *cobra.Command, args []string, query string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptListTags(cfg.bundleID, query))
		})
	})
}

func newTagsSearchCmd() *cobra.Command {
	return commandlib.NewTagsSearchCmd(func(cmd *cobra.Command, args []string, query string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptListTags(cfg.bundleID, query))
		})
	})
}

func newTagsAddCmd() *cobra.Command {
	return commandlib.NewTagsAddCmd(func(cmd *cobra.Command, args []string, name, parent string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAddTag(cfg.bundleID, name, parent))
		})
	})
}

func newTagsEditCmd() *cobra.Command {
	return commandlib.NewTagsEditCmd(func(cmd *cobra.Command, args []string, name, newName, parent string, parentChanged bool) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptEditTag(cfg.bundleID, name, newName, parent, parentChanged))
		})
	})
}

func newTagsDeleteCmd() *cobra.Command {
	return commandlib.NewTagsDeleteCmd(func(cmd *cobra.Command, args []string, name string) error {
		return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptDeleteTag(cfg.bundleID, name))
		})
	})
}
