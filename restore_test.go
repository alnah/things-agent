package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeAppController struct {
	mu           sync.Mutex
	running      []bool
	quitCalls    int
	activateCalls int
	quitErr      error
	activateErr  error
	runningErr   error
}

func (f *fakeAppController) IsRunning(_ context.Context, _ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.runningErr != nil {
		return false, f.runningErr
	}
	if len(f.running) == 0 {
		return false, nil
	}
	state := f.running[0]
	if len(f.running) > 1 {
		f.running = f.running[1:]
	}
	return state, nil
}

func (f *fakeAppController) Quit(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quitCalls++
	return f.quitErr
}

func (f *fakeAppController) Activate(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activateCalls++
	return f.activateErr
}

func (f *fakeAppController) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.quitCalls, f.activateCalls
}

func writeLiveDBSet(t *testing.T, dir string, suffix string) {
	t.Helper()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		payload := []byte(base + ":" + suffix)
		if err := os.WriteFile(filepath.Join(dir, base), payload, 0o644); err != nil {
			t.Fatalf("write %s: %v", base, err)
		}
	}
}

func readLiveDBFile(t *testing.T, dir, base string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, base))
	if err != nil {
		t.Fatalf("read %s: %v", base, err)
	}
	return string(data)
}

func TestRestoreExecutorRestoresAndReopensWhenAppWasRunning(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{true, true, false}}
	exec := &restoreExecutor{
		backups:      bm,
		bundleID:     defaultBundleID,
		app:          app,
		sleep:        func(time.Duration) {},
		pollInterval: time.Millisecond,
		stopTimeout:  time.Second,
	}

	restored, err := exec.Restore(context.Background(), targetTS)
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if len(restored) != 3 {
		t.Fatalf("expected restored trio, got %d", len(restored))
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:before" {
		t.Fatalf("expected restored main.sqlite, got %q", got)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 {
		t.Fatalf("expected one quit call, got %d", quitCalls)
	}
	if activateCalls != 1 {
		t.Fatalf("expected one activate call, got %d", activateCalls)
	}
}

func TestRestoreExecutorRollsBackOnCopyFailure(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	var failed bool
	realCopy := bm.copyFn
	bm.copyFn = func(src, dst string) error {
		if !failed && strings.Contains(src, targetTS) && strings.HasPrefix(filepath.Base(src), "main.sqlite-wal.") {
			failed = true
			return errors.New("copy boom")
		}
		return realCopy(src, dst)
	}

	app := &fakeAppController{running: []bool{true, false}}
	exec := &restoreExecutor{
		backups:      bm,
		bundleID:     defaultBundleID,
		app:          app,
		sleep:        func(time.Duration) {},
		pollInterval: time.Millisecond,
		stopTimeout:  time.Second,
	}

	_, err = exec.Restore(context.Background(), targetTS)
	if err == nil || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected rollback success error, got %v", err)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:after" {
		t.Fatalf("expected rollback to restore main.sqlite, got %q", got)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite-wal"); got != "main.sqlite-wal:after" {
		t.Fatalf("expected rollback to restore main.sqlite-wal, got %q", got)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 {
		t.Fatalf("expected one quit call, got %d", quitCalls)
	}
	if activateCalls != 1 {
		t.Fatalf("expected one activate call after rollback, got %d", activateCalls)
	}
}

func TestRestoreExecutorSkipsLifecycleWhenAppWasNotRunning(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false}}
	exec := &restoreExecutor{
		backups:      bm,
		bundleID:     defaultBundleID,
		app:          app,
		sleep:        func(time.Duration) {},
		pollInterval: time.Millisecond,
		stopTimeout:  time.Second,
	}

	if _, err := exec.Restore(context.Background(), targetTS); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 0 || activateCalls != 0 {
		t.Fatalf("expected no lifecycle calls, quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestRestoreExecutorTimesOutIfAppDoesNotStop(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	app := &fakeAppController{running: []bool{true, true, true, true}}
	exec := &restoreExecutor{
		backups:      bm,
		bundleID:     defaultBundleID,
		app:          app,
		sleep:        func(time.Duration) {},
		pollInterval: time.Millisecond,
		stopTimeout:  time.Nanosecond,
	}

	_, err = exec.Restore(context.Background(), targetTS)
	if err == nil || !strings.Contains(err.Error(), "did not stop") {
		t.Fatalf("expected stop timeout, got %v", err)
	}
}

func TestVerifySnapshotAgainstLiveDetectsMismatch(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}

	writeLiveDBSet(t, tmp, "different")
	err = verifySnapshotAgainstLive(tmp, created)
	if err == nil || !strings.Contains(err.Error(), "live file mismatch") {
		t.Fatalf("expected live mismatch, got %v", err)
	}
}
