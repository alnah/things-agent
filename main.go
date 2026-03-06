package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultBundleID   = "com.culturedcode.ThingsMac"
	backupDirName     = "backups"
	backupTSFormat    = "2006-01-02:15-04-05"
	maxBackupsToKeep  = 50
	defaultListName   = "Inbox"
	cliVersion        = "0.3.0"
	thingsDataPattern = "Library/Group Containers/*.com.culturedcode.ThingsMac/ThingsData-*/Things Database.thingsdatabase"
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
		Short:         "Things CLI via AppleScript (no direct DB access)",
		Long: `This CLI controls Things through AppleScript only.
It creates a timestamped backup in YYYY-MM-DD:hh-mm-ss format
before each write action.`,
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
		newSessionStartCmd(),
		newURLCmd(),
		newListsCmd(),
		newProjectsCmd(),
		newTagsCmd(),
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
