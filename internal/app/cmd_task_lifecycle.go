package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commandlib "github.com/alnah/things-agent/internal/command"
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
	return commandlib.ResolveTaskDestination(areaName, projectName, func() string {
		return resolveDestinationListName("")
	})
}

func resolveEntitySelector(name, id string) (string, string, error) {
	return commandlib.ResolveEntitySelector(name, id)
}

func newShowTaskCmd() *cobra.Command {
	return commandlib.NewShowTaskCmd(func(cmd *cobra.Command, args []string, name, id string, withChildTasks, jsonOutput bool) error {
		return withRuntimeConfig(cmd, func(ctx context.Context, cfg *runtimeConfig) error {
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, id, withChildTasks), parseShowTaskJSON)
			}
			return runResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, id, withChildTasks))
		})
	})
}

func newAddTaskCmd() *cobra.Command {
	return commandlib.NewAddTaskCmd(resolveTaskDestination, func(cmd *cobra.Command, args []string, name, notes, tags, destinationKind, destinationName, due, checklistItems string) error {
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
	})
}

func newEditTaskCmd() *cobra.Command {
	return commandlib.NewEditTaskCmd(func(cmd *cobra.Command, args []string, sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel string) error {
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
	})
}

func newDeleteTaskCmd() *cobra.Command {
	return commandlib.NewDeleteTaskCmd("delete-task", "Delete a task", func(cmd *cobra.Command, args []string, name, id string) error {
		return withWriteBackup(cmd, true, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptDeleteTaskRef(cfg.bundleID, name, id))
		})
	})
}

func newCompleteTaskCmd() *cobra.Command {
	return commandlib.NewDeleteTaskCmd("complete-task", "Mark task as completed", func(cmd *cobra.Command, args []string, name, id string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskCompletionByRef(cfg.bundleID, name, id, true, token))
		})
	})
}

func newUncompleteTaskCmd() *cobra.Command {
	return commandlib.NewDeleteTaskCmd("uncomplete-task", "Mark task as uncompleted", func(cmd *cobra.Command, args []string, name, id string) error {
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskCompletionByRef(cfg.bundleID, name, id, false, token))
		})
	})
}
