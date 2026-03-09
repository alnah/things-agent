package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	restorePollInterval        = 200 * time.Millisecond
	restoreStopTimeout         = 5 * time.Second
	restoreStabilityTimeout    = 2 * time.Second
	restoreStablePasses        = 2
	restoreQuiesceGracePeriod  = 300 * time.Millisecond
	restoreLaunchTimeout       = 15 * time.Second
	restoreSemanticTimeout     = 15 * time.Second
	restoreFullSemanticTimeout = 60 * time.Second
)

type restoreExecutor struct {
	backups             *backupManager
	bundleID            string
	app                 appController
	launchIsolated      offlineAppLaunchFunc
	networkIsolation    string
	offlineHold         time.Duration
	reopenOnline        bool
	sleep               func(time.Duration)
	pollInterval        time.Duration
	stopTimeout         time.Duration
	stabilityTimeout    time.Duration
	stablePasses        int
	quiesceGracePeriod  time.Duration
	launchTimeout       time.Duration
	semanticTimeout     time.Duration
	fullSemanticTimeout time.Duration
	captureFileState    func(string) ([]liveFileState, error)
	semanticCheck       func(context.Context) (backupSemanticManifest, error)
	fullSemanticCheck   func(context.Context) (backupSemanticManifest, error)
}

type restorePreflightReport struct {
	Timestamp        string   `json:"timestamp"`
	Complete         bool     `json:"complete"`
	Files            []string `json:"files"`
	AppRunning       bool     `json:"app_running"`
	QuiesceRequired  bool     `json:"quiesce_required"`
	LiveFilesPresent bool     `json:"live_files_present"`
	LiveFilesStable  bool     `json:"live_files_stable"`
	BackupWritable   bool     `json:"backup_writable"`
	OK               bool     `json:"ok"`
}

type restoreBackupRecord struct {
	Timestamp string   `json:"timestamp"`
	Kind      string   `json:"kind,omitempty"`
	Files     []string `json:"files"`
}

type restoreRollbackReport struct {
	Attempted bool   `json:"attempted"`
	Timestamp string `json:"timestamp,omitempty"`
	Succeeded bool   `json:"succeeded"`
	Error     string `json:"error,omitempty"`
}

type restoreJournalStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type restoreJournal struct {
	RequestedTimestamp     string                       `json:"requested_timestamp,omitempty"`
	Timestamp              string                       `json:"timestamp"`
	DryRun                 bool                         `json:"dry_run"`
	Outcome                string                       `json:"outcome"`
	NetworkIsolation       string                       `json:"network_isolation,omitempty"`
	OfflineHold            string                       `json:"offline_hold,omitempty"`
	RelaunchedOnline       bool                         `json:"relaunched_online,omitempty"`
	AppWasRunning          bool                         `json:"app_was_running"`
	Preflight              restorePreflightReport       `json:"preflight"`
	PreRestoreBackup       *restoreBackupRecord         `json:"pre_restore_backup,omitempty"`
	RestoredFiles          []string                     `json:"restored_files,omitempty"`
	Verification           *restoreVerificationReport   `json:"verification,omitempty"`
	PostLaunchVerification *restoreVerificationReport   `json:"post_launch_verification,omitempty"`
	SemanticVerification   *restoreSemanticVerification `json:"semantic_verification,omitempty"`
	Rollback               *restoreRollbackReport       `json:"rollback,omitempty"`
	Steps                  []restoreJournalStep         `json:"steps"`
}

type restoreSemanticVerification struct {
	OK                 bool                    `json:"ok"`
	Expected           *backupSemanticManifest `json:"expected,omitempty"`
	Actual             backupSemanticManifest  `json:"actual"`
	ComparedToManifest bool                    `json:"compared_to_manifest,omitempty"`
	TemporaryLaunch    bool                    `json:"temporary_launch,omitempty"`
}

