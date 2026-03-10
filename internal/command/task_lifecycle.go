package command

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func NewShowTaskCmd(runE func(*cobra.Command, []string, string, string, bool, bool) error) *cobra.Command {
	var name, id string
	var withChildTasks bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "show-task",
		Short: "Show full details for a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, withChildTasks, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task or project name")
	cmd.Flags().StringVar(&id, "id", "", "Task or project ID")
	cmd.Flags().BoolVar(&withChildTasks, "with-child-tasks", true, "Include child tasks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func NewAddTaskCmd(resolveDestination func(string, string) (string, string, error), runE func(*cobra.Command, []string, string, string, string, string, string, string, string) error) *cobra.Command {
	var name, notes, tags, areaName, projectName, due, checklistItems string
	cmd := &cobra.Command{
		Use:   "add-task",
		Short: "Add a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			destinationKind, destinationName, err := resolveDestination(areaName, projectName)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, notes, tags, destinationKind, destinationName, due, checklistItems)
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

func NewEditTaskCmd(runE func(*cobra.Command, []string, string, string, string, string, string, string, string, string, string, string) error) *cobra.Command {
	var sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel string
	cmd := &cobra.Command{
		Use:   "edit-task",
		Short: "Edit a task (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			sourceName, sourceID, err = ResolveEntitySelector(sourceName, sourceID)
			if err != nil {
				return err
			}
			return runE(cmd, args, sourceName, sourceID, newName, notes, tags, moveTo, due, completion, creation, cancel)
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

func NewDeleteTaskCmd(use, short string, runE func(*cobra.Command, []string, string, string) error) *cobra.Command {
	var name, id string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	return cmd
}
