//go:build integration

package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIntegrationTagsSearchUsesMockRunner(t *testing.T) {
	fr := &fakeRunner{output: "work, urgent"}
	setupTestRuntime(t, t.TempDir(), fr)

	root := newRootCmd()
	root.SetArgs([]string{"tags", "search", "--query", "wo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("root execute failed: %v", err)
	}

	scripts := fr.allScripts()
	if len(scripts) != 1 {
		t.Fatalf("expected one runner call, got %d", len(scripts))
	}
	if !strings.Contains(scripts[0], "every tag whose name contains") {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}

func TestIntegrationAddTaskUsesMockRunnerWithExplicitArea(t *testing.T) {
	fr := &fakeRunner{output: "task-id-1"}
	setupTestRuntimeWithDB(t, fr)

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"add-task", "--name", "integration-task", "--area", "Inbox"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("root execute failed: %v", err)
	}
	if !strings.Contains(stdout, "task-id-1") {
		t.Fatalf("expected created task id on stdout, got %q", stdout)
	}

	scripts := fr.allScripts()
	if len(scripts) == 0 {
		t.Fatal("expected mocked runner to be called")
	}
	if !strings.Contains(scripts[0], `set targetList to first list whose name is "Inbox"`) {
		t.Fatalf("unexpected script content: %s", scripts[0])
	}
}

func TestIntegrationRestoreDryRunReturnsStructuredJournal(t *testing.T) {
	tmp := setupTestRuntimeWithDB(t, &fakeRunner{})
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	runner := runnerFunc(func(_ context.Context, script string) (string, error) {
		switch {
		case strings.Contains(script, "return running"):
			return "false", nil
		case strings.Contains(script, "restore semantic verify"):
			return "L\t0\nP\t0\nT\t0", nil
		case strings.Contains(script, "count of lists") && strings.Contains(script, "count of projects") && strings.Contains(script, "count of to dos"):
			return "L\t0\nP\t0\nT\t0", nil
		default:
			return "ok", nil
		}
	})
	setupTestRuntime(t, tmp, runner)

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"restore", "--timestamp", targetTS, "--dry-run", "--json"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("restore dry-run failed: %v", err)
	}

	var journal map[string]any
	if err := json.Unmarshal([]byte(stdout), &journal); err != nil {
		t.Fatalf("decode restore journal: %v\nstdout=%q", err, stdout)
	}
	if journal["outcome"] != "dry-run" {
		t.Fatalf("expected dry-run outcome, got %#v", journal["outcome"])
	}
	if journal["timestamp"] != targetTS {
		t.Fatalf("expected timestamp %q, got %#v", targetTS, journal["timestamp"])
	}
	preflight, ok := journal["preflight"].(map[string]any)
	if !ok || preflight["ok"] != true {
		t.Fatalf("expected successful preflight report, got %#v", journal["preflight"])
	}
}

func TestIntegrationRestoreListJSONIncludesBackupKinds(t *testing.T) {
	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)

	root := newRootCmd()
	root.SetArgs([]string{"backup"})
	if err := root.Execute(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	root = newRootCmd()
	root.SetArgs([]string{"session-start"})
	if err := root.Execute(); err != nil {
		t.Fatalf("session-start failed: %v", err)
	}

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"restore", "list", "--json"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("restore list failed: %v", err)
	}

	var snapshots []map[string]any
	if err := json.Unmarshal([]byte(stdout), &snapshots); err != nil {
		t.Fatalf("decode restore list json: %v\nstdout=%q", err, stdout)
	}
	if len(snapshots) < 2 {
		t.Fatalf("expected at least two snapshots, got %#v", snapshots)
	}

	foundKinds := map[string]bool{}
	for _, snapshot := range snapshots {
		kind, _ := snapshot["kind"].(string)
		if kind != "" {
			foundKinds[kind] = true
		}
	}
	if !foundKinds[string(backupKindExplicit)] || !foundKinds[string(backupKindSession)] {
		t.Fatalf("expected explicit and session backup kinds, got %#v", snapshots)
	}

	manager := newBackupManager(tmp)
	latestTS, err := manager.Latest(context.Background())
	if err != nil {
		t.Fatalf("latest snapshot failed: %v", err)
	}
	latestMeta, err := manager.loadBackupMetadata(latestTS)
	if err != nil {
		t.Fatalf("loadBackupMetadata failed: %v", err)
	}
	if latestMeta.Kind != backupKindSession || latestMeta.SourceCommand != "session-start" {
		t.Fatalf("unexpected latest backup metadata: %#v", latestMeta)
	}
}

func TestIntegrationRestoreCreatesSafetyBackupMetadata(t *testing.T) {
	tmp := setupTestRuntimeWithDB(t, &fakeRunner{})
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false}}
	exec := newTestRestoreExecutor(bm, app)

	journal, err := exec.Execute(context.Background(), targetTS, false)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if journal.PreRestoreBackup == nil || journal.PreRestoreBackup.Kind != string(backupKindSafety) {
		t.Fatalf("expected safety pre-restore backup kind, got %#v", journal.PreRestoreBackup)
	}

	snapshots, err := bm.List(context.Background())
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	var found bool
	for _, snapshot := range snapshots {
		if snapshot.Kind == backupKindSafety && snapshot.SourceCommand == "restore" && snapshot.Reason == "pre-restore rollback checkpoint" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected restore-created safety backup in snapshot list, got %#v", snapshots)
	}
}