func newRestoreJournal(requestedTimestamp string, dryRun bool, networkIsolation string, offlineHold time.Duration) restoreJournal {
	journal := restoreJournal{
		RequestedTimestamp: strings.TrimSpace(requestedTimestamp),
		DryRun:             dryRun,
		Outcome:            "failed",
		NetworkIsolation:   strings.TrimSpace(networkIsolation),
	}
	if offlineHold > 0 {
		journal.OfflineHold = offlineHold.String()
	}
	return journal
}

func (j *restoreJournal) addStep(name, status string, err error) {
	step := restoreJournalStep{Name: name, Status: status}
	if err != nil {
		step.Error = err.Error()
	}
	j.Steps = append(j.Steps, step)
}

func newRestoreExecutor(cfg *runtimeConfig) *restoreExecutor {
	return &restoreExecutor{
		backups:             newBackupManager(cfg.dataDir),
		bundleID:            cfg.bundleID,
		app:                 scriptAppController{runner: cfg.runner},
		sleep:               time.Sleep,
		pollInterval:        restorePollInterval,
		stopTimeout:         restoreStopTimeout,
		stabilityTimeout:    restoreStabilityTimeout,
		stablePasses:        restoreStablePasses,
		quiesceGracePeriod:  restoreQuiesceGracePeriod,
		launchTimeout:       restoreLaunchTimeout,
		semanticTimeout:     restoreSemanticTimeout,
		fullSemanticTimeout: restoreFullSemanticTimeout,
		captureFileState:    captureLiveFileState,
		semanticCheck:       newScriptSemanticHealthProbe(cfg.bundleID, cfg.runner).Snapshot,
		fullSemanticCheck:   newScriptSemanticManifestProbe(cfg.bundleID, cfg.runner).Snapshot,
	}
}

func (r *restoreExecutor) Restore(ctx context.Context, timestamp string) ([]string, error) {
	journal, err := r.Execute(ctx, timestamp, false)
	if err != nil {
		return nil, err
	}
	return journal.RestoredFiles, nil
}

