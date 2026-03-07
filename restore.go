package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	restorePollInterval    = 200 * time.Millisecond
	restoreStopTimeout     = 5 * time.Second
	restoreStabilityTimeout = 2 * time.Second
	restoreStablePasses    = 2
	restoreQuiesceGracePeriod = 300 * time.Millisecond
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
	backups          *backupManager
	bundleID         string
	app              appController
	sleep            func(time.Duration)
	pollInterval     time.Duration
	stopTimeout      time.Duration
	stabilityTimeout time.Duration
	stablePasses     int
	quiesceGracePeriod time.Duration
	captureFileState func(string) ([]liveFileState, error)
	semanticCheck    func(context.Context) (restoreSemanticVerification, error)
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

type restoreVerifiedFile struct {
	Name     string `json:"name"`
	Snapshot string `json:"snapshot"`
	Live     string `json:"live"`
	Match    bool   `json:"match"`
	Error    string `json:"error,omitempty"`
}

type restoreVerificationReport struct {
	Timestamp string                `json:"timestamp"`
	Match     bool                  `json:"match"`
	Complete  bool                  `json:"complete"`
	Files     []restoreVerifiedFile `json:"files"`
}

type restoreBackupRecord struct {
	Timestamp string   `json:"timestamp"`
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
	RequestedTimestamp string                    `json:"requested_timestamp,omitempty"`
	Timestamp          string                    `json:"timestamp"`
	DryRun             bool                      `json:"dry_run"`
	Outcome            string                    `json:"outcome"`
	AppWasRunning      bool                      `json:"app_was_running"`
	Preflight          restorePreflightReport    `json:"preflight"`
	PreRestoreBackup   *restoreBackupRecord      `json:"pre_restore_backup,omitempty"`
	RestoredFiles      []string                  `json:"restored_files,omitempty"`
	Verification       *restoreVerificationReport `json:"verification,omitempty"`
	SemanticVerification *restoreSemanticVerification `json:"semantic_verification,omitempty"`
	Rollback           *restoreRollbackReport    `json:"rollback,omitempty"`
	Steps              []restoreJournalStep      `json:"steps"`
}

type restoreSemanticVerification struct {
	OK               bool `json:"ok"`
	Lists            int  `json:"lists"`
	Projects         int  `json:"projects"`
	TemporaryLaunch  bool `json:"temporary_launch,omitempty"`
}

func newRestoreExecutor(cfg *runtimeConfig) *restoreExecutor {
	return &restoreExecutor{
		backups:          newBackupManager(cfg.dataDir),
		bundleID:         cfg.bundleID,
		app:              scriptAppController{runner: cfg.runner},
		sleep:            time.Sleep,
		pollInterval:     restorePollInterval,
		stopTimeout:      restoreStopTimeout,
		stabilityTimeout: restoreStabilityTimeout,
		stablePasses:     restoreStablePasses,
		quiesceGracePeriod: restoreQuiesceGracePeriod,
		captureFileState: captureLiveFileState,
		semanticCheck:    newScriptSemanticVerifier(cfg.bundleID, cfg.runner).Check,
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
	journal := restoreJournal{
		RequestedTimestamp: strings.TrimSpace(timestamp),
		DryRun:             dryRun,
		Outcome:            "failed",
	}

	preflight, err := r.Preflight(ctx, timestamp)
	journal.Preflight = preflight
	journal.Timestamp = preflight.Timestamp
	journal.AppWasRunning = preflight.AppRunning
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "preflight", Status: "failed", Error: err.Error()})
		return journal, err
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "preflight", Status: "ok"})

	if dryRun {
		journal.Outcome = "dry-run"
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "dry-run", Status: "ok"})
		return journal, nil
	}

	if err := r.quiesce(ctx, preflight.AppRunning); err != nil {
		stepStatus := "failed"
		stepName := "quiesce"
		if !preflight.AppRunning {
			stepName = "stable-files"
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: stepName, Status: stepStatus, Error: err.Error()})
		return journal, err
	}
	if preflight.AppRunning {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "quiesce", Status: "ok"})
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "stable-files", Status: "ok"})

	preRestoreBackup, err := r.backups.Create(ctx)
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "pre-restore-backup", Status: "failed", Error: err.Error()})
		return journal, fmt.Errorf("pre-restore backup failed: %w", err)
	}
	preRestoreTS := inferTimestamp(preRestoreBackup[0])
	if preRestoreTS == "" {
		err := errors.New("pre-restore backup timestamp could not be inferred")
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "pre-restore-backup", Status: "failed", Error: err.Error()})
		return journal, err
	}
	journal.PreRestoreBackup = &restoreBackupRecord{Timestamp: preRestoreTS, Files: preRestoreBackup}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "pre-restore-backup", Status: "ok"})

	restored, err := r.backups.Restore(ctx, preflight.Timestamp)
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "restore", Status: "failed", Error: err.Error()})
		rollback, restoreErr := r.restoreFailure(ctx, preRestoreTS, preflight.AppRunning, fmt.Errorf("restore snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.RestoredFiles = restored
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "restore", Status: "ok"})

	verification, err := r.Verify(ctx, preflight.Timestamp)
	journal.Verification = &verification
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "verify", Status: "failed", Error: err.Error()})
		rollback, restoreErr := r.restoreFailure(ctx, preRestoreTS, preflight.AppRunning, fmt.Errorf("verify restored snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "verify", Status: "ok"})

	if preflight.AppRunning {
		if err := r.app.Activate(ctx, r.bundleID); err != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reopen", Status: "failed", Error: err.Error()})
			return journal, err
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reopen", Status: "ok"})
	} else if err := r.app.Activate(ctx, r.bundleID); err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-launch", Status: "failed", Error: err.Error()})
		return journal, err
	} else {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-launch", Status: "ok"})
	}

	semantic, err := r.semanticCheck(ctx)
	if !preflight.AppRunning {
		semantic.TemporaryLaunch = true
	}
	journal.SemanticVerification = &semantic
	if err != nil {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-verify", Status: "failed", Error: err.Error()})
		rollback, restoreErr := r.restoreFailureWithAppState(ctx, preRestoreTS, preflight.AppRunning, true, fmt.Errorf("semantic verify restored snapshot %s: %w", preflight.Timestamp, err))
		journal.Rollback = rollback
		return journal, restoreErr
	}
	journal.Steps = append(journal.Steps, restoreJournalStep{Name: "semantic-verify", Status: "ok"})

	if !preflight.AppRunning {
		if err := r.quiesce(ctx, true); err != nil {
			journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reclose", Status: "failed", Error: err.Error()})
			return journal, err
		}
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reclose", Status: "ok"})
	} else {
		journal.Steps = append(journal.Steps, restoreJournalStep{Name: "reclose", Status: "skipped"})
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
		if err := r.quiesce(ctx, true); err != nil {
			report.Error = err.Error()
			return report, fmt.Errorf("%w; rollback precondition failed: %v", cause, err)
		}
	}
	_, rollbackErr := r.backups.Restore(ctx, rollbackTS)
	if rollbackErr != nil {
		report.Error = rollbackErr.Error()
		return report, fmt.Errorf("%w; rollback failed: %v", cause, rollbackErr)
	}
	report.Succeeded = true
	if wasRunning {
		if err := r.app.Activate(ctx, r.bundleID); err != nil {
			report.Error = err.Error()
			return report, fmt.Errorf("%w; rollback succeeded; reopen failed: %v", cause, err)
		}
	}
	return report, fmt.Errorf("%w; rollback succeeded", cause)
}

