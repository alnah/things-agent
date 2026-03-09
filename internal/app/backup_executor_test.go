package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestBackupExecutor(bm *backupManager, app appController) *backupExecutor {
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
	}
	return &backupExecutor{runtime: runtime}
}

func TestBackupExecutorQuiescesRunningApp(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	app := &fakeAppController{running: []bool{true, true, false}}
	exec := newTestBackupExecutor(bm, app)

	created, err := exec.Create(context.Background())
	if err != nil {
		t.Fatalf("safe backup failed: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected trio backup, got %d", len(created))
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected backup to restore app open state, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestBackupExecutorWaitsForSettleDelayBeforeQuiesceWhenAppRunning(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	app := &fakeAppController{running: []bool{true, true, false}}
	exec := newTestBackupExecutor(bm, app)
	var slept []time.Duration
	exec.settleDelay = 5 * time.Second
	exec.runtime.sleep = func(d time.Duration) {
		slept = append(slept, d)
	}

	if _, err := exec.Create(context.Background()); err != nil {
		t.Fatalf("backup create failed: %v", err)
	}
	if len(slept) == 0 || slept[0] != exec.settleDelay {
		t.Fatalf("expected settle delay sleep first, got %#v", slept)
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
	exec := newTestBackupExecutor(bm, app)

	_, err := exec.Create(context.Background())
	if err == nil || !strings.Contains(err.Error(), "copy boom") {
		t.Fatalf("expected copy error, got %v", err)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected reopen after failure, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestBackupExecutorDoesNotOpenAppWhenInitiallyClosed(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "live")

	bm := newBackupManager(tmp)
	app := &fakeAppController{running: []bool{false}}
	exec := newTestBackupExecutor(bm, app)

	_, err := exec.Create(context.Background())
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 0 || activateCalls != 0 {
		t.Fatalf("expected closed app to stay closed, got quit=%d activate=%d", quitCalls, activateCalls)
	}
}
