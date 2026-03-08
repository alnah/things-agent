package main

import (
	"context"
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

func resolveTaskDestination(areaName, projectName string) (string, string, error) {
	areaName = strings.TrimSpace(areaName)
	projectName = strings.TrimSpace(projectName)
	if areaName != "" && projectName != "" {
		return "", "", errors.New("exactly one destination is allowed: use --area or --project")
	}
	if areaName != "" {
		return "area", areaName, nil
	}
	if projectName != "" {
		return "project", projectName, nil
	}
	if fallback := resolveDestinationListName(""); fallback != "" {
		return "area", fallback, nil
	}
	return "", "", errors.New("destination is required: use --area, --project, or THINGS_DEFAULT_LIST")
}

func resolveEntitySelector(name, id string) (string, string, error) {
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	switch {
	case name == "" && id == "":
		return "", "", errors.New("exactly one of --name or --id is required")
	case name != "" && id != "":
		return "", "", errors.New("exactly one of --name or --id is allowed")
	default:
		return name, id, nil
	}
}

func resolveTaskParentSelector(taskName, taskID string) (string, string, error) {
	taskName = strings.TrimSpace(taskName)
	taskID = strings.TrimSpace(taskID)
	switch {
	case taskName == "" && taskID == "":
		return "", "", errors.New("exactly one of --task or --task-id is required")
	case taskName != "" && taskID != "":
		return "", "", errors.New("exactly one of --task or --task-id is allowed")
	default:
		return taskName, taskID, nil
	}
}

func newShowTaskCmd() *cobra.Command {
	var name, id string
	var withChildTasks bool
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
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, id, withChildTasks), parseShowTaskJSON)
			}
			return runResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, id, withChildTasks))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task or project name")
	cmd.Flags().StringVar(&id, "id", "", "Task or project ID")
	cmd.Flags().BoolVar(&withChildTasks, "with-child-tasks", true, "Include child tasks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newAddTaskCmd() *cobra.Command {
	var name, notes, tags, areaName, projectName, due, checklistItems string
	cmd := &cobra.Command{
		Use:   "add-task",
		Short: "Add a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			destinationKind, destinationName, err := resolveTaskDestination(areaName, projectName)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				dueDate, err := parseToAppleDate(due)
				if err != nil {
					return err
				}
				checklistItemsList := parseCSVList(checklistItems)
				var createScript string
				switch destinationKind {
				case "area":
					createScript = scriptAddTaskToArea(cfg.bundleID, destinationName, name, notes, tags, dueDate)
				case "project":
					createScript = scriptAddTaskToProject(cfg.bundleID, destinationName, name, notes, tags, dueDate)
				default:
					return errors.New("unsupported destination kind")
				}
				out, err := cfg.runner.run(ctx, createScript)
				if err != nil {
					return err
				}
				taskID := strings.TrimSpace(out)
				if taskID == "" {
					return errors.New("could not retrieve created task id")
				}
				if len(checklistItemsList) > 0 {
					token, err := requireAuthToken(cfg)
					if err != nil {
						return err
					}
					if _, err := cfg.runner.run(ctx, scriptSetChecklistByID(cfg.bundleID, taskID, checklistItemsList, token)); err != nil {
						return err
					}
				}
				fmt.Println(taskID)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags (comma-separated)")
	cmd.Flags().StringVar(&areaName, "area", "", "Destination area")
	cmd.Flags().StringVar(&projectName, "project", "", "Destination project")
	cmd.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Checklist items (name1, name2, ...)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditTaskCmd() *cobra.Command {
	var sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel string
	cmd := &cobra.Command{
		Use:   "edit-task",
		Short: "Edit a task (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sourceName, sourceID, err = resolveEntitySelector(sourceName, sourceID)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
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
					sourceID,
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
			})
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Task name to edit")
	cmd.Flags().StringVar(&sourceID, "id", "", "Task ID to edit")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&moveTo, "move-to", "", "New area")
	cmd.Flags().StringVar(&due, "due", "", "New due date")
	cmd.Flags().StringVar(&completion, "completion", "", "Completion date")
	cmd.Flags().StringVar(&creation, "creation", "", "Creation date")
	cmd.Flags().StringVar(&cancel, "cancel", "", "Cancellation date")
	return cmd
}

func newDeleteTaskCmd() *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   "delete-task",
		Short: "Delete a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
				return runResult(ctx, cfg, scriptDeleteTaskRef(cfg.bundleID, name, id))
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	return cmd
}

func newCompleteTaskCmd() *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   "complete-task",
		Short: "Mark task as completed",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				return runResult(ctx, cfg, scriptSetTaskCompletionByRef(cfg.bundleID, name, id, true, token))
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	return cmd
}

func newUncompleteTaskCmd() *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   "uncomplete-task",
		Short: "Mark task as uncompleted",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = resolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				return runResult(ctx, cfg, scriptSetTaskCompletionByRef(cfg.bundleID, name, id, false, token))
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	return cmd
}
