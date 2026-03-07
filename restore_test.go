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
	quitCalls     int
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

func newTestRestoreExecutor(bm *backupManager, app appController) *restoreExecutor {
	return &restoreExecutor{
		backups:          bm,
		bundleID:         defaultBundleID,
		app:              app,
		sleep:            func(time.Duration) {},
		pollInterval:     time.Millisecond,
		stopTimeout:      time.Second,
		stabilityTimeout: time.Second,
		stablePasses:     2,
		captureFileState: func(string) ([]liveFileState, error) {
			return []liveFileState{
				{Name: "main.sqlite", Size: 1, ModTime: 1},
				{Name: "main.sqlite-shm", Size: 1, ModTime: 1},
				{Name: "main.sqlite-wal", Size: 1, ModTime: 1},
			}, nil
		},
		semanticCheck: func(context.Context) (restoreSemanticVerification, error) {
			return restoreSemanticVerification{OK: true, Lists: 1, Projects: 0}, nil
		},
	}
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
	exec := newTestRestoreExecutor(bm, app)

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

func TestRestoreExecutorPreflightReportsReadyState(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	app := &fakeAppController{running: []bool{false}}
	exec := newTestRestoreExecutor(bm, app)

	report, err := exec.Preflight(context.Background(), targetTS)
	if err != nil {
		t.Fatalf("preflight failed: %v", err)
	}
	if !report.OK || !report.Complete || !report.LiveFilesPresent || !report.BackupWritable {
		t.Fatalf("unexpected preflight report: %#v", report)
	}
	if report.Timestamp != targetTS {
		t.Fatalf("expected target timestamp %q, got %q", targetTS, report.Timestamp)
	}
}

func TestRestoreExecutorDryRunReturnsJournalWithoutMutatingLiveFiles(t *testing.T) {
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
	exec := newTestRestoreExecutor(bm, app)

	journal, err := exec.Execute(context.Background(), targetTS, true)
	if err != nil {
		t.Fatalf("dry-run execute failed: %v", err)
	}
	if !journal.DryRun || journal.Outcome != "dry-run" {
		t.Fatalf("unexpected dry-run journal: %#v", journal)
	}
	if journal.PreRestoreBackup != nil || len(journal.RestoredFiles) != 0 || journal.Verification != nil {
		t.Fatalf("expected dry-run to skip backup/restore/verify, got %#v", journal)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:after" {
		t.Fatalf("expected dry-run to keep main.sqlite untouched, got %q", got)
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
	exec := newTestRestoreExecutor(bm, app)

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
	exec := newTestRestoreExecutor(bm, app)

	if _, err := exec.Restore(context.Background(), targetTS); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected temporary semantic launch lifecycle, quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestRestoreExecutorRunsSemanticVerificationAndRestoresClosedState(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false, true, false, false}}
	exec := newTestRestoreExecutor(bm, app)
	semanticCalls := 0
	exec.semanticCheck = func(context.Context) (restoreSemanticVerification, error) {
		semanticCalls++
		return restoreSemanticVerification{OK: true, Lists: 2, Projects: 1}, nil
	}

	journal, err := exec.Execute(context.Background(), targetTS, false)
	if err != nil {
		t.Fatalf("restore execute failed: %v", err)
	}
	if semanticCalls != 1 {
		t.Fatalf("expected one semantic verification call, got %d", semanticCalls)
	}
	if journal.SemanticVerification == nil || !journal.SemanticVerification.OK || !journal.SemanticVerification.TemporaryLaunch {
		t.Fatalf("unexpected semantic verification journal: %#v", journal.SemanticVerification)
	}
	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected temporary launch lifecycle, quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestRestoreExecutorRollsBackWhenSemanticVerificationFails(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false, true, false, false}}
	exec := newTestRestoreExecutor(bm, app)
	exec.semanticCheck = func(context.Context) (restoreSemanticVerification, error) {
		return restoreSemanticVerification{}, errors.New("semantic probe failed")
	}

	_, err = exec.Execute(context.Background(), targetTS, false)
	if err == nil || !strings.Contains(err.Error(), "semantic verify restored snapshot") || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected semantic rollback error, got %v", err)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:after" {
		t.Fatalf("expected rollback to restore previous live state, got %q", got)
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
	exec := newTestRestoreExecutor(bm, app)
	exec.stopTimeout = time.Nanosecond

	_, err = exec.Restore(context.Background(), targetTS)
	if err == nil || !strings.Contains(err.Error(), "did not stop") {
		t.Fatalf("expected stop timeout, got %v", err)
	}
}

func TestRestoreExecutorAppliesQuiesceGuardDelayAfterStop(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	var slept []time.Duration
	app := &fakeAppController{running: []bool{true, true, false, false}}
	exec := newTestRestoreExecutor(bm, app)
	exec.quiesceGracePeriod = 5 * time.Millisecond
	exec.sleep = func(d time.Duration) {
		slept = append(slept, d)
	}

	if _, err := exec.Restore(context.Background(), targetTS); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	foundGuard := false
	for _, d := range slept {
		if d == exec.quiesceGracePeriod {
			foundGuard = true
			break
		}
	}
	if !foundGuard {
		t.Fatalf("expected quiesce grace period sleep, got %#v", slept)
	}
}

func TestRestoreExecutorFailsWhenAppRestartsDuringQuiesce(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])

	app := &fakeAppController{running: []bool{true, true, false, true}}
	exec := newTestRestoreExecutor(bm, app)
	exec.quiesceGracePeriod = time.Millisecond

	_, err = exec.Restore(context.Background(), targetTS)
	if err == nil || !strings.Contains(err.Error(), "restarted during quiescence") {
		t.Fatalf("expected restart during quiescence error, got %v", err)
	}
}

