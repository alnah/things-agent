package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	commandlib "github.com/alnah/things-agent/internal/command"
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
	return commandlib.NewSetTagsCmd(func(cmd *cobra.Command, args []string, name, id, tags string) error {
		if strings.TrimSpace(tags) == "" {
			return errors.New("--tags is required")
		}
		tagList := parseCSVList(tags)
		if len(tagList) == 0 {
			return errors.New("specify at least one tag in --tags")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			script := fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, cfg.bundleID, scriptResolveItemRef(name, id), escapeApple(strings.Join(tagList, ", ")))
			return runResult(ctx, cfg, script)
		})
	})
}

func newSetTaskTagsCmd() *cobra.Command {
	return commandlib.NewSetTaskTagsCmd(func(cmd *cobra.Command, args []string, name, id, tags string) error {
		if strings.TrimSpace(tags) == "" {
			return errors.New("--tags is required")
		}
		tagList := parseCSVList(tags)
		if len(tagList) == 0 {
			return errors.New("specify at least one tag in --tags")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"tags":       strings.Join(tagList, ", "),
			})
		})
	})
}

func newAddTaskTagsCmd() *cobra.Command {
	return commandlib.NewAddTaskTagsCmd(func(cmd *cobra.Command, args []string, name, id, tags string) error {
		if strings.TrimSpace(tags) == "" {
			return errors.New("--tags is required")
		}
		tagList := parseCSVList(tags)
		if len(tagList) == 0 {
			return errors.New("specify at least one tag in --tags")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"add-tags":   strings.Join(tagList, ", "),
			})
		})
	})
}

func newRemoveTaskTagsCmd() *cobra.Command {
	return commandlib.NewRemoveTaskTagsCmd(func(cmd *cobra.Command, args []string, name, id, tags string) error {
		if strings.TrimSpace(tags) == "" {
			return errors.New("--tags is required")
		}
		tagList := parseCSVList(tags)
		if len(tagList) == 0 {
			return errors.New("specify at least one tag in --tags")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			item, err := readCurrentTaskItem(cmd, cfg, name, id)
			if err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "update", map[string]string{
				"auth-token": token,
				"id":         item.ID,
				"tags":       strings.Join(filterTagList(item.Tags, tagList), ", "),
			})
		})
	})
}

func newSetTaskNotesCmd() *cobra.Command {
	return commandlib.NewSetTaskNotesCmd(func(cmd *cobra.Command, args []string, name, id, notes string) error {
		if strings.TrimSpace(notes) == "" {
			return errors.New("--notes is required")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptSetTaskNotes(cfg.bundleID, name, id, notes))
		})
	})
}

func newAppendTaskNotesCmd() *cobra.Command {
	return commandlib.NewAppendTaskNotesCmd(func(cmd *cobra.Command, args []string, name, id, notes, separator string) error {
		if strings.TrimSpace(notes) == "" {
			return errors.New("--notes is required")
		}
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			return runResult(ctx, cfg, scriptAppendTaskNotes(cfg.bundleID, name, id, notes, separator))
		})
	})
}

func newSetTaskDateCmd() *cobra.Command {
	return commandlib.NewSetTaskDateCmd(func(cmd *cobra.Command, args []string, name, id, due, deadline string, clearDue, clearDeadline bool) error {
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
		return withWriteBackup(cmd, false, func(ctx context.Context, cfg *runtimeConfig) error {
			if clearDue {
				item, err := readCurrentTaskItem(cmd, cfg, name, id)
				if err != nil {
					return err
				}
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				if err := runThingsURL(ctx, cfg, "update", map[string]string{
					"auth-token": token,
					"id":         item.ID,
					"when":       "",
				}); err != nil {
					return err
				}
			}
			if dueDate != "" {
				if err := runResult(ctx, cfg, scriptSetTaskDate(cfg.bundleID, name, id, dueDate, false)); err != nil {
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
		})
	})
}
