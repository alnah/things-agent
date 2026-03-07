package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func resolveParentSelector(parentName, parentID string) (string, string, error) {
	parentName = strings.TrimSpace(parentName)
	parentID = strings.TrimSpace(parentID)
	switch {
	case parentName == "" && parentID == "":
		return "", "", errors.New("exactly one of --parent or --parent-id is required")
	case parentName != "" && parentID != "":
		return "", "", errors.New("exactly one of --parent or --parent-id is allowed")
	default:
		return parentName, parentID, nil
	}
}

func resolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID string, childTaskIndex int) (string, string, string, int, error) {
	childTaskID = strings.TrimSpace(childTaskID)
	if childTaskID != "" {
		if strings.TrimSpace(parentName) != "" || strings.TrimSpace(parentID) != "" || strings.TrimSpace(childTaskName) != "" || childTaskIndex > 0 {
			return "", "", "", 0, errors.New("use either --id or a parent selector with --name/--index")
		}
		return "", childTaskID, "", 0, nil
	}

	var err error
	parentName, parentID, err = resolveParentSelector(parentName, parentID)
	if err != nil {
		return "", "", "", 0, err
	}
	childTaskName = strings.TrimSpace(childTaskName)
	if childTaskIndex <= 0 && childTaskName == "" {
		return "", "", "", 0, errors.New("provide --id or --index (>=1) or --name")
	}
	return parentName, parentID, childTaskName, childTaskIndex, nil
}

func newAddChecklistItemCmd() *cobra.Command {
	var taskName, taskID, itemName string
	cmd := &cobra.Command{
		Use:   "add-checklist-item",
		Short: "Add a native checklist item to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName, taskID, err = resolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			itemName = strings.TrimSpace(itemName)
			if itemName == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendChecklistByRef(cfg.bundleID, taskName, taskID, []string{itemName}, token))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&itemName, "name", "", "Checklist item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newListChildTasksCmd() *cobra.Command {
	var parentName, parentID string
	cmd := &cobra.Command{
		Use:   "list-child-tasks",
		Short: "List child tasks for a parent item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, err = resolveParentSelector(parentName, parentID)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptListChildTasks(cfg.bundleID, parentName, parentID))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	return cmd
}

func newAddChildTaskCmd() *cobra.Command {
	var parentName, parentID, childTaskName, notes string
	cmd := &cobra.Command{
		Use:   "add-child-task",
		Short: "Add a child task under a parent item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, err = resolveParentSelector(parentName, parentID)
			if err != nil {
				return err
			}
			childTaskName = strings.TrimSpace(childTaskName)
			notes = strings.TrimSpace(notes)
			if childTaskName == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddChildTask(cfg.bundleID, parentName, parentID, childTaskName, notes))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Child task notes")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditChildTaskCmd() *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID, newName, notes string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-child-task",
		Short: "Edit a child task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, childTaskName, childTaskIndex, err = resolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			newName = strings.TrimSpace(newName)
			notes = strings.TrimSpace(notes)
			if newName == "" && notes == "" {
				return errors.New("provide --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditChildTask(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, newName, notes))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Target child task name")
	cmd.Flags().StringVar(&childTaskID, "id", "", "Target child task ID")
	cmd.Flags().IntVar(&childTaskIndex, "index", 0, "Target child task index (1-based)")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	return cmd
}

func newDeleteChildTaskCmd() *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   "delete-child-task",
		Short: "Delete a child task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, childTaskName, childTaskIndex, err = resolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteChildTask(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&childTaskID, "id", "", "Child task ID")
	cmd.Flags().IntVar(&childTaskIndex, "index", 0, "Child task index (1-based)")
	return cmd
}

func newCompleteChildTaskCmd() *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   "complete-child-task",
		Short: "Mark child task as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, childTaskName, childTaskIndex, err = resolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetChildTaskStatus(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, true))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&childTaskID, "id", "", "Child task ID")
	cmd.Flags().IntVar(&childTaskIndex, "index", 0, "Child task index (1-based)")
	return cmd
}

func newUncompleteChildTaskCmd() *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   "uncomplete-child-task",
		Short: "Mark child task as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			parentName, parentID, childTaskName, childTaskIndex, err = resolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetChildTaskStatus(cfg.bundleID, parentName, parentID, childTaskName, childTaskID, childTaskIndex, false))
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&childTaskID, "id", "", "Child task ID")
	cmd.Flags().IntVar(&childTaskIndex, "index", 0, "Child task index (1-based)")
	return cmd
}
