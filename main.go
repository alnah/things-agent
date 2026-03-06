package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultBundleID    = "com.culturedcode.ThingsMac"
	defaultDataPathRel = "Library/Group Containers/<THINGS_GROUP_CONTAINER>/<THINGS_DATA_DIR_ID>/Things Database.thingsdatabase"
	backupDirName      = "backups"
	backupTSFormat     = "2006-01-02:15-04-05"
	maxBackupsToKeep   = 50
	defaultListName    = "Inbox"
	cliVersion         = "0.3.0"
)

var config = struct {
	bundleID  string
	dataDir   string
	authToken string
}{
	bundleID: envOrDefault("THINGS_BUNDLE_ID", defaultBundleID),
}

type runtimeConfig struct {
	bundleID  string
	dataDir   string
	authToken string
	runner    *runner
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "agent-things",
		SilenceErrors: false,
		SilenceUsage:  true,
		Short:         "Things CLI via AppleScript (no direct DB access)",
		Long: `This CLI controls Things through AppleScript only.
It creates a timestamped backup in YYYY-MM-DD:hh-mm-ss format
before each write action.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&config.bundleID, "bundle-id", envOrDefault("THINGS_BUNDLE_ID", defaultBundleID), "Things app bundle id")
	root.PersistentFlags().StringVar(&config.dataDir, "data-dir", envOrDefault("THINGS_DATA_DIR", ""), "Things database path")
	root.PersistentFlags().StringVar(&config.authToken, "auth-token", envOrDefault("THINGS_AUTH_TOKEN", ""), "Things URL Scheme auth token (Settings > General)")

	root.AddCommand(
		newBackupCmd(),
		newRestoreCmd(),
		newSessionStartCmd(),
		newURLCmd(),
		newListsCmd(),
		newProjectsCmd(),
		newTasksCmd(),
		newSearchCmd(),
		newShowTaskCmd(),
		newAddTaskCmd(),
		newAddProjectCmd(),
		newAddListCmd(),
		newEditTaskCmd(),
		newEditProjectCmd(),
		newEditListCmd(),
		newDeleteTaskCmd(),
		newDeleteProjectCmd(),
		newDeleteListCmd(),
		newCompleteTaskCmd(),
		newUncompleteTaskCmd(),
		newSetTagsCmd(),
		newSetTaskTagsCmd(),
		newAddTaskTagsCmd(),
		newRemoveTaskTagsCmd(),
		newSetTaskNotesCmd(),
		newAppendTaskNotesCmd(),
		newSetTaskDateCmd(),
		newListSubtasksCmd(),
		newAddSubtaskCmd(),
		newEditSubtaskCmd(),
		newDeleteSubtaskCmd(),
		newCompleteSubtaskCmd(),
		newUncompleteSubtaskCmd(),
		&cobra.Command{
			Use:   "version",
			Short: "Show version",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("things", cliVersion)
			},
		},
	)

	return root
}

func resolveRuntimeConfig(ctx context.Context) (*runtimeConfig, error) {
	dataDir := strings.TrimSpace(config.dataDir)
	if dataDir == "" {
		var err error
		dataDir, err = resolveDataDir()
		if err != nil {
			return nil, err
		}
	}

	r := newRunner(config.bundleID)
	if err := r.ensureReachable(ctx); err != nil {
		return nil, err
	}

	return &runtimeConfig{
		bundleID:  config.bundleID,
		dataDir:   dataDir,
		authToken: strings.TrimSpace(config.authToken),
		runner:    r,
	}, nil
}

func backupIfNeeded(ctx context.Context, cfg *runtimeConfig) error {
	bm := newBackupManager(cfg.dataDir)
	paths, err := bm.Create(ctx)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	_ = paths
	return nil
}

func runResult(ctx context.Context, cfg *runtimeConfig, script string) error {
	out, err := cfg.runner.run(ctx, script)
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out != "" {
		fmt.Println(out)
	}
	return nil
}

func runThingsURL(ctx context.Context, cfg *runtimeConfig, command string, params map[string]string) error {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	thingsURL := "things:///" + command
	if encoded := values.Encode(); encoded != "" {
		thingsURL += "?" + encoded
	}
	return runResult(ctx, cfg, scriptOpenURL(thingsURL))
}

func scriptOpenURL(rawURL string) string {
	return fmt.Sprintf(`open location "%s"
return "ok"`, escapeApple(rawURL))
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
	var target string
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a backup (latest by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			bm := newBackupManager(cfg.dataDir)
			if strings.TrimSpace(target) == "" {
				ts, err := bm.Latest(ctx)
				if err != nil {
					return err
				}
				restored, err := bm.Restore(ctx, ts)
				if err != nil {
					return err
				}
				for _, p := range restored {
					fmt.Println(p)
				}
				return nil
			}

			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				if err := bm.RestoreFile(ctx, target); err != nil {
					return err
				}
				fmt.Println(target)
				return nil
			}

			ts := inferTimestamp(target)
			if ts == "" {
				ts = target
			}
			restored, err := bm.Restore(ctx, ts)
			if err != nil {
				return err
			}
			for _, p := range restored {
				fmt.Println(p)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "file", "", "Chemin du fichier backup (optionnel)")
	return cmd
}

func newSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Initialiser la session (backup + purge des anciens backups)",
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
		Short: "Lister les domaines",
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
	return &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAllProjects(cfg.bundleID))
		},
	}
}

func newTasksCmd() *cobra.Command {
	var listName, query string
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks (optionally filtered)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptTasks(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&listName, "list", "", "Domaine")
	cmd.Flags().StringVar(&query, "query", "", "Filter by name / notes")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var listName, query string
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
			return runResult(ctx, cfg, scriptSearch(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	cmd.Flags().StringVar(&listName, "list", "", "Limit to area")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url",
		Short: "Things URL Scheme commands (official API)",
	}
	cmd.AddCommand(
		newURLAddCmd(),
		newURLUpdateCmd(),
		newURLAddProjectCmd(),
		newURLUpdateProjectCmd(),
		newURLShowCmd(),
		newURLSearchCmd(),
		newURLVersionCmd(),
		newURLAddJSONCmd(),
	)
	return cmd
}

func newURLAddCmd() *cobra.Command {
	var (
		title, notes, when, deadline, tags, checklistItems, listName, listID, heading, headingID, notesTemplate string
		completed, canceled, reveal                                                                               bool
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "things:///add",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfNotEmpty(params, "list", listName)
			setIfNotEmpty(params, "list-id", listID)
			setIfNotEmpty(params, "heading", heading)
			setIfNotEmpty(params, "heading-id", headingID)
			setIfNotEmpty(params, "notes-template", notesTemplate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add", params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When field (today, tomorrow, evening, someday, etc.)")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Checklist (lignes ou CSV)")
	cmd.Flags().StringVar(&listName, "list", "", "Destination project/area name")
	cmd.Flags().StringVar(&listID, "list-id", "", "Destination project/area ID")
	cmd.Flags().StringVar(&heading, "heading", "", "Destination heading name")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID du heading destination")
	cmd.Flags().StringVar(&notesTemplate, "notes-template", "", "replace-title|replace-notes|replace-checklist-items")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal after creation")
	return cmd
}

func newURLUpdateCmd() *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, checklistItems, prependChecklist, appendChecklist string
		listName, listID, heading, headingID                                                                                            string
		completed, canceled, reveal, duplicate                                                                                         bool
		creationDate, completionDate                                                                                                   string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "things:///update",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params := map[string]string{
				"auth-token": token,
				"id":         id,
			}
			setIfNotEmpty(params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "checklist-items", normalizeChecklistInput(checklistItems))
			setIfChanged(cmd, params, "prepend-checklist-items", normalizeChecklistInput(prependChecklist))
			setIfChanged(cmd, params, "append-checklist-items", normalizeChecklistInput(appendChecklist))
			setIfChanged(cmd, params, "list", listName)
			setIfChanged(cmd, params, "list-id", listID)
			setIfChanged(cmd, params, "heading", heading)
			setIfChanged(cmd, params, "heading-id", headingID)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			return runThingsURL(ctx, cfg, "update", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID of the to-do to update")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes (empty to clear)")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Replace tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&checklistItems, "checklist-items", "", "Replace checklist (lines or CSV)")
	cmd.Flags().StringVar(&prependChecklist, "prepend-checklist-items", "", "Prepend checklist")
	cmd.Flags().StringVar(&appendChecklist, "append-checklist-items", "", "Append checklist")
	cmd.Flags().StringVar(&listName, "list", "", "Destination project/area")
	cmd.Flags().StringVar(&listID, "list-id", "", "Destination project/area ID")
	cmd.Flags().StringVar(&heading, "heading", "", "Heading destination")
	cmd.Flags().StringVar(&headingID, "heading-id", "", "ID heading destination")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed status")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled status")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal item")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newURLAddProjectCmd() *cobra.Command {
	var (
		title, notes, when, deadline, tags, area, areaID, todos, creationDate, completionDate string
		completed, canceled, reveal                                                             bool
	)
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "things:///add-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "title", title)
			setIfNotEmpty(params, "notes", notes)
			setIfNotEmpty(params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfNotEmpty(params, "tags", tags)
			setIfNotEmpty(params, "area", area)
			setIfNotEmpty(params, "area-id", areaID)
			setIfNotEmpty(params, "to-dos", normalizeChecklistInput(todos))
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add-project", params)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Project title")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&area, "area", "", "Destination area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&todos, "to-dos", "", "Initial to-dos (lines or CSV)")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Create as completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Create as canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	return cmd
}

func newURLUpdateProjectCmd() *cobra.Command {
	var (
		id, title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, area, areaID, creationDate, completionDate string
		completed, canceled, reveal, duplicate                                                                                bool
	)
	cmd := &cobra.Command{
		Use:   "update-project",
		Short: "things:///update-project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(id) == "" {
				return errors.New("--id is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			token, err := requireAuthToken(cfg)
			if err != nil {
				return err
			}
			params := map[string]string{
				"auth-token": token,
				"id":         id,
			}
			setIfChanged(cmd, params, "title", title)
			setIfChanged(cmd, params, "notes", notes)
			setIfChanged(cmd, params, "prepend-notes", prependNotes)
			setIfChanged(cmd, params, "append-notes", appendNotes)
			setIfChanged(cmd, params, "when", when)
			setIfChanged(cmd, params, "deadline", deadline)
			setIfChanged(cmd, params, "tags", tags)
			setIfChanged(cmd, params, "add-tags", addTags)
			setIfChanged(cmd, params, "area", area)
			setIfChanged(cmd, params, "area-id", areaID)
			setIfChanged(cmd, params, "creation-date", creationDate)
			setIfChanged(cmd, params, "completion-date", completionDate)
			setBoolIfChanged(cmd, params, "completed", completed)
			setBoolIfChanged(cmd, params, "canceled", canceled)
			setBoolIfChanged(cmd, params, "reveal", reveal)
			setBoolIfChanged(cmd, params, "duplicate", duplicate)
			return runThingsURL(ctx, cfg, "update-project", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Project ID")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "Prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "Append notes")
	cmd.Flags().StringVar(&when, "when", "", "When")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Deadline (vide pour effacer)")
	cmd.Flags().StringVar(&tags, "tags", "", "Remplacer tags")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "Add tags")
	cmd.Flags().StringVar(&area, "area", "", "Area destination")
	cmd.Flags().StringVar(&areaID, "area-id", "", "Destination area ID")
	cmd.Flags().StringVar(&creationDate, "creation-date", "", "Creation date ISO8601")
	cmd.Flags().StringVar(&completionDate, "completion-date", "", "Completion date ISO8601")
	cmd.Flags().BoolVar(&completed, "completed", false, "Set completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "Set canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal project")
	cmd.Flags().BoolVar(&duplicate, "duplicate", false, "Duplicate before update")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newURLShowCmd() *cobra.Command {
	var id, query, filter string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "things:///show",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			params := map[string]string{}
			setIfNotEmpty(params, "id", id)
			setIfNotEmpty(params, "query", query)
			setIfNotEmpty(params, "filter", filter)
			if len(params) == 0 {
				return errors.New("fournir au moins --id ou --query")
			}
			return runThingsURL(ctx, cfg, "show", params)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "ID to reveal (or built-in list)")
	cmd.Flags().StringVar(&query, "query", "", "Recherche quick find")
	cmd.Flags().StringVar(&filter, "filter", "", "Tags de filtre (CSV)")
	return cmd
}

func newURLSearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "things:///search",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query is required")
			}
			return runThingsURL(ctx, cfg, "search", map[string]string{"query": query})
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search text")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newURLVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "things:///version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			return runThingsURL(ctx, cfg, "version", map[string]string{})
		},
	}
}

func newURLAddJSONCmd() *cobra.Command {
	var data string
	var reveal bool
	cmd := &cobra.Command{
		Use:   "add-json",
		Short: "things:///add-json",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(data) == "" {
				return errors.New("--data is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			params := map[string]string{"data": data}
			if strings.Contains(data, `"operation":"update"`) || strings.Contains(data, `"operation": "update"`) {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				params["auth-token"] = token
			}
			setBoolIfChanged(cmd, params, "reveal", reveal)
			return runThingsURL(ctx, cfg, "add-json", params)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "Payload JSON")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Reveal created item")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func newShowTaskCmd() *cobra.Command {
	var name string
	var withSubtasks bool
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
			return runResult(ctx, cfg, scriptShowTask(cfg.bundleID, name, withSubtasks))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task or project name")
	cmd.Flags().BoolVar(&withSubtasks, "with-subtasks", true, "Include subtasks")
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
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			subtasksList := parseCSVList(subtasks)
			out, err := cfg.runner.run(ctx, scriptAddTask(cfg.bundleID, strings.TrimSpace(listName), name, notes, tags, dueDate))
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
	cmd.Flags().StringVar(&listName, "list", envOrDefault("THINGS_DEFAULT_LIST", defaultListName), "Destination area")
	cmd.Flags().StringVar(&due, "due", "", "Due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&subtasks, "subtasks", "", "Subtasks (name1, name2, ...)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddProjectCmd() *cobra.Command {
	var name, notes, listName string
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "Add a project",
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
			return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, strings.TrimSpace(listName), name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&listName, "list", envOrDefault("THINGS_DEFAULT_LIST", defaultListName), "Destination area")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddListCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-list",
		Short: "Add an area",
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
			script := fmt.Sprintf(`tell application id "%s"
  make new area with properties {name:"%s"}
  return "ok"
end tell`, cfg.bundleID, escapeApple(name))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Area name")
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

func newEditProjectCmd() *cobra.Command {
	var sourceName, newName, notes string
	cmd := &cobra.Command{
		Use:   "edit-project",
		Short: "Edit a project (by name)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("specify --new-name and/or --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditProject(cfg.bundleID, sourceName, newName, notes))
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Project name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditListCmd() *cobra.Command {
	var sourceName, newName string
	cmd := &cobra.Command{
		Use:   "edit-list",
		Short: "Rename an area",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(newName) == "" {
				return errors.New("--new-name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  set name of l to "%s"
  return "ok"
end tell`, cfg.bundleID, escapeApple(sourceName), escapeApple(newName))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Area name")
	cmd.Flags().StringVar(&newName, "new-name", "", "New name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newDeleteTaskCmd() *cobra.Command {
	return newDeleteCmd("task", "delete-task")
}

