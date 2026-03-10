package app

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const backupSettleDelay = 5 * time.Second

type backupExecutor struct {
	runtime     *restoreExecutor
	settleDelay time.Duration
	createMeta  backupCreateMetadata
}

func newBackupExecutor(cfg *runtimeConfig) *backupExecutor {
	return newBackupExecutorWithMetadata(cfg, backupCreateMetadata{
		Kind:          backupKindExplicit,
		SourceCommand: "backup",
		Reason:        "manual checkpoint",
	})
}

func newSessionBackupExecutor(cfg *runtimeConfig) *backupExecutor {
	return newBackupExecutorWithMetadata(cfg, backupCreateMetadata{
		Kind:          backupKindSession,
		SourceCommand: "session-start",
		Reason:        "session bootstrap checkpoint",
	})
}

func newDestructiveBackupExecutor(cfg *runtimeConfig) *backupExecutor {
	return newBackupExecutorWithMetadata(cfg, backupCreateMetadata{
		Kind:          backupKindSafety,
		SourceCommand: "auto-safety",
		Reason:        "automatic rollback checkpoint",
	})
}

func newBackupExecutorWithMetadata(cfg *runtimeConfig, meta backupCreateMetadata) *backupExecutor {
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
	runtime.semanticCheck = newScriptSemanticManifestProbe(bundleID, runner).Snapshot
	runtime.semanticTimeout = restoreFullSemanticTimeout
	runtime.backups = newBackupManager(cfg.dataDir)
	return &backupExecutor{
		runtime:     runtime,
		settleDelay: backupSettleDelay,
		createMeta:  meta,
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
		if !wasRunning || !quiesced || reopened {
			return
		}
		if reopenErr := b.runtime.app.Activate(ctx, b.runtime.bundleID); reopenErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; reopen after backup failure: %v", err, reopenErr)
			} else {
				err = fmt.Errorf("backup succeeded but reopen failed: %w", reopenErr)
			}
			return
		}
		reopened = true
	}()

	if wasRunning && b.settleDelay > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b.runtime.sleep(b.settleDelay)
	}

	if err := b.runtime.quiesce(ctx, wasRunning); err != nil {
		return nil, fmt.Errorf("quiesce before backup: %w", err)
	}
	quiesced = true

	paths, err = b.runtime.backups.CreateWithMetadata(ctx, b.createMeta)
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
	return paths, nil
}