func (r *restoreExecutor) Execute(ctx context.Context, timestamp string, dryRun bool) (restoreJournal, error) {
	journal := newRestoreJournal(timestamp, dryRun, r.networkIsolation, r.offlineHold)

	preflight, err := r.Preflight(ctx, timestamp)
	journal.Preflight = preflight
	journal.Timestamp = preflight.Timestamp
	journal.AppWasRunning = preflight.AppRunning
	if err != nil {
		journal.addStep("preflight", "failed", err)
		return journal, err
	}
	journal.addStep("preflight", "ok", nil)

	if dryRun {
		journal.Outcome = "dry-run"
		journal.addStep("dry-run", "ok", nil)
		return journal, nil
	}

	if err := r.quiesce(ctx, preflight.AppRunning); err != nil {
		stepStatus := "failed"
		stepName := "quiesce"
		if !preflight.AppRunning {
			stepName = "stable-files"
		}
		journal.addStep(stepName, stepStatus, err)
		return journal, err
	}
	if preflight.AppRunning {
		journal.addStep("quiesce", "ok", nil)
	}
	journal.addStep("stable-files", "ok", nil)

	preRestoreBackup, preRestoreTS, err := r.createPreRestoreBackup(ctx)
	if err != nil {
		journal.addStep("pre-restore-backup", "failed", err)
		return journal, err
	}
	journal.PreRestoreBackup = preRestoreBackup
	journal.addStep("pre-restore-backup", "ok", nil)

	restored, err := r.backups.Restore(ctx, preflight.Timestamp)
	if err != nil {
		journal.addStep("restore", "failed", err)
		rollback, restoreErr := r.restoreFailure(ctx, preRestoreTS, preflight.AppRunning, fmt.Errorf("restore snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.RestoredFiles = restored
	journal.addStep("restore", "ok", nil)

	verification, err := r.Verify(ctx, preflight.Timestamp)
	journal.Verification = &verification
	if err != nil {
		journal.addStep("verify", "failed", err)
		rollback, restoreErr := r.restoreFailure(ctx, preRestoreTS, preflight.AppRunning, fmt.Errorf("verify restored snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.addStep("verify", "ok", nil)

	if r.backups.packageMode() {
		if err := clearRestoreSyncMetadata(filepath.Join(r.backups.dataDir, "main.sqlite")); err != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "prepare-launch", Status: "failed", Error: err.Error()})
			rollback, restoreErr := r.restoreFailure(ctx, preRestoreTS, preflight.AppRunning, fmt.Errorf("prepare restored snapshot %s for launch: %w", preflight.Timestamp, err))
			journal.Rollback = rollback
			return journal, restoreErr
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "prepare-launch", Status: "ok"})
	}

	if r.launchIsolated != nil {
		actualSemantic, launchErr := r.launchIsolatedAndSmoke(ctx)
		semanticReport := restoreSemanticVerification{
			OK:              launchErr == nil,
			Actual:          actualSemantic,
			TemporaryLaunch: !preflight.AppRunning,
		}
		journal.SemanticVerification = &semanticReport
		if launchErr != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "offline-launch", Status: "failed", Error: launchErr.Error()})
			rollback, restoreErr := r.restoreFailureWithAppState(ctx, preRestoreTS, preflight.AppRunning, false, fmt.Errorf("offline launch verify restored snapshot %s: %w", preflight.Timestamp, launchErr))
			journal.Rollback = rollback
			return journal, restoreErr
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "offline-launch", Status: "ok"})
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "offline-smoke", Status: "ok"})
		if err := r.waitOfflineHold(ctx); err != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "offline-hold", Status: "failed", Error: err.Error()})
			return journal, fmt.Errorf("restore succeeded but offline hold failed: %w", err)
		}
		if r.offlineHold > 0 {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "offline-hold", Status: "ok"})
		}
		if r.reopenOnline {
			if err := r.reopenOnlineAfterIsolation(ctx); err != nil {
				journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reopen-online", Status: "failed", Error: err.Error()})
				return journal, fmt.Errorf("restore succeeded but online relaunch failed: %w", err)
			}
			journal.RelaunchedOnline = true
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reopen-online", Status: "ok"})
		}
		journal.Outcome = "restored"
		return journal, nil
	}

	actualSemantic, err := r.semanticCheckForRestore(ctx, preflight.Timestamp)
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-verify", Status: "failed", Error: err.Error()})
		rollback, restoreErr := r.restoreFailureWithAppState(ctx, preRestoreTS, preflight.AppRunning, true, fmt.Errorf("semantic verify restored snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-launch", Status: "ok"})
	semanticReport, err := r.buildSemanticVerification(preflight.Timestamp, actualSemantic)
	if !preflight.AppRunning {
		semanticReport.TemporaryLaunch = true
	}
	journal.SemanticVerification = &semanticReport
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-verify", Status: "failed", Error: err.Error()})
		rollback, restoreErr := r.restoreFailureWithAppState(ctx, preRestoreTS, preflight.AppRunning, true, fmt.Errorf("semantic verify restored snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-verify", Status: "ok"})

	if semanticReport.ComparedToManifest {
		if !preflight.AppRunning {
			if err := r.closeAfterTemporaryLaunch(ctx); err != nil {
				journal.Steps = append(journal.Steps, restoreJournalStep{Name: "restore-app-state", Status: "failed", Error: err.Error()})
				return journal, err
			}
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "restore-app-state", Status: "ok"})
		}
		journal.Outcome = "restored"
		return journal, nil
	}

	if err := r.quiesce(ctx, true); err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reclose", Status: "failed", Error: err.Error()})
		return journal, err
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reclose", Status: "ok"})

	if !semanticReport.ComparedToManifest && !r.backups.packageMode() {
		postLaunchVerification, err := r.Verify(ctx, preflight.Timestamp)
		journal.PostLaunchVerification = &postLaunchVerification
		if err != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "post-launch-verify", Status: "failed", Error: err.Error()})
			rollback, restoreErr := r.restoreFailureWithAppState(ctx, preRestoreTS, preflight.AppRunning, false, fmt.Errorf("post-launch verify restored snapshot %s: %w", preflight.Timestamp, err))
			journal.Rollback = rollback
			return journal, restoreErr
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "post-launch-verify", Status: "ok"})
	}

	journal.Outcome = "restored"
	return journal, nil
}

