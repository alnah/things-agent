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
			paths, err := newSemanticBackupManager(cfg).Create(ctx)
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
	var dryRun bool
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Safely restore a backup timestamp (latest by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			journal, err := newRestoreExecutor(cfg).Execute(ctx, timestamp, dryRun)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(journal)
			}
			if dryRun {
				fmt.Printf("%s\tdry-run=true\tok=%t\n", journal.Timestamp, journal.Preflight.OK)
				return nil
			}
			for _, p := range journal.RestoredFiles {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to restore (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run restore preflight only without mutating live files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	cmd.AddCommand(newRestoreListCmd(), newRestorePreflightCmd(), newRestoreVerifyCmd())
	return cmd
}

func newRestoreListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List restore snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			snapshots, err := newBackupManager(cfg.dataDir).List(ctx)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(snapshots)
			}
			for _, snapshot := range snapshots {
				fmt.Printf("%s\tcomplete=%t\tfiles=%d\n", snapshot.Timestamp, snapshot.Complete, len(snapshot.Files))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	return cmd
}

func newRestoreVerifyCmd() *cobra.Command {
	var timestamp string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that live files match a snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			payload, err := newRestoreExecutor(cfg).Verify(ctx, timestamp)
			if jsonOutput {
				if writeErr := writeJSON(payload); writeErr != nil {
					return writeErr
				}
				return err
			}
			fmt.Printf("%s\tmatch=%t\tcomplete=%t\tfiles=%d\n", payload.Timestamp, payload.Match, payload.Complete, len(payload.Files))
			return err
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to verify (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("timestamp")
	return cmd
}

func newRestorePreflightCmd() *cobra.Command {
	var timestamp string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Validate restore readiness without mutating live files",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			payload, err := newRestoreExecutor(cfg).Preflight(ctx, timestamp)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(payload)
			}
			fmt.Printf("%s\tok=%t\tcomplete=%t\tapp-running=%t\tstable=%t\n", payload.Timestamp, payload.OK, payload.Complete, payload.AppRunning, payload.LiveFilesStable)
			return nil
		},
	}
	cmd.Flags().StringVar(&timestamp, "timestamp", "", "Backup timestamp to validate (YYYY-MM-DD:HH-MM-SS)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
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
			paths, err := newSemanticBackupManager(cfg).Create(ctx)
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
		Short: "List Things areas and built-in lists",
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

func newAreasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "areas",
		Short: "List Things areas",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAllAreas(cfg.bundleID))
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
	cmd.Flags().StringVar(&listName, "list", "", "Limit to a Things list or area")
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
	cmd.Flags().StringVar(&listName, "list", "", "Limit to a Things list or area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output structured JSON")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}