func newDeleteProjectCmd() *cobra.Command {
	return newDeleteCmd("project", "delete-project")
}

func newDeleteListCmd() *cobra.Command {
	return newDeleteCmd("list", "delete-list")
}

func newDeleteCmd(kind, name string) *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   name,
		Short: "Delete an item",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(target) == "" {
				return errors.New("--name is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script, err := scriptDelete(cfg.bundleID, kind, target)
			if err != nil {
				return err
			}
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&target, "name", "", "Item name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
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

func newSetTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "set-tags",
		Short: "Set tags on a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, cfg.bundleID, scriptResolveTaskByName(name), escapeApple(tags))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "set-task-tags",
		Short: "Set task tags exactly",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newAddTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "add-task-tags",
		Short: "Add tags to a task (merge with existing tags)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newRemoveTaskTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "remove-task-tags",
		Short: "Remove tags from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name and --tags are required")
			}
			tagList := parseCSVList(tags)
			if len(tagList) == 0 {
				return errors.New("specify at least one tag in --tags")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptRemoveTaskTags(cfg.bundleID, name, tagList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tagss")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newSetTaskNotesCmd() *cobra.Command {
	var name, notes string
	cmd := &cobra.Command{
		Use:   "set-task-notes",
		Short: "Set task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetTaskNotes(cfg.bundleID, name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "New notes")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newAppendTaskNotesCmd() *cobra.Command {
	var name, notes, separator string
	cmd := &cobra.Command{
		Use:   "append-task-notes",
		Short: "Append notes to task notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			if strings.TrimSpace(notes) == "" {
				return errors.New("--notes is required")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAppendTaskNotes(cfg.bundleID, name, notes, separator))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&notes, "notes", "", "Text to append to notes")
	cmd.Flags().StringVar(&separator, "separator", "\n", "Append separator (default: newline)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("notes")
	return cmd
}

func newSetTaskDateCmd() *cobra.Command {
	var name, due, deadline string
	var clear bool
	cmd := &cobra.Command{
		Use:   "set-task-date",
		Short: "Set/update task due date",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			deadlineDate, err := parseToAppleDate(deadline)
			if err != nil {
				return err
			}
			if !clear && dueDate == "" && deadlineDate == "" {
				return errors.New("provide --due, --deadline, or --clear")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			if clear && dueDate == "" && deadlineDate == "" {
				token, err := requireAuthToken(cfg)
				if err != nil {
					return err
				}
				return runResult(ctx, cfg, scriptClearTaskDeadlineByName(cfg.bundleID, name, token))
			}
			return runResult(ctx, cfg, scriptSetTaskDate(cfg.bundleID, name, dueDate, deadlineDate, clear))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&due, "due", "", "New due date (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&deadline, "deadline", "", "Due date alias (same format)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear due date")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

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

type runner struct {
	bundleID string
}

func newRunner(bundleID string) *runner {
	return &runner{
		bundleID: bundleID,
	}
}

func (r *runner) run(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (r *runner) ensureReachable(ctx context.Context) error {
	script := fmt.Sprintf(`tell application id "%s"
  return name
end tell`, r.bundleID)
	if _, err := r.run(ctx, script); err != nil {
		return fmt.Errorf("Things app not found (%s): %w", r.bundleID, err)
	}
	return nil
}

func scriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
}

func scriptResolveTaskByName(taskName string) string {
	taskName = escapeApple(taskName)
	return fmt.Sprintf(`  try
    set t to first project whose name is "%s"
  on error
    set t to first «class tstk» whose name is "%s"
  end try
`, taskName, taskName)
}

func scriptAllProjects(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of projects
end tell`, bundleID)
}

func scriptTasks(bundleID, listName, query string) string {
	listName = strings.TrimSpace(listName)
	query = strings.TrimSpace(query)
	if listName == "" && query == "" {
		return fmt.Sprintf(`tell application id "%s"
  return name of (every «class tstk»)
end tell`, bundleID)
	}
	if listName == "" {
		return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  return name of (every «class tstk» whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query))
	}
	if query == "" {
		return fmt.Sprintf(`tell application id "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l)
end tell`, bundleID, escapeApple(listName))
	}
	return fmt.Sprintf(`tell application id "%s"
  set q to "%s"
  set l to first list whose name is "%s"
  return name of (every «class tstk» of l whose (name contains q or notes contains q))
end tell`, bundleID, escapeApple(query), escapeApple(listName))
}

func scriptSearch(bundleID, listName, query string) string {
	return scriptTasks(bundleID, listName, query)
}

func scriptAddTask(bundleID, listName, name, notes, tags, due string) string {
	if strings.TrimSpace(listName) == "" {
		listName = envOrDefault("THINGS_DEFAULT_LIST", defaultListName)
	}
	parts := []string{fmt.Sprintf(`name:"%s"`, escapeApple(name))}
	if strings.TrimSpace(notes) != "" {
		parts = append(parts, fmt.Sprintf(`notes:"%s"`, escapeApple(notes)))
	}
	if strings.TrimSpace(tags) != "" {
		parts = append(parts, fmt.Sprintf(`tag names:"%s"`, escapeApple(tags)))
	}
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set t to make new «class tstk» at end of to dos of targetList with properties {%s}
`, bundleID, escapeApple(listName), strings.Join(parts, ", "))
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	script += `  return id of t
end tell`
	return script
}

func requireAuthToken(cfg *runtimeConfig) (string, error) {
	token := strings.TrimSpace(cfg.authToken)
	if token == "" {
		return "", errors.New("auth-token is required for native checklist (Things > Settings > General). Use --auth-token or THINGS_AUTH_TOKEN")
	}
	return token, nil
}

func urlEncodeChecklist(items []string) string {
	return url.QueryEscape(strings.Join(items, "\n"))
}

func scriptSetChecklistByID(bundleID, taskID string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first to do whose id is "%s"
  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&checklist-items=%s"
return tid`, bundleID, escapeApple(taskID), escapeApple(url.QueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func scriptAppendChecklistByName(bundleID, taskName string, items []string, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
  set t to first to do whose name is "%s"
  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&append-checklist-items=%s"
return tid`, bundleID, escapeApple(taskName), escapeApple(url.QueryEscape(authToken)), escapeApple(urlEncodeChecklist(items)))
}

func parseCSVList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func scriptListLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, fmt.Sprintf(`"%s"`, escapeApple(value)))
	}
	return "{" + strings.Join(items, ", ") + "}"
}

