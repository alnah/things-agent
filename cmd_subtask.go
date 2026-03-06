package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newListSubtasksCmd() *cobra.Command {
	var taskName string
	cmd := &cobra.Command{
		Use:   "list-subtasks",
		Short: "List task subtasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(taskName) == "" {
				return errors.New("--task is required")
			}
			return runResult(ctx, cfg, scriptListSubtasks(cfg.bundleID, taskName))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newAddSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	cmd := &cobra.Command{
		Use:   "add-subtask",
		Short: "Add a native checklist item to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" || subtaskName == "" {
				return errors.New("--task and --name are required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendChecklistByName(cfg.bundleID, taskName, []string{subtaskName}, token))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditSubtaskCmd() *cobra.Command {
	var taskName, subtaskName, newName, notes string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-subtask",
		Short: "Edit a subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			newName = strings.TrimSpace(newName)
			notes = strings.TrimSpace(notes)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if newName == "" && notes == "" {
				return errors.New("provide --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex, newName, notes))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Target subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Target subtask index (1-based)")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newDeleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "delete-subtask",
		Short: "Delete a subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newCompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "complete-subtask",
		Short: "Mark subtask as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, true))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newUncompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "uncomplete-subtask",
		Short: "Mark subtask as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task is required")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("provide --index (>=1) or --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, false))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Task name parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Subtask name")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Subtask index (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}