func (r *restoreExecutor) resolveTimestamp(ctx context.Context, timestamp string) (string, error) {
	timestamp = strings.TrimSpace(timestamp)
	if timestamp == "" {
		latest, err := r.backups.Latest(ctx)
		if err != nil {
			return "", err
		}
		timestamp = latest
	}
	return timestamp, nil
}

func (r *restoreExecutor) Preflight(ctx context.Context, timestamp string) (restorePreflightReport, error) {
	resolvedTS, err := r.resolveTimestamp(ctx, timestamp)
	if err != nil {
		return restorePreflightReport{}, err
	}

	targetFiles, err := r.backups.FilesForTimestamp(ctx, resolvedTS)
	if err != nil {
		return restorePreflightReport{Timestamp: resolvedTS}, err
	}
	report := restorePreflightReport{
		Timestamp: resolvedTS,
		Complete:  true,
		Files:     targetFiles,
	}

	wasRunning, err := r.app.IsRunning(ctx, r.bundleID)
	if err != nil {
		return report, err
	}
	report.AppRunning = wasRunning
	report.QuiesceRequired = wasRunning

	if _, err := r.captureFileState(r.backups.dataDir); err != nil {
		return report, fmt.Errorf("inspect live database files: %w", err)
	}
	report.LiveFilesPresent = true

	if err := r.ensureBackupWritable(); err != nil {
		return report, fmt.Errorf("check backup directory writability: %w", err)
	}
	report.BackupWritable = true

	if !wasRunning {
		if err := r.quiesce(ctx, false); err != nil {
			return report, fmt.Errorf("preflight stable files: %w", err)
		}
		report.LiveFilesStable = true
	}
	report.OK = report.Complete && report.LiveFilesPresent && report.BackupWritable
	if !wasRunning {
		report.OK = report.OK && report.LiveFilesStable
	}
	return report, nil
}

func (r *restoreExecutor) Verify(ctx context.Context, timestamp string) (restoreVerificationReport, error) {
	resolvedTS, err := r.resolveTimestamp(ctx, timestamp)
	if err != nil {
		return restoreVerificationReport{}, err
	}
	targetFiles, err := r.backups.FilesForTimestamp(ctx, resolvedTS)
	if err != nil {
		return restoreVerificationReport{Timestamp: resolvedTS}, err
	}
	report, err := buildSnapshotVerification(r.backups.dataDir, targetFiles)
	report.Timestamp = resolvedTS
	return report, verificationError(report)
}

func (r *restoreExecutor) semanticCheckWithin(ctx context.Context, label string) (backupSemanticManifest, error) {
	timeout := r.semanticTimeout
	if timeout <= 0 {
		timeout = restoreSemanticTimeout
	}
	semanticCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	manifest, err := r.semanticCheck(semanticCtx)
	if err == nil {
		return manifest, nil
	}
	if errors.Is(semanticCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return backupSemanticManifest{}, fmt.Errorf("%s timed out after %s", label, timeout)
	}
	return backupSemanticManifest{}, err
}

func (r *restoreExecutor) fullSemanticCheckWithin(ctx context.Context, label string) (backupSemanticManifest, error) {
	timeout := r.fullSemanticTimeout
	if timeout <= 0 {
		timeout = restoreFullSemanticTimeout
	}
	semanticCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	manifest, err := r.fullSemanticCheck(semanticCtx)
	if err == nil {
		return manifest, nil
	}
	if errors.Is(semanticCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return backupSemanticManifest{}, fmt.Errorf("%s timed out after %s", label, timeout)
	}
	return backupSemanticManifest{}, err
}

