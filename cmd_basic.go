package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func runThingsURL(ctx context.Context, cfg *runtimeConfig, command string, params map[string]string) error {
	thingsURL := "things:///" + command
	if encoded := encodeThingsURLParams(params); encoded != "" {
		thingsURL += "?" + encoded
	}
	return runResult(ctx, cfg, scriptOpenURL(cfg.bundleID, thingsURL))
}

func encodeThingsURLParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return strings.ReplaceAll(values.Encode(), "+", "%20")
}

func scriptOpenURL(bundleID, rawURL string) string {
	return fmt.Sprintf(`tell application id "%s"
  open location "%s"
end tell
return "ok"`, escapeApple(bundleID), escapeApple(rawURL))
}

func setIfNotEmpty(params map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	params[key] = value
}

func setIfChanged(cmd *cobra.Command, params map[string]string, key, value string) {
	if !cmd.Flags().Changed(key) {
		return
	}
	params[key] = strings.TrimSpace(value)
}

func setBoolIfChanged(cmd *cobra.Command, params map[string]string, key string, value bool) {
	if !cmd.Flags().Changed(key) {
		return
	}
	if value {
		params[key] = "true"
		return
	}
	params[key] = "false"
}

func normalizeChecklistInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "\n") {
		return raw
	}
	items := parseCSVList(raw)
	if len(items) == 0 {
		return raw
	}
	return strings.Join(items, "\n")
}

func newBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Create a Things DB backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			paths, err := newBackupManager(cfg.dataDir).Create(ctx)
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		},
	}
}

func newRestoreCmd() *cobra.Command {
	var timestamp string
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Safely restore a backup timestamp (latest by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			restored, err := newRestoreExecutor(cfg).Restore(ctx, timestamp)
			if err != nil {
				return err
			}
			for _, p := range restored {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to restore (YYYY-MM-DD:HH-MM-SS)")
	return cmd
}

func newSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Create a session backup and prune old backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			paths, err := newBackupManager(cfg.dataDir).Create(ctx)
			if err != nil {
				return err
			}
			for _, p := range paths {
				fmt.Println(p)
			}
			return nil
		},
	}
}

func newListsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lists",
		Short: "List Things areas/lists",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAllLists(cfg.bundleID))
		},
	}
}

func newProjectsCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptAllProjectsStructured(cfg.bundleID), parseProjectListJSON)
			}
			return runResult(ctx, cfg, scriptAllProjects(cfg.bundleID))
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newTasksCmd() *cobra.Command {
	var listName, query string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks (optionally filtered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptTasksStructured(cfg.bundleID, listName, query), parseTaskListJSON)
			}
			return runResult(ctx, cfg, scriptTasks(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&listName, "list", "", "Domaine")
	cmd.Flags().StringVar(&query, "query", "", "Filter by name / notes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var listName, query string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			if jsonOutput {
				return runJSONResult(ctx, cfg, scriptTasksStructured(cfg.bundleID, listName, query), parseTaskListJSON)
			}
			return runResult(ctx, cfg, scriptSearch(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	cmd.Flags().StringVar(&listName, "list", "", "Limit to area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}