func scriptAddProject(bundleID, listName, name, notes string) string {
	if strings.TrimSpace(listName) == "" {
		listName = envOrDefault("THINGS_DEFAULT_LIST", defaultListName)
	}
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set p to make new project at end of to dos of targetList with properties {name:"%s"}
`, bundleID, escapeApple(listName), escapeApple(name))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptEditTask(bundleID, source, newName, notes, tags, moveTo, due, completion, creation, cancel string) (string, error) {
	if source == "" {
		return "", errors.New("source name is required")
	}
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskByName(source))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of t to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of t to "%s"
`, escapeApple(notes))
	}
	if strings.TrimSpace(tags) != "" {
		script += fmt.Sprintf(`  set tag names of t to "%s"
`, escapeApple(tags))
	}
	if strings.TrimSpace(moveTo) != "" {
	script += fmt.Sprintf(`  move t to end of to dos of (first list whose name is "%s")
`, escapeApple(moveTo))
	}
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	if strings.TrimSpace(completion) != "" {
		script += fmt.Sprintf(`  set completion date of t to date "%s"
`, completion)
	}
	if strings.TrimSpace(creation) != "" {
		script += fmt.Sprintf(`  set creation date of t to date "%s"
`, creation)
	}
	if strings.TrimSpace(cancel) != "" {
		script += fmt.Sprintf(`  set cancellation date of t to date "%s"
`, cancel)
	}
	script += `  return id of t
end tell`
	return script, nil
}

