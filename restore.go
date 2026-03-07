package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	restorePollInterval = 200 * time.Millisecond
	restoreStopTimeout  = 5 * time.Second
)

type appController interface {
	IsRunning(ctx context.Context, bundleID string) (bool, error)
	Quit(ctx context.Context, bundleID string) error
	Activate(ctx context.Context, bundleID string) error
}

type scriptAppController struct {
	runner scriptRunner
}

func (c scriptAppController) IsRunning(ctx context.Context, bundleID string) (bool, error) {
	out, err := c.runner.run(ctx, scriptAppRunning(bundleID))
	if err != nil {
		return false, fmt.Errorf("check Things running state: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(out)) {
	case "true":
		return true, nil
	case "", "false":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected running state output: %q", out)
	}
}

func (c scriptAppController) Quit(ctx context.Context, bundleID string) error {
	if _, err := c.runner.run(ctx, scriptQuitApp(bundleID)); err != nil {
		return fmt.Errorf("quit Things: %w", err)
	}
	return nil
}

func (c scriptAppController) Activate(ctx context.Context, bundleID string) error {
	if _, err := c.runner.run(ctx, scriptActivateApp(bundleID)); err != nil {
		return fmt.Errorf("reopen Things: %w", err)
	}
	return nil
}

type restoreExecutor struct {
	backups      *backupManager
	bundleID     string
	app          appController
	sleep        func(time.Duration)
	pollInterval time.Duration
	stopTimeout  time.Duration
}

func newRestoreExecutor(cfg *runtimeConfig) *restoreExecutor {
	return &restoreExecutor{
		backups:      newBackupManager(cfg.dataDir),
		bundleID:     cfg.bundleID,
		app:          scriptAppController{runner: cfg.runner},
		sleep:        time.Sleep,
		pollInterval: restorePollInterval,
		stopTimeout:  restoreStopTimeout,
	}
}

func (r *restoreExecutor) Restore(ctx context.Context, timestamp string) ([]string, error) {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		latest, err := r.backups.Latest(ctx)
		if err != nil {
			return nil, err
		}
		timestamp = latest
	}

	targetFiles, err := r.backups.FilesForTimestamp(ctx, timestamp)
	if err != nil {
		return nil, err
	}

	wasRunning, err := r.app.IsRunning(ctx, r.bundleID)
	if err != nil {
		return nil, err
	}

	preRestoreBackup, err := r.backups.Create(ctx)
	if err != nil {
		return nil, fmt.Errorf("pre-restore backup failed: %w", err)
	}
	preRestoreTS := inferTimestamp(preRestoreBackup[0])
	if preRestoreTS == "" {
		return nil, errors.New("pre-restore backup timestamp could not be inferred")
	}

	if wasRunning {
		if err := r.app.Quit(ctx, r.bundleID); err != nil {
			return nil, err
		}
		if err := r.waitForStopped(ctx); err != nil {
			return nil, err
		}
	}

	restored, err := r.backups.Restore(ctx, timestamp)
	if err != nil {
		return nil, r.restoreFailure(ctx, preRestoreTS, wasRunning, fmt.Errorf("restore snapshot %s: %w", timestamp, err))
	}
	if err := verifySnapshotAgainstLive(r.backups.dataDir, targetFiles); err != nil {
		return nil, r.restoreFailure(ctx, preRestoreTS, wasRunning, fmt.Errorf("verify restored snapshot %s: %w", timestamp, err))
	}

	if wasRunning {
		if err := r.app.Activate(ctx, r.bundleID); err != nil {
			return restored, err
		}
	}
	return restored, nil
}

func (r *restoreExecutor) waitForStopped(ctx context.Context) error {
	deadline := time.Now().Add(r.stopTimeout)
	for {
		running, err := r.app.IsRunning(ctx, r.bundleID)
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Things did not stop within %s", r.stopTimeout)
		}
		r.sleep(r.pollInterval)
	}
}

func (r *restoreExecutor) restoreFailure(ctx context.Context, rollbackTS string, wasRunning bool, cause error) error {
	_, rollbackErr := r.backups.Restore(ctx, rollbackTS)
	if rollbackErr != nil {
		return fmt.Errorf("%w; rollback failed: %v", cause, rollbackErr)
	}
	if wasRunning {
		if err := r.app.Activate(ctx, r.bundleID); err != nil {
			return fmt.Errorf("%w; rollback succeeded; reopen failed: %v", cause, err)
		}
	}
	return fmt.Errorf("%w; rollback succeeded", cause)
}

func verifySnapshotAgainstLive(dataDir string, snapshotFiles []string) error {
	for _, snapshot := range snapshotFiles {
		live := filepath.Join(dataDir, liveDBBaseName(snapshot))
		match, err := filesEqual(snapshot, live)
		if err != nil {
			return fmt.Errorf("compare %s with %s: %w", snapshot, live, err)
		}
		if !match {
			return fmt.Errorf("live file mismatch for %s", filepath.Base(live))
		}
	}
	return nil
}

func liveDBBaseName(snapshotPath string) string {
	base := filepath.Base(snapshotPath)
	switch {
	case strings.HasPrefix(base, "main.sqlite-shm."):
		return "main.sqlite-shm"
	case strings.HasPrefix(base, "main.sqlite-wal."):
		return "main.sqlite-wal"
	default:
		return "main.sqlite"
	}
}

func filesEqual(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}

	lf, err := os.Open(left)
	if err != nil {
		return false, err
	}
	defer lf.Close()

	rf, err := os.Open(right)
	if err != nil {
		return false, err
	}
	defer rf.Close()

	leftBuf := make([]byte, 32*1024)
	rightBuf := make([]byte, 32*1024)
	for {
		leftN, leftErr := lf.Read(leftBuf)
		rightN, rightErr := rf.Read(rightBuf)
		if leftN != rightN {
			return false, nil
		}
		if leftN > 0 && !bytesEqual(leftBuf[:leftN], rightBuf[:rightN]) {
			return false, nil
		}
		if errors.Is(leftErr, io.EOF) && errors.Is(rightErr, io.EOF) {
			return true, nil
		}
		if leftErr != nil && !errors.Is(leftErr, io.EOF) {
			return false, leftErr
		}
		if rightErr != nil && !errors.Is(rightErr, io.EOF) {
			return false, rightErr
		}
	}
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func scriptAppRunning(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  return running
end tell`, escapeApple(bundleID))
}

func scriptQuitApp(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  quit
end tell
return "ok"`, escapeApple(bundleID))
}

func scriptActivateApp(bundleID string) string {
	return fmt.Sprintf(`tell application id "%s"
  activate
end tell
return "ok"`, escapeApple(bundleID))
}
