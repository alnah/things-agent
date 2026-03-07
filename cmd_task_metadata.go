package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func readCurrentTaskItem(cmd *cobra.Command, cfg *runtimeConfig, name, id string) (readItem, error) {
	out, err := cfg.runner.run(cmd.Context(), scriptShowTask(cfg.bundleID, name, id, false))
	if err != nil {
		return readItem{}, err
	}
	item, err := parseShowTaskOutput(strings.TrimSpace(out))
	if err != nil {
		return readItem{}, fmt.Errorf("read current task tags: %w", err)
	}
	return item, nil
}

func filterTagList(existing, removals []string) []string {
	removeSet := make(map[string]struct{}, len(removals))
	for _, tag := range removals {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		removeSet[tag] = struct{}{}
	}
	filtered := make([]string, 0, len(existing))
	for _, tag := range existing {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := removeSet[tag]; ok {
			continue
		}
		filtered = append(filtered, tag)
	}
	return filtered
}

func newSetTagsCmd() *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "set-tags",
		Short: "Set tags on a task or project",
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
			if strings.TrimSpace(tags) == "" {
				return errors.New("--tags is required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, cfg.bundleID, scriptResolveItemRef(name, id), escapeApple(strings.Join(tagList, ", ")))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task or project ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskTagsCmd() *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "set-task-tags",
		Short: "Set task tags exactly",
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
			if strings.TrimSpace(tags) == "" {
				return errors.New("--tags is required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"tags":       strings.Join(tagList, ", "),
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newAddTaskTagsCmd() *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "add-task-tags",
		Short: "Add tags to a task (merge with existing tags)",
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
			if strings.TrimSpace(tags) == "" {
				return errors.New("--tags is required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"add-tags":   strings.Join(tagList, ", "),
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newRemoveTaskTagsCmd() *cobra.Command {
	var name, id, tags string
	cmd := &cobra.Command{
		Use:   "remove-task-tags",
		Short: "Remove tags from a task",
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
			if strings.TrimSpace(tags) == "" {
				return errors.New("--tags is required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"tags":       strings.Join(filterTagList(item.Tags, tagList), ", "),
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskNotesCmd() *cobra.Command {
	var name, id, notes string
	cmd := &cobra.Command{
		Use:   "set-task-notes",
		Short: "Set task notes",
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
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskNotes(cfg.bundleID, name, id, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newAppendTaskNotesCmd() *cobra.Command {
	var name, id, notes, separator string
	cmd := &cobra.Command{
		Use:   "append-task-notes",
		Short: "Append notes to task notes",
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
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendTaskNotes(cfg.bundleID, name, id, notes, separator))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&id, "id", "", "Task ID")
	cmd.Flags().StringVar(&notes, "notes", "", "Text to append to notes")
	cmd.Flags().StringVar(&separator, "separator", "\n", "Append separator (default: newline)")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newSetTaskDateCmd() *cobra.Command {
	var name, id, due, deadline string
	var clearDue, clearDeadline bool
	cmd := &cobra.Command{
		Use:   "set-task-date",
		Short: "Set/update task due date",
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
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			deadlineDate, err := parseToAppleDate(deadline)
			if err != nil {
				return err
			}
			if !clearDue && !clearDeadline && dueDate == "" && deadlineDate == "" {
				return errors.New("provide --due, --deadline, --clear-due, or --clear-deadline")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			if clearDue || dueDate != "" {
				if err := runResult(ctx, cfg, scriptSetTaskDate(cfg.bundleID, name, id, dueDate, clearDue)); err != nil {
					return err
				}
			}
			if clearDeadline || deadlineDate != "" {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				if clearDeadline && deadlineDate == "" {
					return runResult(ctx, cfg, scriptClearTaskDeadlineByRef(cfg.bundleID, name, id, token))
				}
				if err := runResult(ctx, cfg, scriptSetTaskDeadlineByRef(cfg.bundleID, name, id, deadlineDate, token)); err != nil {
					return err
				}
			}
			return nil
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
