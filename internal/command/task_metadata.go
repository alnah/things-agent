package command

import (
	"github.com/spf13/cobra"
)

func NewSetTagsCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "set-tags",
		Short: "Set tags on a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, tags)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task or project ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func NewSetTaskTagsCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "set-task-tags",
		Short: "Set task tags exactly",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, tags)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func NewAddTaskTagsCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "add-task-tags",
		Short: "Add tags to a task (merge with existing tags)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, tags)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func NewRemoveTaskTagsCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "remove-task-tags",
		Short: "Remove tags from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, tags)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func NewSetTaskNotesCmd(runE func(*cobra.Command, []string, string, string, string) error) *cobra.Command {
	var name, id, notes string
	cmd := &cobra.Command{
		Use:   "set-task-notes",
		Short: "Set task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, notes)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func NewAppendTaskNotesCmd(runE func(*cobra.Command, []string, string, string, string, string) error) *cobra.Command {
	var name, id, notes, separator string
	cmd := &cobra.Command{
		Use:   "append-task-notes",
		Short: "Append notes to task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, notes, separator)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&notes, "notes", "", "Text to append to notes")
	cmd.Flags().StringVar(&separator, "separator", "\n", "Append separator (default: newline)")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func NewSetTaskDateCmd(runE func(*cobra.Command, []string, string, string, string, string, bool, bool) error) *cobra.Command {
	var name, id, due, deadline string
	var clearDue, clearDeadline bool
	cmd := &cobra.Command{
		Use:   "set-task-date",
		Short: "Set/update task due date",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			name, id, err = ResolveEntitySelector(name, id)
			if err != nil {
				return err
			}
			return runE(cmd, args, name, id, due, deadline, clearDue, clearDeadline)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&due, "due", "", "New due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&deadline, "deadline", "", "New deadline (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear due date")
	cmd.Flags().BoolVar(&clearDeadline, "clear-deadline", false, "Clear deadline")
	return cmd
}