func scriptEditProject(bundleID, source, newName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
  set p to first project whose name is "%s"
`, bundleID, escapeApple(source))
	if strings.TrimSpace(newName) != "" {
		script += fmt.Sprintf(`  set name of p to "%s"
`, escapeApple(newName))
	}
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of p to "%s"
`, escapeApple(notes))
	}
	script += `  return id of p
end tell`
	return script
}

func scriptSetTaskNotes(bundleID, taskName, notes string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set notes of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(notes))
}

func scriptAppendTaskNotes(bundleID, taskName, notes, separator string) string {
	if strings.TrimSpace(separator) == "" {
		separator = "\n"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  if (notes of t is missing value) or (notes of t is "") then
    set notes of t to "%s"
  else
    set notes of t to (notes of t & "%s" & "%s")
  end if
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(notes), escapeApple(separator), escapeApple(notes))
}

func scriptSetTaskDate(bundleID, taskName, dueDate, deadlineDate string, clear bool) string {
	script := fmt.Sprintf(`tell application id "%s"
%s`, bundleID, scriptResolveTaskByName(taskName))
	if clear {
		script += `  set due date of t to missing value
`
	}
	if strings.TrimSpace(dueDate) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, dueDate)
	}
	if strings.TrimSpace(deadlineDate) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, deadlineDate)
	}
	script += `  return id of t
	end tell`
	return script
}

