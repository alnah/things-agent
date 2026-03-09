package app

import (
	"context"

	commandlib "github.com/alnah/things-agent/internal/command"
	"github.com/spf13/cobra"
)

func newAddChecklistItemCmd() *cobra.Command {
	return commandlib.NewAddChecklistItemCmd(func(cmd *cobra.Command, args []string, taskName, taskID, itemName string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendChecklistByRef(cfg.bundleID, taskName, taskID, []string{itemName}, token))
		})
	})
}

func newListChildTasksCmd() *cobra.Command {
	return commandlib.NewListChildTasksCmd(func(cmd *cobra.Command, args []string, parentName, parentID string) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptListChildTasks(cfg.bundleID, parentName, parentID))
		})
	})
}

func newAddChildTaskCmd() *cobra.Command {
	return commandlib.NewAddChildTaskCmd(func(cmd *cobra.Command, args []string, parentName, parentID, childTaskName, notes string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAddChildTask(cfg.bundleID, parentName, parentID, childTaskName, notes))
		})
	})
}

func newEditChildTaskCmd() *cobra.Command {
	return commandlib.NewEditChildTaskCmd(func(cmd *cobra.Command, args []string, parentName, parentID, childTaskName, childTaskID string, childTaskIndex int, newName, notes string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptEditChildTask(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, newName, notes))
		})
	})
}

func newDeleteChildTaskCmd() *cobra.Command {
	return commandlib.NewDeleteChildTaskCmd("delete-child-task", "Delete a child task", func(cmd *cobra.Command, args []string, parentName, parentID, childTaskName, childTaskID string, childTaskIndex int) error {
		return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptDeleteChildTask(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex))
		})
	})
}

func newCompleteChildTaskCmd() *cobra.Command {
	return commandlib.NewDeleteChildTaskCmd("complete-child-task", "Mark child task as completed", func(cmd *cobra.Command, args []string, parentName, parentID, childTaskName, childTaskID string, childTaskIndex int) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptSetChildTaskStatus(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, true))
		})
	})
}

func newUncompleteChildTaskCmd() *cobra.Command {
	return commandlib.NewDeleteChildTaskCmd("uncomplete-child-task", "Mark child task as uncompleted", func(cmd *cobra.Command, args []string, parentName, parentID, childTaskName, childTaskID string, childTaskIndex int) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptSetChildTaskStatus(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, false))
		})
	})
}