type scriptSemanticVerifier struct {
	bundleID string
	runner   scriptRunner
}

func newScriptSemanticVerifier(bundleID string, runner scriptRunner) scriptSemanticVerifier {
	return scriptSemanticVerifier{
		bundleID: bundleID,
		runner:   runner,
	}
}

func (v scriptSemanticVerifier) Check(ctx context.Context) (restoreSemanticVerification, error) {
	out, err := v.runner.run(ctx, scriptRestoreSemanticCheck(v.bundleID))
	if err != nil {
		return restoreSemanticVerification{}, fmt.Errorf("run semantic restore check: %w", err)
	}
	return parseRestoreSemanticVerification(out)
}

func parseRestoreSemanticVerification(raw string) (restoreSemanticVerification, error) {
	rows, err := parseStructuredRows(raw, 2)
	if err != nil {
		return restoreSemanticVerification{}, err
	}
	if len(rows) != 1 {
		return restoreSemanticVerification{}, errors.New("semantic restore check returned no result")
	}
	lists, err := strconv.Atoi(rows[0][0])
	if err != nil {
		return restoreSemanticVerification{}, fmt.Errorf("parse semantic list count: %w", err)
	}
	projects, err := strconv.Atoi(rows[0][1])
	if err != nil {
		return restoreSemanticVerification{}, fmt.Errorf("parse semantic project count: %w", err)
	}
	return restoreSemanticVerification{
		OK:       true,
		Lists:    lists,
		Projects: projects,
	}, nil
}

func verifySnapshotAgainstLive(dataDir string, snapshotFiles []string) error {
	report, err := buildSnapshotVerification(dataDir, snapshotFiles)
	if err != nil {
		return err
	}
	return verificationError(report)
}

func buildSnapshotVerification(dataDir string, snapshotFiles []string) (restoreVerificationReport, error) {
	report := restoreVerificationReport{
		Match:    true,
		Complete: len(snapshotFiles) == 3,
		Files:    make([]restoreVerifiedFile, 0, len(snapshotFiles)),
	}
	var firstErr error
	for _, snapshot := range snapshotFiles {
		live := filepath.Join(dataDir, liveDBBaseName(snapshot))
		fileReport := restoreVerifiedFile{
			Name:     filepath.Base(live),
			Snapshot: snapshot,
			Live:     live,
			Match:    true,
		}
		match, err := filesEqual(snapshot, live)
		if err != nil {
			fileReport.Match = false
			fileReport.Error = err.Error()
			if firstErr == nil {
				firstErr = fmt.Errorf("compare %s with %s: %w", snapshot, live, err)
			}
		} else if !match {
			fileReport.Match = false
		}
		if !fileReport.Match {
			report.Match = false
		}
		report.Files = append(report.Files, fileReport)
	}
	return report, firstErr
}

func verificationError(report restoreVerificationReport) error {
	if !report.Complete {
		return errors.New("snapshot is incomplete")
	}
	if report.Match {
		return nil
	}
	for _, file := range report.Files {
		if file.Error != "" {
			return fmt.Errorf("verification failed for %s: %s", file.Name, file.Error)
		}
		if !file.Match {
			return fmt.Errorf("live file mismatch for %s", file.Name)
		}
	}
	return errors.New("live files do not match snapshot")
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

type liveFileState struct {
	Name    string
	Size    int64
	ModTime int64
}

func captureLiveFileState(dataDir string) ([]liveFileState, error) {
	states := make([]liveFileState, 0, 3)
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		info, err := os.Stat(filepath.Join(dataDir, base))
		if err != nil {
			return nil, err
		}
		states = append(states, liveFileState{
			Name:    base,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
		})
	}
	return states, nil
}

func liveFileStatesEqual(left, right []liveFileState) bool {
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
