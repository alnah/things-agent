package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCmdBasicHelpers(t *testing.T) {
	params := map[string]string{}
	setIfNotEmpty(params, "a", "x")
	setIfNotEmpty(params, "b", "  ")
	if params["a"] != "x" {
		t.Fatalf("setIfNotEmpty should set non-empty value")
	}
	if _, ok := params["b"]; ok {
		t.Fatal("setIfNotEmpty should skip empty value")
	}

	cmd := &cobra.Command{Use: "x"}
	cmd.Flags().String("name", "", "")
	cmd.Flags().Bool("done", false, "")
	cmd.Flags().Bool("done-false", true, "")
	if err := cmd.Flags().Set("name", "alpha"); err != nil {
		t.Fatalf("set name flag failed: %v", err)
	}
	if err := cmd.Flags().Set("done", "true"); err != nil {
		t.Fatalf("set done flag failed: %v", err)
	}

	setIfChanged(cmd, params, "name", "alpha")
	setBoolIfChanged(cmd, params, "done", true)
	if err := cmd.Flags().Set("done-false", "false"); err != nil {
		t.Fatalf("set done-false flag failed: %v", err)
	}
	setBoolIfChanged(cmd, params, "done-false", false)
	if params["name"] != "alpha" || params["done"] != "true" {
		t.Fatalf("unexpected params after setIfChanged/setBoolIfChanged: %#v", params)
	}
	if params["done-false"] != "false" {
		t.Fatalf("expected done-false=false param, got %#v", params)
	}
}

func TestBasicReadCommands(t *testing.T) {
	fr := &fakeRunner{output: "ok"}
	setupTestRuntime(t, t.TempDir(), fr)

	areas := newAreasCmd()
	if err := areas.Execute(); err != nil {
		t.Fatalf("areas failed: %v", err)
	}
	lists := newListsCmd()
	if err := lists.Execute(); err != nil {
		t.Fatalf("lists failed: %v", err)
	}
	projects := newProjectsCmd()
	if err := projects.Execute(); err != nil {
		t.Fatalf("projects failed: %v", err)
	}
	tasks := newTasksCmd()
	tasks.SetArgs([]string{"--list", "Inbox", "--query", "x"})
	if err := tasks.Execute(); err != nil {
		t.Fatalf("tasks failed: %v", err)
	}
	search := newSearchCmd()
	search.SetArgs([]string{"--query", "x", "--list", "Inbox"})
	if err := search.Execute(); err != nil {
		t.Fatalf("search failed: %v", err)
	}
}

func TestBackupRestoreSessionCommands(t *testing.T) {
	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)

	backup := newBackupCmd()
	if err := backup.Execute(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	session := newSessionStartCmd()
	if err := session.Execute(); err != nil {
		t.Fatalf("session-start failed: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmp, backupDirName))
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected backup files, err=%v count=%d", err, len(entries))
	}
	manager := newBackupManager(config.dataDir)
	sessionTS, err := manager.Latest(context.Background())
	if err != nil {
		t.Fatalf("latest snapshot failed after session-start: %v", err)
	}
	sessionMeta, err := manager.loadBackupMetadata(sessionTS)
	if err != nil {
		t.Fatalf("loadBackupMetadata failed: %v", err)
	}
	if sessionMeta.Kind != backupKindSession || sessionMeta.SourceCommand != "session-start" {
		t.Fatalf("unexpected session metadata: %#v", sessionMeta)
	}

	restore := newRestoreCmd()
	if err := restore.Execute(); err != nil {
		t.Fatalf("restore latest failed: %v", err)
	}

	restoreMissing := newRestoreCmd()
	restoreMissing.SetArgs([]string{"--timestamp", "missing-ts"})
	if err := restoreMissing.Execute(); err == nil {
		t.Fatal("expected restore error for missing timestamp/file")
	}

	restoreByTimestamp := newRestoreCmd()
	restoreByTimestamp.SetArgs([]string{"--timestamp", sessionTS})
	if err := restoreByTimestamp.Execute(); err != nil {
		t.Fatalf("restore by timestamp failed: %v", err)
	}
}

func TestRestoreLatestWithoutBackupReturnsError(t *testing.T) {
	fr := &fakeRunner{}
	tmp := t.TempDir()
	setupTestRuntime(t, tmp, fr)
	restore := newRestoreCmd()
	if err := restore.Execute(); err == nil {
		t.Fatal("expected restore latest error when no backups exist")
	}
}

func TestBackupCommandsFailWithoutDBFiles(t *testing.T) {
	fr := &fakeRunner{}
	setupTestRuntime(t, t.TempDir(), fr)

	backup := newBackupCmd()
	if err := backup.Execute(); err == nil {
		t.Fatal("expected backup failure without sqlite files")
	}

	session := newSessionStartCmd()
	if err := session.Execute(); err == nil {
		t.Fatal("expected session-start failure without sqlite files")
	}
}

func TestRestoreListAndVerifyCommands(t *testing.T) {
	fr := &fakeRunner{}
	setupTestRuntimeWithDB(t, fr)

	backup := newBackupCmd()
	if err := backup.Execute(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	list := newRestoreListCmd()
	if err := list.Execute(); err != nil {
		t.Fatalf("restore list failed: %v", err)
	}

	snapshotsCmd := newRestoreListCmd()
	snapshotsCmd.SetArgs([]string{"--json"})
	if err := snapshotsCmd.Execute(); err != nil {
		t.Fatalf("restore list --json failed: %v", err)
	}

	manager := newBackupManager(config.dataDir)
	ts, err := manager.Latest(context.Background())
	if err != nil {
		t.Fatalf("latest snapshot failed: %v", err)
	}

	verify := newRestoreVerifyCmd()
	verify.SetArgs([]string{"--timestamp", ts})
	if err := verify.Execute(); err != nil {
		t.Fatalf("restore verify failed: %v", err)
	}

	verifyJSON := newRestoreVerifyCmd()
	verifyJSON.SetArgs([]string{"--timestamp", ts, "--json"})
	if err := verifyJSON.Execute(); err != nil {
		t.Fatalf("restore verify --json failed: %v", err)
	}
}