func (r *restoreExecutor) semanticCheckForRestore(ctx context.Context, timestamp string) (backupSemanticManifest, error) {
	expected, err := r.backups.loadSemanticManifest(timestamp)
	if err != nil {
		if os.IsNotExist(err) {
			return r.semanticCheckWithin(ctx, "semantic verify")
		}
		return backupSemanticManifest{}, fmt.Errorf("load semantic manifest %s: %w", timestamp, err)
	}
	if expected.ListsHash != "" || expected.ProjectsHash != "" || expected.TasksHash != "" {
		return r.fullSemanticCheckWithin(ctx, "semantic verify")
	}
	return r.semanticCheckWithin(ctx, "semantic verify")
}

func (r *restoreExecutor) launchIsolatedAndSmoke(ctx context.Context) (backupSemanticManifest, error) {
	if err := r.launchIsolatedWithin(ctx, "offline launch"); err != nil {
		return backupSemanticManifest{}, err
	}
	return r.semanticCheckWithin(ctx, "offline smoke verify")
}

func (r *restoreExecutor) createPreRestoreBackup(ctx context.Context) (*restoreBackupRecord, string, error) {
	files, err := r.backups.CreateWithMetadata(ctx, backupCreateMetadata{
		Kind:          backupKindSafety,
		SourceCommand: "restore",
		Reason:        "pre-restore rollback checkpoint",
	})
	if err != nil {
		return nil, "", fmt.Errorf("pre-restore backup failed: %w", err)
	}
	timestamp := inferTimestamp(files[0])
	if timestamp == "" {
		return nil, "", errors.New("pre-restore backup timestamp could not be inferred")
	}
	record := &restoreBackupRecord{
		Timestamp: timestamp,
		Kind:      string(backupKindSafety),
		Files:     files,
	}
	return record, timestamp, nil
}

func (r *restoreExecutor) ensureBackupWritable() error {
	dir, err := r.backups.ensureBackupDir()
	if err != nil {
		return err
	}
	probe, err := os.CreateTemp(dir, ".restore-preflight-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func (r *restoreExecutor) restoreFailure(ctx context.Context, rollbackTS string, wasRunning bool, cause error) (*restoreRollbackReport, error) {
	return r.restoreFailureWithAppState(ctx, rollbackTS, wasRunning, false, cause)
}

func (r *restoreExecutor) restoreFailureWithAppState(ctx context.Context, rollbackTS string, wasRunning bool, appRunningNow bool, cause error) (*restoreRollbackReport, error) {
	report := &restoreRollbackReport{
		Attempted: true,
		Timestamp: rollbackTS,
	}
	if appRunningNow {
		running, err := r.app.IsRunning(ctx, r.bundleID)
		if err != nil {
			report.Error = err.Error()
			return report, fmt.Errorf("%w; rollback precondition failed: %v", cause, err)
		}
		if running {
			if err := r.quiesce(ctx, true); err != nil {
				report.Error = err.Error()
				return report, fmt.Errorf("%w; rollback precondition failed: %v", cause, err)
			}
		}
	}
	_, rollbackErr := r.backups.Restore(ctx, rollbackTS)
	if rollbackErr != nil {
		report.Error = rollbackErr.Error()
		return report, fmt.Errorf("%w; rollback failed: %v", cause, rollbackErr)
	}
	report.Succeeded = true
	if wasRunning {
		if err := r.activateWithin(ctx, "rollback reopen"); err != nil {
			report.Error = err.Error()
			return report, fmt.Errorf("%w; rollback succeeded; reopen failed: %v", cause, err)
		}
	}
	return report, fmt.Errorf("%w; rollback succeeded", cause)
}

func (r *restoreExecutor) buildSemanticVerification(timestamp string, actual backupSemanticManifest) (restoreSemanticVerification, error) {
	report := restoreSemanticVerification{
		OK:     true,
		Actual: actual,
	}

	expected, err := r.backups.loadSemanticManifest(timestamp)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, fmt.Errorf("load semantic manifest %s: %w", timestamp, err)
	}
	report.Expected = &expected
	report.ComparedToManifest = true
	if err := compareSemanticManifests(expected, actual); err != nil {
		report.OK = false
		return report, err
	}
	return report, nil
}
