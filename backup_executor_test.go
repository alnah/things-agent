package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestBackupExecutor(bm *backupManager, app appController, semantic func(context.Context) (backupSemanticSnapshot, error), state func(context.Context) (thingsStateSnapshot, error)) *backupExecutor {
	runtime := &restoreExecutor{
		backups:             bm,
		bundleID:            defaultBundleID,
		app:                 app,
		sleep:               func(time.Duration) {},
		pollInterval:        time.Millisecond,
		stopTimeout:         time.Second,
		stabilityTimeout:    time.Second,
		stablePasses:        2,
		launchTimeout:       time.Second,
		semanticTimeout:     time.Second,
		fullSemanticTimeout: time.Second,
		captureFileState: func(string) ([]liveFileState, error) {
			return []liveFileState{
				{Name: "main.sqlite", Size: 1, ModTime: 1},
				{Name: "main.sqlite-shm", Size: 1, ModTime: 1},
				{Name: "main.sqlite-wal", Size: 1, ModTime: 1},
			}, nil
		},
		semanticCheck:     semantic,
		fullSemanticCheck: semantic,
	}
	return &backupExecutor{runtime: runtime, stateCheck: state}
}

func TestBackupExecutorQuiescesRunningAppAndWritesManifest(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	expected := testSemanticSnapshot(2, 1, 3)
	expected.TaskRefs = []string{"task-1", "task-2", "task-3"}

	app := &fakeAppController{running: []bool{true, true, false}}
	exec := newTestBackupExecutor(bm, app, func(context.Context) (backupSemanticSnapshot, error) {
		return expected, nil
	}, func(context.Context) (thingsStateSnapshot, error) {
		return thingsStateSnapshot{SchemaVersion: 1, Areas: []thingsStateArea{{ID: "area-1", Name: "Area A"}}}, nil
	})

	created, err := exec.Create(context.Background())
	if err != nil {
		t.Fatalf("safe backup failed: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected trio backup, got %d", len(created))
	}

	ts := inferTimestamp(created[0])
	got, err := bm.loadSemanticSnapshot(ts)
	if err != nil {
		t.Fatalf("loadSemanticSnapshot failed: %v", err)
	}
	if got.TasksHash != expected.TasksHash || len(got.TaskRefs) != len(expected.TaskRefs) {
		t.Fatalf("unexpected semantic manifest: %#v", got)
	}
	state, err := bm.loadStateSnapshot(ts)
	if err != nil {
		t.Fatalf("loadStateSnapshot failed: %v", err)
	}
	if len(state.Areas) != 1 || state.Areas[0].Name != "Area A" {
		t.Fatalf("unexpected state snapshot: %#v", state)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 0 {
		t.Fatalf("expected quiesce without explicit activate, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestBackupExecutorTemporarilyLaunchesWhenAppWasNotRunning(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	expected := testSemanticSnapshot(1, 0, 1)
	expected.TaskRefs = []string{"task-1"}

	app := &fakeAppController{running: []bool{false, false}}
	exec := newTestBackupExecutor(bm, app, func(context.Context) (backupSemanticSnapshot, error) {
		return expected, nil
	}, func(context.Context) (thingsStateSnapshot, error) {
		return thingsStateSnapshot{SchemaVersion: 1}, nil
	})

	created, err := exec.Create(context.Background())
	if err != nil {
		t.Fatalf("safe backup failed: %v", err)
	}
	ts := inferTimestamp(created[0])
	got, err := bm.loadSemanticSnapshot(ts)
	if err != nil {
		t.Fatalf("loadSemanticSnapshot failed: %v", err)
	}
	if got.TasksCount != 1 || len(got.TaskRefs) != 1 {
		t.Fatalf("unexpected semantic manifest: %#v", got)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 0 {
		t.Fatalf("expected temporary semantic close-only lifecycle, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestBackupExecutorReopensRunningAppWhenBackupCopyFails(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	bm.copyFn = func(string, string) error {
		return errors.New("copy boom")
	}

	app := &fakeAppController{running: []bool{true, true, false}}
	exec := newTestBackupExecutor(bm, app, func(context.Context) (backupSemanticSnapshot, error) {
		return backupSemanticSnapshot{}, nil
	}, func(context.Context) (thingsStateSnapshot, error) {
		return thingsStateSnapshot{SchemaVersion: 1}, nil
	})

	_, err := exec.Create(context.Background())
	if err == nil || !strings.Contains(err.Error(), "copy boom") {
		t.Fatalf("expected copy error, got %v", err)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected reopen after failure, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestBackupExecutorFallsBackToLightSemanticManifestOnTimeout(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	app := &fakeAppController{running: []bool{false, false}}
	exec := newTestBackupExecutor(bm, app, func(ctx context.Context) (backupSemanticSnapshot, error) {
		<-ctx.Done()
		return backupSemanticSnapshot{}, ctx.Err()
	}, func(context.Context) (thingsStateSnapshot, error) {
		return thingsStateSnapshot{SchemaVersion: 1}, nil
	})
	exec.runtime.semanticTimeout = time.Nanosecond
	exec.healthCheck = func(context.Context) (backupSemanticSnapshot, error) {
		return backupSemanticSnapshot{ListsCount: 3, ProjectsCount: 2, TasksCount: 9}, nil
	}

	created, err := exec.Create(context.Background())
	if err != nil {
		t.Fatalf("safe backup with fallback failed: %v", err)
	}

	ts := inferTimestamp(created[0])
	got, err := bm.loadSemanticSnapshot(ts)
	if err != nil {
		t.Fatalf("loadSemanticSnapshot failed: %v", err)
	}
	if got.ListsCount != 3 || got.ProjectsCount != 2 || got.TasksCount != 9 {
		t.Fatalf("unexpected fallback semantic manifest: %#v", got)
	}
	if got.TasksHash != "" || len(got.TaskRefs) != 0 {
		t.Fatalf("expected count-only fallback manifest, got %#v", got)
	}
}