func scriptClearTaskDeadlineByName(bundleID, taskName, authToken string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set tid to id of t
end tell
open location "things:///update?auth-token=%s&id=" & tid & "&deadline="
return tid`, bundleID, scriptResolveTaskByName(taskName), escapeApple(url.QueryEscape(authToken)))
}

func scriptSetTaskTags(bundleID, taskName string, tags []string) string {
	tagText := strings.Join(tags, ", ")
	return fmt.Sprintf(`tell application id "%s"
%s  set tag names of t to "%s"
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), escapeApple(tagText))
}

func scriptAddTaskTags(bundleID, taskName string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  repeat with aTag in %s
    if not (aTag is in existingTags) then
      set end of existingTags to (aTag as string)
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set mergedTagsText to existingTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to mergedTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), scriptListLiteral(tags))
}

func scriptRemoveTaskTags(bundleID, taskName string, tags []string) string {
	return fmt.Sprintf(`tell application id "%s"
%s  set existingTags to {}
  try
    set existingTags to tag names of t
  end try
  if existingTags is missing value then
    set existingTags to {}
  else if class of existingTags is text then
    set existingTags to {existingTags as string}
  end if
  set filteredTags to {}
  repeat with aTag in existingTags
    if not (aTag is in %s) then
      set end of filteredTags to aTag
    end if
  end repeat
  set AppleScript's text item delimiters to ", "
  set filteredTagsText to filteredTags as text
  set AppleScript's text item delimiters to ""
  set tag names of t to filteredTagsText
  return id of t
end tell`, bundleID, scriptResolveTaskByName(taskName), scriptListLiteral(tags))
}