func TestRestoreExecutorWaitsForStableFiles(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	exec := &restoreExecutor{
		backups:          newBackupManager(tmp),
		app:              &fakeAppController{},
		bundleID:         defaultBundleID,
		sleep:            func(time.Duration) {},
		pollInterval:     time.Millisecond,
		stabilityTimeout: time.Second,
		stablePasses:     2,
	}

	states := [][]liveFileState{
		{
			{Name: "main.sqlite", Size: 1, ModTime: 1},
			{Name: "main.sqlite-shm", Size: 1, ModTime: 1},
			{Name: "main.sqlite-wal", Size: 1, ModTime: 1},
		},
		{
			{Name: "main.sqlite", Size: 2, ModTime: 2},
			{Name: "main.sqlite-shm", Size: 2, ModTime: 2},
			{Name: "main.sqlite-wal", Size: 2, ModTime: 2},
		},
		{
			{Name: "main.sqlite", Size: 2, ModTime: 2},
			{Name: "main.sqlite-shm", Size: 2, ModTime: 2},
			{Name: "main.sqlite-wal", Size: 2, ModTime: 2},
		},
	}
	var index int
	exec.captureFileState = func(string) ([]liveFileState, error) {
		if index >= len(states) {
			return states[len(states)-1], nil
		}
		current := states[index]
		index++
		return current, nil
	}

	if err := exec.waitForStableFiles(context.Background()); err != nil {
		t.Fatalf("waitForStableFiles failed: %v", err)
	}
	if index < 3 {
		t.Fatalf("expected multiple stability probes, got %d", index)
	}
}

func TestRestoreExecutorStableFilesTimeout(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	exec := &restoreExecutor{
		backups:          newBackupManager(tmp),
		app:              &fakeAppController{},
		bundleID:         defaultBundleID,
		sleep:            func(time.Duration) {},
		pollInterval:     time.Millisecond,
		stabilityTimeout: time.Nanosecond,
		stablePasses:     2,
		captureFileState: func(string) ([]liveFileState, error) {
			return []liveFileState{
				{Name: "main.sqlite", Size: time.Now().UnixNano(), ModTime: time.Now().UnixNano()},
				{Name: "main.sqlite-shm", Size: 1, ModTime: 1},
				{Name: "main.sqlite-wal", Size: 1, ModTime: 1},
			}, nil
		},
	}

	err := exec.waitForStableFiles(context.Background())
	if err == nil || !strings.Contains(err.Error(), "did not stabilize") {
		t.Fatalf("expected stability timeout, got %v", err)
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

func TestBuildSnapshotVerificationReturnsPerFileDetails(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	writeLiveDBSet(t, tmp, "after")

	report, err := buildSnapshotVerification(tmp, created)
	if err != nil {
		t.Fatalf("buildSnapshotVerification failed: %v", err)
	}
	if report.Match {
		t.Fatalf("expected verification mismatch, got %#v", report)
	}
	if len(report.Files) != 3 {
		t.Fatalf("expected per-file verification entries, got %#v", report.Files)
	}
	mismatches := 0
	for _, file := range report.Files {
		if !file.Match {
			mismatches++
		}
	}
	if mismatches == 0 {
		t.Fatalf("expected at least one mismatching file, got %#v", report.Files)
	}
}
