package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func resolveDestinationListName(value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return envOrDefault("THINGS_DEFAULT_LIST", "")
}

func newShowTaskCmd() *cobra.Command {
	var name string
	var withSubtasks bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "show-task",
		Short: "Show full details for a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return errors.New("--name is required")
			}
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, withSubtasks), parseShowTaskJSON)
			}
			return runResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, withSubtasks))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task or project name")
	cmd.Flags().BoolVar(&withSubtasks, "with-subtasks", true, "Include subtasks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddTaskCmd() *cobra.Command {
	var name, notes, tags, listName, due, subtasks string
	cmd := &cobra.Command{
		Use:   "add-task",
		Short: "Add a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			listName = resolveDestinationListName(listName)
			if listName == "" {
				return errors.New("destination is required: use --list or THINGS_DEFAULT_LIST")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			subtasksList := parseCSVList(subtasks)
			out, err := cfg.runner.run(ctx, scriptAddTask(cfg.bundleID, listName, name, notes, tags, dueDate))
			if err != nil {
				return err
			}
			taskID := strings.TrimSpace(out)
			if taskID == "" {
				return errors.New("could not retrieve created task id")
			}
			if len(subtasksList) > 0 {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				if _, err := cfg.runner.run(ctx, scriptSetChecklistByID(cfg.bundleID, taskID, subtasksList, token)); err != nil {
					return err
				}
			}
			fmt.Println(taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags (comma-separated)")
	cmd.Flags().StringVar(&listName, "list", "", "Destination area")
	cmd.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&subtasks, "subtasks", "", "Subtasks (name1, name2, ...)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditTaskCmd() *cobra.Command {
	var sourceName, newName, notes, tags, moveTo, due, completion, creation, cancel string
	cmd := &cobra.Command{
		Use:   "edit-task",
		Short: "Edit a task (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}

			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			completionDate, err := parseToAppleDate(completion)
			if err != nil {
				return err
			}
			creationDate, err := parseToAppleDate(creation)
			if err != nil {
				return err
			}
			cancelDate, err := parseToAppleDate(cancel)
			if err != nil {
				return err
			}

			script, err := scriptEditTask(
				cfg.bundleID,
				sourceName,
				newName,
				notes,
				tags,
				moveTo,
				dueDate,
				completionDate,
				creationDate,
				cancelDate,
			)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Task name to edit")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&moveTo, "move-to", "", "New area")
	cmd.Flags().StringVar(&due, "due", "", "New due date")
	cmd.Flags().StringVar(&completion, "completion", "", "Completion date")
	cmd.Flags().StringVar(&creation, "creation", "", "Creation date")
	cmd.Flags().StringVar(&cancel, "cancel", "", "Cancellation date")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDeleteTaskCmd() *cobra.Command {
	return newDeleteCmd("task", "delete-task")
}

func newCompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "complete-task",
		Short: "Mark task as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, true))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUncompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "uncomplete-task",
		Short: "Mark task as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, false))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