func scriptListSubtasks(bundleID, taskName string) string {
	taskName = strings.TrimSpace(taskName)
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set subtasks to to dos of t
    set out to ""
    repeat with i from 1 to count subtasks
      set s to item i of subtasks
      set outLine to (i as string) & ". " & (name of s)
      if (notes of s is not missing value) and (notes of s is not "") then
        set outLine to outLine & " | " & (notes of s)
      end if
      if out is "" then
        set out to outLine
      else
        set out to out & linefeed & outLine
      end if
    end repeat
    if out is "" then
      return "No subtasks"
    end if
    return out
  on error
    return "No subtasks"
  end try
end tell`, bundleID, scriptResolveTaskByName(taskName))
}

func scriptAddSubtask(bundleID, taskName, subtaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
%s  try
    set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, scriptResolveTaskByName(taskName), escapeApple(subtaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
  on error
    error "Cannot add a subtask to this item."
  end try
end tell`
	return script
}

func scriptFindSubtask(bundleID, taskName, subtaskName string, index int) string {
	taskName = strings.TrimSpace(taskName)
	subtaskName = strings.TrimSpace(subtaskName)
	var target string
	if index > 0 {
		target = fmt.Sprintf("item %d of to dos of t", index)
	} else {
		target = fmt.Sprintf(`first to do of to dos of t whose name is "%s"`, escapeApple(subtaskName))
	}
	return fmt.Sprintf(`tell application id "%s"
%s  try
    set s to %s
  on error
    error "No subtask found on this item."
  end try
`, bundleID, scriptResolveTaskByName(taskName), target)
}

