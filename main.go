package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultBundleID   = "com.culturedcode.ThingsMac"
	backupDirName     = "Backups"
	backupTSFormat    = "2006-01-02:15-04-05"
	maxBackupsToKeep  = 50
	thingsDataPattern = "Library/Group Containers/*.com.culturedcode.ThingsMac/ThingsData-*/Things Database.thingsdatabase"
)

var cliVersion = "dev"

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
	runner    scriptRunner
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "things-agent",
		SilenceErrors: false,
		SilenceUsage:  true,
		Short:         "Things CLI via AppleScript with a safe restore harness",
		Long: `This CLI controls Things through AppleScript for normal reads and writes.
Restore uses an internal SQLite-backed package-swap harness.
It creates a timestamped backup in YYYY-MM-DD:hh-mm-ss format
before destructive delete actions, explicit backup commands, and restore.
Restore creates a pre-restore backup,
quiesces Things, verifies restored files, and rolls back on failure.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	bundleIDDefault := strings.TrimSpace(config.bundleID)
	if bundleIDDefault == "" {
		bundleIDDefault = envOrDefault("THINGS_BUNDLE_ID", defaultBundleID)
	}
	dataDirDefault := strings.TrimSpace(config.dataDir)
	if dataDirDefault == "" {
		dataDirDefault = envOrDefault("THINGS_DATA_DIR", "")
	}
	authTokenDefault := strings.TrimSpace(config.authToken)

	root.PersistentFlags().StringVar(&config.bundleID, "bundle-id", bundleIDDefault, "Things app bundle id")
	root.PersistentFlags().StringVar(&config.dataDir, "data-dir", dataDirDefault, "Things database path")
	root.PersistentFlags().StringVar(&config.authToken, "auth-token", authTokenDefault, "Things URL Scheme auth token (Settings > General)")

	root.AddCommand(
		newBackupCmd(),
		newRestoreCmd(),
		newDateCmd(),
		newSessionStartCmd(),
		newURLCmd(),
		newAreasCmd(),
		newListsCmd(),
		newProjectsCmd(),
		newTagsCmd(),
		newTasksCmd(),
		newSearchCmd(),
		newShowTaskCmd(),
		newAddTaskCmd(),
		newAddProjectCmd(),
		newAddAreaCmd(),
		newEditTaskCmd(),
		newEditProjectCmd(),
		newEditAreaCmd(),
		newDeleteTaskCmd(),
		newDeleteProjectCmd(),
		newDeleteAreaCmd(),
		newCompleteTaskCmd(),
		newUncompleteTaskCmd(),
		newSetTagsCmd(),
		newSetTaskTagsCmd(),
		newAddTaskTagsCmd(),
		newRemoveTaskTagsCmd(),
		newSetTaskNotesCmd(),
		newAppendTaskNotesCmd(),
		newSetTaskDateCmd(),
		newAddChecklistItemCmd(),
		newListChildTasksCmd(),
		newAddChildTaskCmd(),
		newEditChildTaskCmd(),
		newDeleteChildTaskCmd(),
		newCompleteChildTaskCmd(),
		newUncompleteChildTaskCmd(),
		newMoveTaskCmd(),
		newMoveProjectCmd(),
		newReorderProjectItemsCmd(),
		newReorderAreaItemsCmd(),
		&cobra.Command{
			Use:   "version",
			Short: "Show CLI version",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Fprintln(cmd.OutOrStdout(), "things", effectiveCLIVersion())
			},
		},
	)

	return root
}
