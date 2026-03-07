package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type backupExecutor struct {
	runtime     *restoreExecutor
	healthCheck func(context.Context) (backupSemanticSnapshot, error)
	stateCheck  func(context.Context) (thingsStateSnapshot, error)
}

func newBackupExecutor(cfg *runtimeConfig) *backupExecutor {
	bundleID := cfg.bundleID
	if bundleID == "" {
		bundleID = defaultBundleID
	}
	runner := cfg.runner
	if runner == nil {
		runner = newRuntimeRunner(bundleID)
	}
	runtime := newRestoreExecutor(cfg)
	runtime.bundleID = bundleID
	runtime.app = scriptAppController{runner: runner}
	runtime.semanticCheck = newScriptSemanticSnapshotter(bundleID, runner).Snapshot
	runtime.semanticTimeout = restoreFullSemanticTimeout
	runtime.backups = newBackupManager(cfg.dataDir)
	return &backupExecutor{
		runtime:     runtime,
		healthCheck: newScriptSemanticHealthSnapshotter(bundleID, runner).Snapshot,
		stateCheck:  newScriptStateSnapshotter(bundleID, runner).Snapshot,
	}
}

func (b *backupExecutor) Create(ctx context.Context) (paths []string, err error) {
	if err := b.runtime.ensureBackupWritable(); err != nil {
		return nil, fmt.Errorf("check backup directory writability: %w", err)
	}

	wasRunning, err := b.runtime.app.IsRunning(ctx, b.runtime.bundleID)
	if err != nil {
		return nil, err
	}

	reopened := !wasRunning
	quiesced := false
	defer func() {
		if err == nil || !wasRunning || !quiesced || reopened {
			return
		}
		if reopenErr := b.runtime.app.Activate(ctx, b.runtime.bundleID); reopenErr != nil {
			err = fmt.Errorf("%w; reopen after backup failure: %v", err, reopenErr)
			return
		}
		reopened = true
	}()

	if err := b.runtime.quiesce(ctx, wasRunning); err != nil {
		return nil, fmt.Errorf("quiesce before backup: %w", err)
	}
	quiesced = true

	paths, err = b.runtime.backups.Create(ctx)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("backup created no files")
	}

	timestamp := inferTimestamp(paths[0])
	if timestamp == "" {
		return paths, errors.New("backup created but timestamp could not be inferred")
	}

	semantic, state, err := b.captureBackupManifests(ctx, wasRunning)
	if err != nil {
		return paths, fmt.Errorf("backup created but state manifests failed: %w", err)
	}

	if err := b.runtime.backups.writeSemanticSnapshot(timestamp, semantic); err != nil {
		return paths, fmt.Errorf("backup created but semantic snapshot save failed: %w", err)
	}
	if err := b.runtime.backups.writeStateSnapshot(timestamp, state); err != nil {
		return paths, fmt.Errorf("backup created but state snapshot save failed: %w", err)
	}

	if wasRunning {
		reopened = true
	}
	return paths, nil
}

func (b *backupExecutor) captureBackupManifests(ctx context.Context, wasRunning bool) (backupSemanticSnapshot, thingsStateSnapshot, error) {
	snapshot, snapshotErr := b.runtime.semanticCheckWithin(ctx, "backup semantic snapshot")
	if snapshotErr != nil && strings.Contains(snapshotErr.Error(), "timed out") && b.healthCheck != nil {
		snapshot, snapshotErr = b.healthCheck(ctx)
	}
	stateSnapshot := thingsStateSnapshot{}
	stateErr := error(nil)
	if b.stateCheck != nil {
		stateSnapshot, stateErr = b.stateCheck(ctx)
	}
	if !wasRunning {
		if closeErr := b.runtime.closeAfterTemporaryLaunch(ctx); closeErr != nil {
			if snapshotErr != nil {
				return backupSemanticSnapshot{}, thingsStateSnapshot{}, fmt.Errorf("%w; restore app state: %v", snapshotErr, closeErr)
			}
			if stateErr != nil {
				return backupSemanticSnapshot{}, thingsStateSnapshot{}, fmt.Errorf("%w; restore app state: %v", stateErr, closeErr)
			}
			return backupSemanticSnapshot{}, thingsStateSnapshot{}, fmt.Errorf("restore app state: %w", closeErr)
		}
	}
	if snapshotErr != nil {
		return backupSemanticSnapshot{}, thingsStateSnapshot{}, snapshotErr
	}
	if stateErr != nil {
		return backupSemanticSnapshot{}, thingsStateSnapshot{}, stateErr
	}
	return snapshot, stateSnapshot, nil
}
