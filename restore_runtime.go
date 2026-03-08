package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
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

func (r *restoreExecutor) waitForRunning(ctx context.Context) error {
	deadline := time.Now().Add(r.launchTimeout)
	for {
		running, err := r.app.IsRunning(ctx, r.bundleID)
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Things did not launch within %s", r.launchTimeout)
		}
		r.sleep(r.pollInterval)
	}
}

func (r *restoreExecutor) closeAfterTemporaryLaunch(ctx context.Context) error {
	if err := r.app.Quit(ctx, r.bundleID); err != nil {
		return err
	}
	return r.waitForStopped(ctx)
}

func (r *restoreExecutor) activateWithin(ctx context.Context, label string) error {
	timeout := r.launchTimeout
	if timeout <= 0 {
		timeout = restoreLaunchTimeout
	}
	activateCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err := r.app.Activate(activateCtx, r.bundleID)
	if err == nil {
		return nil
	}
	if errors.Is(activateCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out after %s", label, timeout)
	}
	return err
}

func (r *restoreExecutor) launchIsolatedWithin(ctx context.Context, label string) error {
	timeout := r.launchTimeout
	if timeout <= 0 {
		timeout = restoreLaunchTimeout
	}
	launchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := r.launchIsolated(launchCtx, r.bundleID); err != nil {
		if errors.Is(launchCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s timed out after %s", label, timeout)
		}
		return err
	}
	if err := r.waitForRunning(launchCtx); err != nil {
		if errors.Is(launchCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s timed out after %s", label, timeout)
		}
		return err
	}
	return nil
}

func (r *restoreExecutor) waitOfflineHold(ctx context.Context) error {
	if r.offlineHold <= 0 {
		return nil
	}
	timer := time.NewTimer(r.offlineHold)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *restoreExecutor) reopenOnlineAfterIsolation(ctx context.Context) error {
	if err := r.quiesce(ctx, true); err != nil {
		return err
	}
	return r.activateWithin(ctx, "reopen online")
}

func (r *restoreExecutor) waitForStableFiles(ctx context.Context) error {
	deadline := time.Now().Add(r.stabilityTimeout)
	requiredPasses := r.stablePasses
	if requiredPasses <= 0 {
		requiredPasses = restoreStablePasses
	}

	var previous []liveFileState
	stableCount := 0
	for {
		current, err := r.captureFileState(r.backups.dataDir)
		if err != nil {
			return fmt.Errorf("capture live file state: %w", err)
		}
		if liveFileStatesEqual(previous, current) {
			stableCount++
			if stableCount >= requiredPasses {
				return nil
			}
		} else {
			stableCount = 1
			previous = current
		}

		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Things database files did not stabilize within %s", r.stabilityTimeout)
		}
		r.sleep(r.pollInterval)
	}
}

func (r *restoreExecutor) quiesce(ctx context.Context, wasRunning bool) error {
	if wasRunning {
		if err := r.app.Quit(ctx, r.bundleID); err != nil {
			return err
		}
		if err := r.waitForStopped(ctx); err != nil {
			return err
		}
		if r.quiesceGracePeriod > 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			r.sleep(r.quiesceGracePeriod)
		}
		running, err := r.app.IsRunning(ctx, r.bundleID)
		if err != nil {
			return err
		}
		if running {
			return errors.New("Things restarted during quiescence")
		}
	}
	return r.waitForStableFiles(ctx)
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
