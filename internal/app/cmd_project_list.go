package app

import (
	"context"
	"fmt"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newAddProjectCmd() *cobra.Command {
	return commandlib.NewAddProjectCmd(func(cmd *cobra.Command, args []string, name, notes, areaName string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, areaName, name, notes))
		})
	})
}

func newAddAreaCmd() *cobra.Command {
	return commandlib.NewAddAreaCmd(func(cmd *cobra.Command, args []string, name string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			script := fmt.Sprintf(`tell application id "%s"
  set a to make new area with properties {name:"%s"}
  return id of a
end tell`, cfg.bundleID, escapeApple(name))
			return runResult(ctx, cfg, script)
		})
	})
}

func newEditProjectCmd() *cobra.Command {
	return commandlib.NewEditProjectCmd(func(cmd *cobra.Command, args []string, sourceName, sourceID, newName, notes string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptEditProjectRef(cfg.bundleID, sourceName, sourceID, newName, notes))
		})
	})
}

func newEditAreaCmd() *cobra.Command {
	return commandlib.NewEditAreaCmd(func(cmd *cobra.Command, args []string, sourceName, newName string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			script := fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  set name of l to "%s"
  return "ok"
end tell`, cfg.bundleID, escapeApple(sourceName), escapeApple(newName))
			return runResult(ctx, cfg, script)
		})
	})
}

func newDeleteProjectCmd() *cobra.Command {
	return commandlib.NewDeleteProjectCmd(func(cmd *cobra.Command, args []string, name, id string) error {
		return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptDeleteProjectRef(cfg.bundleID, name, id))
		})
	})
}

func newDeleteAreaCmd() *cobra.Command {
	return newDeleteCmd("list", "delete-area", "Delete an area")
}

func newDeleteCmd(kind, name, short string) *cobra.Command {
	return commandlib.NewDeleteCmd(kind, name, short, func(cmd *cobra.Command, args []string, kind, target string) error {
		return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
			script, err := scriptDelete(cfg.bundleID, kind, target)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		})
	})
}