func scriptShowTask(bundleID, taskName string, withSubtasks bool) string {
	subtasksBlock := "false"
	if withSubtasks {
		subtasksBlock = "true"
	}
	return fmt.Sprintf(`tell application id "%s"
%s  set out to "ID: " & (id of t)
  set out to out & linefeed & "Name: " & (name of t)
  set out to out & linefeed & "Type: " & (class of t as string)
  set out to out & linefeed & "Statut: " & (status of t as string)
  if due date of t is not missing value then
    set out to out & linefeed & "Due: " & (due date of t as string)
  else
    set out to out & linefeed & "Due: "
  end if
  if completion date of t is not missing value then
    set out to out & linefeed & "Completed on: " & (completion date of t as string)
  else
    set out to out & linefeed & "Completed on: "
  end if
  if creation date of t is not missing value then
    set out to out & linefeed & "Created on: " & (creation date of t as string)
  else
    set out to out & linefeed & "Created on: "
  end if
  set tagText to ""
  try
    set taskTags to tag names of t
    repeat with i from 1 to count taskTags
      set tagLine to item i of taskTags
      if tagText is "" then
        set tagText to tagLine
      else
        set tagText to tagText & ", " & tagLine
      end if
    end repeat
  end try
  set out to out & linefeed & "Tags: " & tagText
  if notes of t is missing value then
    set out to out & linefeed & "Notes: "
  else
    set out to out & linefeed & "Notes: " & (notes of t)
  end if
  if %s then
    try
      set subtasks to to dos of t
      set subtaskLines to "No subtasks"
      if (count subtasks) > 0 then
        set subtaskLines to ""
        repeat with i from 1 to count subtasks
          set s to item i of subtasks
          set lineItem to (i as string) & ". " & (name of s) & " [" & (status of s as string) & "]"
          if (notes of s is not missing value) and (notes of s is not "") then
            set lineItem to lineItem & " | " & (notes of s)
          end if
          if subtaskLines is "" then
            set subtaskLines to lineItem
          else
            set subtaskLines to subtaskLines & linefeed & lineItem
          end if
        end repeat
      end if
      set out to out & linefeed & "Subtasks:" & linefeed & subtaskLines
    on error
      set out to out & linefeed & "Subtasks: not supported"
    end try
  end if
  return out
end tell`, bundleID, scriptResolveTaskByName(taskName), subtasksBlock)
}

func scriptEditSubtask(bundleID, taskName, subtaskName string, index int, newName, notes string) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	if newName != "" {
		script += fmt.Sprintf(`  set name of s to "%s"
`, escapeApple(newName))
	}
	if notes != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
end tell`
	return script
}

func scriptDeleteSubtask(bundleID, taskName, subtaskName string, index int) string {
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += `  delete s
  return "ok"
end tell`
	return script
}

func scriptSetSubtaskStatus(bundleID, taskName, subtaskName string, index int, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	script := scriptFindSubtask(bundleID, taskName, subtaskName, index)
	script += fmt.Sprintf(`  set status of s to %s
  return id of s
