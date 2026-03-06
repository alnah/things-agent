package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	backupDirName      = ".agent-backups"
	backupTSFormat     = "2006-01-02:15-04-05"
	defaultListName    = "À classer"
	cliVersion         = "0.2.1"
)

var config = struct {
	bundleID string
	dataDir  string
}{
	bundleID: envOrDefault("THINGS_BUNDLE_ID", defaultBundleID),
}

type runtimeConfig struct {
	bundleID string
	dataDir  string
	runner   *runner
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "things",
		SilenceErrors: false,
		SilenceUsage:  true,
		Short:         "CLI Things via AppleScript (aucun accès direct à la base)",
		Long: `Ce CLI pilote Things via AppleScript uniquement.
Il crée une sauvegarde timestampée au format YYYY-MM-DD:hh-mm-ss avant chaque action
qui modifie des données.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&config.bundleID, "bundle-id", envOrDefault("THINGS_BUNDLE_ID", defaultBundleID), "Bundle id de l'application Things")
	root.PersistentFlags().StringVar(&config.dataDir, "data-dir", envOrDefault("THINGS_DATA_DIR", ""), "Chemin de la base Things")

	root.AddCommand(
		newBackupCmd(),
		newRestoreCmd(),
		newListsCmd(),
		newProjectsCmd(),
		newTasksCmd(),
		newSearchCmd(),
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
		newListSubtasksCmd(),
		newAddSubtaskCmd(),
		newEditSubtaskCmd(),
		newDeleteSubtaskCmd(),
		newCompleteSubtaskCmd(),
		newUncompleteSubtaskCmd(),
		&cobra.Command{
			Use:   "version",
			Short: "Afficher la version",
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
		bundleID: config.bundleID,
		dataDir:  dataDir,
		runner:   r,
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

func newBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Créer une sauvegarde de la base Things",
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
		Short: "Restaurer un backup (dernier par défaut)",
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
		Short: "Lister les projets",
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
		Short: "Lister les tâches (optionnellement filtrées)",
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
	cmd.Flags().StringVar(&query, "query", "", "Filtre nom / notes")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var listName, query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Rechercher tâches",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(query) == "" {
				return errors.New("--query est requis")
			}
			return runResult(ctx, cfg, scriptSearch(cfg.bundleID, listName, query))
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Texte recherché")
	cmd.Flags().StringVar(&listName, "list", "", "Limiter au domaine")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func newAddTaskCmd() *cobra.Command {
	var name, notes, tags, listName, due, subtasks string
	cmd := &cobra.Command{
		Use:   "add-task",
		Short: "Ajouter une tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name est requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			dueDate, err := parseToAppleDate(due)
			if err != nil {
				return err
			}
			subtasksList := parseCSVList(subtasks)
			return runResult(ctx, cfg, scriptAddTask(cfg.bundleID, strings.TrimSpace(listName), name, notes, tags, dueDate, subtasksList))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom de la tâche")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags (séparés par virgules)")
	cmd.Flags().StringVar(&listName, "list", defaultListName, "Domaine")
	cmd.Flags().StringVar(&due, "due", "", "Date d'échéance (YYYY-MM-DD [HH:mm[:ss]])")
	cmd.Flags().StringVar(&subtasks, "subtasks", "", "Sous-tâches (nom1, nom2, ...)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddProjectCmd() *cobra.Command {
	var name, notes, listName string
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "Ajouter un projet",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name est requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddProject(cfg.bundleID, strings.TrimSpace(listName), name, notes))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom du projet")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes")
	cmd.Flags().StringVar(&listName, "list", defaultListName, "Domaine")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newAddListCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add-list",
		Short: "Ajouter un domaine",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name est requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  make new list with properties {name:"%s"}
  return "ok"
end tell`, cfg.bundleID, escapeApple(name))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom du domaine")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditTaskCmd() *cobra.Command {
	var sourceName, newName, notes, tags, moveTo, due, completion, creation, cancel string
	cmd := &cobra.Command{
		Use:   "edit-task",
		Short: "Modifier une tâche (via son nom)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name est requis")
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
	cmd.Flags().StringVar(&sourceName, "name", "", "Nom de la tâche à modifier")
	cmd.Flags().StringVar(&newName, "new-name", "", "Nouveau nom")
	cmd.Flags().StringVar(&notes, "notes", "", "Nouvelles notes")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags")
	cmd.Flags().StringVar(&moveTo, "move-to", "", "Nouveau domaine")
	cmd.Flags().StringVar(&due, "due", "", "Nouvelle date d'échéance")
	cmd.Flags().StringVar(&completion, "completion", "", "Date de completion")
	cmd.Flags().StringVar(&creation, "creation", "", "Date de création")
	cmd.Flags().StringVar(&cancel, "cancel", "", "Date d'annulation")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditProjectCmd() *cobra.Command {
	var sourceName, newName, notes string
	cmd := &cobra.Command{
		Use:   "edit-project",
		Short: "Modifier un projet (via son nom)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name est requis")
			}
			if strings.TrimSpace(newName) == "" && strings.TrimSpace(notes) == "" {
				return errors.New("spécifie --new-name et/ou --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditProject(cfg.bundleID, sourceName, newName, notes))
		},
	}
	cmd.Flags().StringVar(&sourceName, "name", "", "Nom du projet")
	cmd.Flags().StringVar(&newName, "new-name", "", "Nouveau nom")
	cmd.Flags().StringVar(&notes, "notes", "", "Nouvelles notes")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditListCmd() *cobra.Command {
	var sourceName, newName string
	cmd := &cobra.Command{
		Use:   "edit-list",
		Short: "Renommer un domaine",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(sourceName) == "" {
				return errors.New("--name est requis")
			}
			if strings.TrimSpace(newName) == "" {
				return errors.New("--new-name est requis")
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
	cmd.Flags().StringVar(&sourceName, "name", "", "Nom du domaine")
	cmd.Flags().StringVar(&newName, "new-name", "", "Nouveau nom")
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
		Short: "Supprimer un élément",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(target) == "" {
				return errors.New("--name est requis")
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
	cmd.Flags().StringVar(&target, "name", "", "Nom de l'élément")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newCompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "complete-task",
		Short: "Marquer une tâche comme réalisée",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name est requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, true))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom de la tâche")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUncompleteTaskCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "uncomplete-task",
		Short: "Annuler la réalisation d'une tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" {
				return errors.New("--name est requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptCompleteTask(cfg.bundleID, name, false))
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom de la tâche")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSetTagsCmd() *cobra.Command {
	var name, tags string
	cmd := &cobra.Command{
		Use:   "set-tags",
		Short: "Définir les tags d'une tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(tags) == "" {
				return errors.New("--name et --tags sont requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			script := fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
  set tag names of t to "%s"
  return id of t
end tell`, cfg.bundleID, escapeApple(name), escapeApple(tags))
			return runResult(ctx, cfg, script)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Nom de la tâche")
	cmd.Flags().StringVar(&tags, "tags", "", "Tags séparés par virgules")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func newListSubtasksCmd() *cobra.Command {
	var taskName string
	cmd := &cobra.Command{
		Use:   "list-subtasks",
		Short: "Lister les sous-tâches d'une tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(taskName) == "" {
				return errors.New("--task est requis")
			}
			return runResult(ctx, cfg, scriptListSubtasks(cfg.bundleID, taskName))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newAddSubtaskCmd() *cobra.Command {
	var taskName, subtaskName, notes string
	cmd := &cobra.Command{
		Use:   "add-subtask",
		Short: "Ajouter une sous-tâche à une tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" || subtaskName == "" {
				return errors.New("--task et --name sont requis")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptAddSubtask(cfg.bundleID, taskName, subtaskName, notes))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Nom de la sous-tâche")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes de la sous-tâche")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newEditSubtaskCmd() *cobra.Command {
	var taskName, subtaskName, newName, notes string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "edit-subtask",
		Short: "Modifier une sous-tâche",
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
				return errors.New("--task est requis")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("fournir --index (>=1) ou --name")
			}
			if newName == "" && notes == "" {
				return errors.New("fournir --new-name et/ou --notes")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptEditSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex, newName, notes))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Nom de la sous-tâche à cibler")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Index de la sous-tâche à cibler (1-based)")
	cmd.Flags().StringVar(&newName, "new-name", "", "Nouveau nom")
	cmd.Flags().StringVar(&notes, "notes", "", "Nouvelles notes")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newDeleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "delete-subtask",
		Short: "Supprimer une sous-tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task est requis")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("fournir --index (>=1) ou --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptDeleteSubtask(cfg.bundleID, taskName, subtaskName, subtaskIndex))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Nom de la sous-tâche")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Index de la sous-tâche (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newCompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "complete-subtask",
		Short: "Marquer une sous-tâche comme réalisée",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task est requis")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("fournir --index (>=1) ou --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, true))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Nom de la sous-tâche")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Index de la sous-tâche (1-based)")
	_ = cmd.MarkFlagRequired("task")
	return cmd
}

func newUncompleteSubtaskCmd() *cobra.Command {
	var taskName, subtaskName string
	var subtaskIndex int
	cmd := &cobra.Command{
		Use:   "uncomplete-subtask",
		Short: "Annuler la réalisation d'une sous-tâche",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := resolveRuntimeConfig(ctx)
			if err != nil {
				return err
			}
			taskName = strings.TrimSpace(taskName)
			subtaskName = strings.TrimSpace(subtaskName)
			if taskName == "" {
				return errors.New("--task est requis")
			}
			if subtaskIndex <= 0 && subtaskName == "" {
				return errors.New("fournir --index (>=1) ou --name")
			}
			if err := backupIfNeeded(ctx, cfg); err != nil {
				return err
			}
			return runResult(ctx, cfg, scriptSetSubtaskStatus(cfg.bundleID, taskName, subtaskName, subtaskIndex, false))
		},
	}
	cmd.Flags().StringVar(&taskName, "task", "", "Nom de la tâche parent")
	cmd.Flags().StringVar(&subtaskName, "name", "", "Nom de la sous-tâche")
	cmd.Flags().IntVar(&subtaskIndex, "index", 0, "Index de la sous-tâche (1-based)")
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
		return fmt.Errorf("Things Mac introuvable (%s): %w", r.bundleID, err)
	}
	return nil
}

func scriptAllLists(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  get name of lists
end tell`, bundleID)
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

func scriptAddTask(bundleID, listName, name, notes, tags, due string, subtasks []string) string {
	if strings.TrimSpace(listName) == "" {
		listName = defaultListName
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
  set t to make new «class tstk» at end of tasks of targetList with properties {%s}
`, bundleID, escapeApple(listName), strings.Join(parts, ", "))
	if strings.TrimSpace(due) != "" {
		script += fmt.Sprintf(`  set due date of t to date "%s"
`, due)
	}
	if len(subtasks) > 0 {
		for _, sub := range subtasks {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			script += fmt.Sprintf(`  make new to do at end of to dos of t with properties {name:"%s"}
`, escapeApple(sub))
		}
	}
	script += `  return id of t
end tell`
	return script
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

func scriptAddProject(bundleID, listName, name, notes string) string {
	if strings.TrimSpace(listName) == "" {
		listName = defaultListName
	}
	script := fmt.Sprintf(`tell application id "%s"
  set targetList to first list whose name is "%s"
  set p to make new project at end of tasks of targetList with properties {name:"%s"}
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
		return "", errors.New("source name required")
	}
	script := fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
`, bundleID, escapeApple(source))
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
		script += fmt.Sprintf(`  move t to end of tasks of (first list whose name is "%s")
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

func scriptListSubtasks(bundleID, taskName string) string {
	taskName = strings.TrimSpace(taskName)
	return fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
  set subtasks to to dos of t
  set out to ""
  repeat with i from 1 to count subtasks
    set s to item i of subtasks
    set outLine to (i as string) & ". " & name of s
    if (notes of s is not missing value) and (notes of s is not "") then
      set outLine to outLine & " | " & notes of s
    end if
    if out is "" then
      set out to outLine
    else
      set out to out & linefeed & outLine
    end if
  end repeat
  if out is "" then
    return "Aucune sous-tâche"
  end if
  return out
end tell`, bundleID, escapeApple(taskName))
}

func scriptAddSubtask(bundleID, taskName, subtaskName, notes string) string {
	script := fmt.Sprintf(`tell application id "%s"
  set t to first «class tstk» whose name is "%s"
  set s to make new to do at end of to dos of t with properties {name:"%s"}
`, bundleID, escapeApple(taskName), escapeApple(subtaskName))
	if strings.TrimSpace(notes) != "" {
		script += fmt.Sprintf(`  set notes of s to "%s"
`, escapeApple(notes))
	}
	script += `  return id of s
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
  set t to first «class tstk» whose name is "%s"
  set s to %s
`, bundleID, escapeApple(taskName), target)
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
		return "", fmt.Errorf("kind inconnu: %s", kind)
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
		return nil, errors.New("aucun fichier de base backupable trouvé")
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
		return "", errors.New("aucun backup disponible")
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
		return nil, fmt.Errorf("aucun fichier pour le timestamp %s", ts)
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
	return t.Format("January 2, 2006 at 15:04:05"), nil
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
	return time.Time{}, fmt.Errorf("format de date non reconnu: %s", v)
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
