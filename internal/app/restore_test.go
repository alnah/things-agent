package app

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
	mu            sync.Mutex
	running       []bool
	quitCalls     int
	activateCalls int
	activateWait  <-chan struct{}
	quitFn        func()
	activateFn    func()
	quitErr       error
	activateErr   error
	runningErr    error
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
	if f.quitFn != nil {
		f.quitFn()
	}
	return f.quitErr
}

func (f *fakeAppController) Activate(ctx context.Context, _ string) error {
	f.mu.Lock()
	f.activateCalls++
	wait := f.activateWait
	fn := f.activateFn
	err := f.activateErr
	f.mu.Unlock()
	if wait != nil {
		select {
		case <-wait:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if fn != nil {
		fn()
	}
	return err
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

func testSemanticManifest(lists, projects, tasks int) backupSemanticManifest {
	return backupSemanticManifest{
		ListsCount:    lists,
		ListsHash:     strings.Repeat("l", 64),
		ProjectsCount: projects,
		ProjectsHash:  strings.Repeat("p", 64),
		TasksCount:    tasks,
		TasksHash:     strings.Repeat("t", 64),
	}
}

func newTestRestoreExecutor(bm *backupManager, app appController) *restoreExecutor {
	return &restoreExecutor{
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
		semanticCheck: func(context.Context) (backupSemanticManifest, error) {
			return testSemanticManifest(1, 0, 0), nil
		},
		fullSemanticCheck: func(context.Context) (backupSemanticManifest, error) {
			return testSemanticManifest(1, 0, 0), nil
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
	if quitCalls != 2 {
		t.Fatalf("expected two quit calls, got %d", quitCalls)
	}
	if activateCalls != 0 {
		t.Fatalf("expected no explicit activate calls, got %d", activateCalls)
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

func TestRestoreExecutorRecordsSafetyPreRestoreBackup(t *testing.T) {
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

	journal, err := exec.Execute(context.Background(), targetTS, false)
	if err != nil {
		t.Fatalf("execute restore failed: %v", err)
	}
	if journal.PreRestoreBackup == nil || journal.PreRestoreBackup.Kind != string(backupKindSafety) {
		t.Fatalf("expected safety pre-restore backup metadata, got %#v", journal.PreRestoreBackup)
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
	if quitCalls != 1 || activateCalls != 0 {
		t.Fatalf("expected semantic close-only lifecycle, quit=%d activate=%d", quitCalls, activateCalls)
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
	exec.semanticCheck = func(context.Context) (backupSemanticManifest, error) {
		semanticCalls++
		return testSemanticManifest(2, 1, 3), nil
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
	if journal.SemanticVerification.Actual.ListsCount != 2 || journal.SemanticVerification.Actual.ProjectsCount != 1 || journal.SemanticVerification.Actual.TasksCount != 3 {
		t.Fatalf("unexpected semantic verification actual snapshot: %#v", journal.SemanticVerification)
	}
	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 0 {
		t.Fatalf("expected temporary semantic close-only lifecycle, quit=%d activate=%d", quitCalls, activateCalls)
	}
}

func TestRestoreExecutorLaunchesOfflineAndRelaunchesOnline(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false, false, true, true, false}}
	exec := newTestRestoreExecutor(bm, app)
	launchCalls := 0
	exec.launchIsolated = func(context.Context, string) error {
		launchCalls++
		return nil
	}
	exec.networkIsolation = networkIsolationSandboxNoNetwork
	exec.offlineHold = time.Millisecond
	exec.reopenOnline = true
	exec.semanticCheck = func(context.Context) (backupSemanticManifest, error) {
		return testSemanticManifest(2, 1, 3), nil
	}

	journal, err := exec.Execute(context.Background(), targetTS, false)
	if err != nil {
		t.Fatalf("restore execute failed: %v", err)
	}
	if launchCalls != 1 {
		t.Fatalf("expected one isolated launch, got %d", launchCalls)
	}
	if journal.NetworkIsolation != networkIsolationSandboxNoNetwork || !journal.RelaunchedOnline {
		t.Fatalf("unexpected network isolation journal: %#v", journal)
	}
	if journal.SemanticVerification == nil || !journal.SemanticVerification.OK || journal.SemanticVerification.Actual.TasksCount != 3 {
		t.Fatalf("unexpected offline smoke verification: %#v", journal.SemanticVerification)
	}
	quitCalls, activateCalls := app.counts()
	if quitCalls != 1 || activateCalls != 1 {
		t.Fatalf("expected one quit and one online activate, quit=%d activate=%d", quitCalls, activateCalls)
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
	exec.semanticCheck = func(context.Context) (backupSemanticManifest, error) {
		return backupSemanticManifest{}, errors.New("semantic probe failed")
	}

	_, err = exec.Execute(context.Background(), targetTS, false)
	if err == nil || !strings.Contains(err.Error(), "semantic verify restored snapshot") || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected semantic rollback error, got %v", err)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:after" {
		t.Fatalf("expected rollback to restore previous live state, got %q", got)
	}
}

func TestRestoreExecutorRollsBackWhenSemanticCheckTimesOut(t *testing.T) {
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
	exec.semanticTimeout = time.Nanosecond
	exec.semanticCheck = func(ctx context.Context) (backupSemanticManifest, error) {
		<-ctx.Done()
		return backupSemanticManifest{}, ctx.Err()
	}

	_, err = exec.Execute(context.Background(), targetTS, false)
	if err == nil || !strings.Contains(err.Error(), "semantic verify restored snapshot") || !strings.Contains(err.Error(), "semantic verify timed out") || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected semantic timeout rollback error, got %v", err)
	}
}

func TestRestoreExecutorRollsBackWhenPostLaunchVerificationFails(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{true, true, false, false, true, false, false}}
	exec := newTestRestoreExecutor(bm, app)
	exec.semanticCheck = func(context.Context) (backupSemanticManifest, error) {
		writeLiveDBSet(t, tmp, "drifted")
		return testSemanticManifest(1, 0, 0), nil
	}

	_, err = exec.Execute(context.Background(), targetTS, false)
	if err == nil || !strings.Contains(err.Error(), "post-launch verify restored snapshot") || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected post-launch verification rollback error, got %v", err)
	}
	if got := readLiveDBFile(t, tmp, "main.sqlite"); got != "main.sqlite:after" {
		t.Fatalf("expected rollback to restore previous live state, got %q", got)
	}
}

func TestRestoreExecutorUsesSemanticManifestInsteadOfPostLaunchFileDiff(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	expectedManifest := testSemanticManifest(2, 1, 3)
	bm.semanticManifest = func(context.Context) (backupSemanticManifest, error) {
		return expectedManifest, nil
	}
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{
		running: []bool{true, true, false},
	}
	exec := newTestRestoreExecutor(bm, app)
	exec.fullSemanticCheck = func(context.Context) (backupSemanticManifest, error) {
		return expectedManifest, nil
	}

	journal, err := exec.Execute(context.Background(), targetTS, false)
	if err != nil {
		t.Fatalf("expected manifest-backed restore to succeed: %v", err)
	}
	if journal.SemanticVerification == nil || !journal.SemanticVerification.ComparedToManifest || !journal.SemanticVerification.OK {
		t.Fatalf("expected manifest-backed semantic verification, got %#v", journal.SemanticVerification)
	}
	if journal.PostLaunchVerification != nil {
		t.Fatalf("expected post-launch file verification to be skipped when manifest exists, got %#v", journal.PostLaunchVerification)
	}
}

func TestRestoreExecutorRollsBackWhenSemanticManifestMismatches(t *testing.T) {
	tmp := t.TempDir()
	writeLiveDBSet(t, tmp, "before")

	bm := newBackupManager(tmp)
	bm.semanticManifest = func(context.Context) (backupSemanticManifest, error) {
		return testSemanticManifest(2, 1, 3), nil
	}
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("seed snapshot: %v", err)
	}
	targetTS := inferTimestamp(created[0])
	writeLiveDBSet(t, tmp, "after")

	app := &fakeAppController{running: []bool{false, true, false, false}}
	exec := newTestRestoreExecutor(bm, app)
	exec.fullSemanticCheck = func(context.Context) (backupSemanticManifest, error) {
		return testSemanticManifest(2, 1, 4), nil
	}

	_, err = exec.Execute(context.Background(), targetTS, false)
	if err == nil || !strings.Contains(err.Error(), "semantic verify restored snapshot") || !strings.Contains(err.Error(), "task manifest mismatch") || !strings.Contains(err.Error(), "rollback succeeded") {
		t.Fatalf("expected semantic manifest rollback error, got %v", err)
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