end tell`, state)
	return script
}

func scriptDelete(bundleID, kind, name string) (string, error) {
	var subject string
	switch kind {
	case "task":
		subject = "«class tstk»"
	case "project":
		subject = "project"
	case "list":
		subject = "list"
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
	return fmt.Sprintf(`tell application id "%s"
  delete first %s whose name is "%s"
end tell`, bundleID, subject, escapeApple(name)), nil
}

func scriptCompleteTask(bundleID, name string, done bool) string {
	state := "open"
	if done {
		state = "completed"
	}
	return fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
  set status of t to %s
  return id of t
end tell`, bundleID, escapeApple(name), state)
}

type backupManager struct {
	dataDir string
}

func newBackupManager(dataDir string) *backupManager {
	return &backupManager{dataDir: dataDir}
}

func (bm *backupManager) Create(ctx context.Context) ([]string, error) {
	_ = ctx
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	ts := time.Now().Format(backupTSFormat)
	var created []string
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		src := filepath.Join(bm.dataDir, base)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(dir, base+"."+ts+".bak")
		if err := copyFile(src, dst); err != nil {
			return nil, err
		}
		created = append(created, dst)
	}
	if len(created) == 0 {
		return nil, errors.New("no backupable database file found")
	}
	if err := bm.prune(ctx, maxBackupsToKeep); err != nil {
		return nil, fmt.Errorf("backup created but retention failed: %w", err)
	}
	sort.Strings(created)
	return created, nil
}

func (bm *backupManager) Latest(ctx context.Context) (string, error) {
	_ = ctx
	candidates, err := bm.allTimestamps()
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", errors.New("no backup available")
	}
	return candidates[0], nil
}

func (bm *backupManager) FilesForTimestamp(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	var paths []string
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		candidate := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no file for timestamp %s", ts)
	}
	return paths, nil
}

func (bm *backupManager) Restore(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	files, err := bm.FilesForTimestamp(ctx, ts)
	if err != nil {
		return nil, err
	}
	for _, src := range files {
		if err := bm.RestoreFile(ctx, src); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (bm *backupManager) RestoreFile(ctx context.Context, path string) error {
	_ = ctx
	base := filepath.Base(path)
	var baseTarget string
	if strings.HasPrefix(base, "main.sqlite.") {
		baseTarget = "main.sqlite"
	} else if strings.HasPrefix(base, "main.sqlite-shm.") {
		baseTarget = "main.sqlite-shm"
	} else if strings.HasPrefix(base, "main.sqlite-wal.") {
		baseTarget = "main.sqlite-wal"
	} else {
		return fmt.Errorf("nom de backup invalide: %s", base)
	}
	dst := filepath.Join(bm.dataDir, baseTarget)
	return copyFile(path, dst)
}

func (bm *backupManager) prune(ctx context.Context, keep int) error {
	_ = ctx
	if keep <= 0 {
		return nil
	}
	timestamps, err := bm.allTimestamps()
	if err != nil {
		return err
	}
	if len(timestamps) <= keep {
		return nil
	}
	for _, ts := range timestamps[keep:] {
		for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
			target := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func (bm *backupManager) allTimestamps() ([]string, error) {
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tsSet := map[string]struct{}{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ts := extractTimestamp(e.Name())
		if ts != "" {
			tsSet[ts] = struct{}{}
		}
	}
	var ts []string
	for k := range tsSet {
		ts = append(ts, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ts)))
	return ts, nil
}

func (bm *backupManager) backupPath() string {
	return filepath.Join(bm.dataDir, backupDirName)
}

func (bm *backupManager) ensureBackupDir() (string, error) {
	path := bm.backupPath()
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func parseToAppleDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	t, err := parseDate(value)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

func parseDate(v string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", v)
}

func inferTimestamp(file string) string {
	base := filepath.Base(file)
	candidates := []string{
		"main.sqlite.",
		"main.sqlite-shm.",
		"main.sqlite-wal.",
	}
	for _, p := range candidates {
		if strings.HasPrefix(base, p) && strings.HasSuffix(base, ".bak") {
			return strings.TrimSuffix(strings.TrimPrefix(base, p), ".bak")
		}
	}
	return ""
}

func extractTimestamp(file string) string {
	base := filepath.Base(file)
	return inferTimestamp(base)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func resolveDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultDataPathRel), nil
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func escapeApple(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}
