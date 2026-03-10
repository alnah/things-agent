package command

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func NewAddChecklistItemCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var taskName, taskID, itemName string
	cmd := &cobra.Command{
		Use:   "add-checklist-item",
		Short: "Add a native checklist item to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			taskName, taskID, err = ResolveTaskParentSelector(taskName, taskID)
			if err != nil {
				return err
			}
			itemName = strings.TrimSpace(itemName)
			if itemName == "" {
				return errors.New("--name is required")
			}
			return runE(cmd, args, taskName, taskID, itemName)
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Task ID parent")
	cmd.Flags().StringVar(&itemName, "name", "", "Checklist item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func NewListChildTasksCmd(runE func(*cobra.Command, []string, string, string) error) *cobra.Command {
	var parentName, parentID string
	cmd := &cobra.Command{
		Use:   "list-child-tasks",
		Short: "List child tasks for a parent item",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			parentName, parentID, err = ResolveParentSelector(parentName, parentID)
			if err != nil {
				return err
			}
			return runE(cmd, args, parentName, parentID)
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	return cmd
}

func NewAddChildTaskCmd(runE func(*cobra.Command, []string, string, string, string, string) error) *cobra.Command {
	var parentName, parentID, childTaskName, notes string
	cmd := &cobra.Command{
		Use:   "add-child-task",
		Short: "Add a child task under a parent item",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			parentName, parentID, err = ResolveParentSelector(parentName, parentID)
			if err != nil {
				return err
			}
			childTaskName = strings.TrimSpace(childTaskName)
			notes = strings.TrimSpace(notes)
			if childTaskName == "" {
				return errors.New("--name is required")
			}
			return runE(cmd, args, parentName, parentID, childTaskName, notes)
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Child task notes")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func NewEditChildTaskCmd(runE func(*cobra.Command, []string, string, string, string, string, int, string, string) error) *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID, newName, notes string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-child-task",
		Short: "Edit a child task",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			parentName, parentID, childTaskName, childTaskIndex, err = ResolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			newName = strings.TrimSpace(newName)
			notes = strings.TrimSpace(notes)
			if newName == "" && notes == "" {
				return errors.New("provide --new-name and/or --notes")
			}
			return runE(cmd, args, parentName, parentID, childTaskName, childTaskID, childTaskIndex, newName, notes)
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

func NewDeleteChildTaskCmd(use, short string, runE func(*cobra.Command, []string, string, string, string, string, int) error) *cobra.Command {
	var parentName, parentID, childTaskName, childTaskID string
	var childTaskIndex int
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			parentName, parentID, childTaskName, childTaskIndex, err = ResolveChildTaskMutationSelector(parentName, parentID, childTaskName, childTaskID, childTaskIndex)
			if err != nil {
				return err
			}
			return runE(cmd, args, parentName, parentID, childTaskName, childTaskID, childTaskIndex)
		},
	}
	cmd.Flags().StringVar(&parentName, "parent", "", "Parent item name")
	cmd.Flags().StringVar(&parentID, "parent-id", "", "Parent item ID")
	cmd.Flags().StringVar(&childTaskName, "name", "", "Child task name")
	cmd.Flags().StringVar(&childTaskID, "id", "", "Child task ID")
	cmd.Flags().IntVar(&childTaskIndex, "index", 0, "Child task index (1-based)")
	return cmd
}
