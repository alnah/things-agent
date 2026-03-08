package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type backupManager struct {
	dataDir          string
	copyFn           func(src, dst string) error
	nowFn            func() time.Time
	semanticSnapshot func(context.Context) (backupSemanticSnapshot, error)
}

type backupKind string

const (
	backupKindUnknown  backupKind = "unknown"
	backupKindSession  backupKind = "session"
	backupKindExplicit backupKind = "explicit"
	backupKindSafety   backupKind = "safety"
)

type backupCreateMetadata struct {
	Kind          backupKind
	SourceCommand string
	Reason        string
}

type backupSnapshot struct {
	Timestamp     string     `json:"timestamp"`
	Kind          backupKind `json:"kind"`
	CreatedAt     string     `json:"created_at,omitempty"`
	SourceCommand string     `json:"source_command,omitempty"`
	Reason        string     `json:"reason,omitempty"`
	Complete      bool       `json:"complete"`
	Files         []string   `json:"files"`
}

type backupSemanticSnapshot struct {
	ListsCount    int      `json:"lists_count"`
	ListsHash     string   `json:"lists_hash"`
	ProjectsCount int      `json:"projects_count"`
	ProjectsHash  string   `json:"projects_hash"`
	TasksCount    int      `json:"tasks_count"`
	TasksHash     string   `json:"tasks_hash"`
	TaskRefs      []string `json:"task_refs,omitempty"`
}

func newBackupManager(dataDir string) *backupManager {
	return &backupManager{
		dataDir: dataDir,
		copyFn:  copyFile,
		nowFn:   time.Now,
	}
}

func (bm *backupManager) Create(ctx context.Context) ([]string, error) {
	return bm.CreateWithMetadata(ctx, backupCreateMetadata{
		Kind:          backupKindExplicit,
		SourceCommand: "backup",
		Reason:        "manual checkpoint",
	})
}

func (bm *backupManager) CreateWithMetadata(ctx context.Context, meta backupCreateMetadata) ([]string, error) {
	_ = ctx
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	ts, err := bm.nextTimestamp()
	if err != nil {
		return nil, err
	}
	var created []string
	if bm.packageMode() {
		dstDir := bm.snapshotDir(ts)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return nil, err
		}
		src := bm.livePrimaryDBPath()
		if _, err := os.Stat(src); err != nil {
			return nil, errors.New("no backupable database file found")
		}
		dst := filepath.Join(dstDir, "main.sqlite")
		if err := bm.copyFn(src, dst); err != nil {
			return nil, err
		}
		created = append(created, dst)
	} else {
		for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
			src := filepath.Join(bm.dataDir, base)
			if _, err := os.Stat(src); err != nil {
				continue
			}
			dst := filepath.Join(dir, base+"."+ts+".bak")
			if err := bm.copyFn(src, dst); err != nil {
				return nil, err
			}
			created = append(created, dst)
		}
	}
	if len(created) == 0 {
		return nil, errors.New("no backupable database file found")
	}
	if err := bm.writeBackupMetadata(buildBackupSnapshot(ts, created, meta)); err != nil {
		return nil, fmt.Errorf("backup created but metadata save failed: %w", err)
	}
	if bm.semanticSnapshot != nil {
		snapshot, err := bm.semanticSnapshot(ctx)
		if err != nil {
			return nil, fmt.Errorf("backup created but semantic snapshot failed: %w", err)
		}
		if err := bm.writeSemanticSnapshot(ts, snapshot); err != nil {
			return nil, fmt.Errorf("backup created but semantic snapshot save failed: %w", err)
		}
	}
	if err := bm.prune(ctx, maxBackupsToKeep); err != nil {
		return nil, fmt.Errorf("backup created but retention failed: %w", err)
	}
	sort.Strings(created)
	return created, nil
}

func (bm *backupManager) Latest(ctx context.Context) (string, error) {
	_ = ctx
	candidates, err := bm.allTimestamps()
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", errors.New("no backup available")
	}
	return candidates[0], nil
}

func (bm *backupManager) List(ctx context.Context) ([]backupSnapshot, error) {
	_ = ctx
	timestamps, err := bm.allTimestamps()
	if err != nil {
		return nil, err
	}
	snapshots := make([]backupSnapshot, 0, len(timestamps))
	for _, ts := range timestamps {
		files, err := bm.FilesForTimestamp(ctx, ts)
		if err != nil {
			return nil, err
		}
		snapshot := buildBackupSnapshot(ts, files, backupCreateMetadata{})
		if metadata, err := bm.loadBackupMetadata(ts); err == nil {
			snapshot = metadata
			snapshot.Files = append([]string(nil), files...)
			snapshot.Complete = len(files) > 0
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func (bm *backupManager) Verify(ctx context.Context, ts string) (backupSnapshot, error) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return backupSnapshot{}, errors.New("timestamp is required")
	}
	files, err := bm.FilesForTimestamp(ctx, ts)
	if err != nil {
		return backupSnapshot{}, err
	}
	if err := verifySnapshotAgainstLive(bm.dataDir, files); err != nil {
		return backupSnapshot{}, err
	}
	snapshot := buildBackupSnapshot(ts, files, backupCreateMetadata{})
	if metadata, err := bm.loadBackupMetadata(ts); err == nil {
		snapshot = metadata
		snapshot.Files = append([]string(nil), files...)
		snapshot.Complete = true
	} else if err != nil && !os.IsNotExist(err) {
		return backupSnapshot{}, err
	}
	return snapshot, nil
}

func (bm *backupManager) FilesForTimestamp(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	if bm.packageMode() {
		candidate := filepath.Join(bm.snapshotDir(ts), "main.sqlite")
		if _, err := os.Stat(candidate); err == nil {
			return []string{candidate}, nil
		} else if os.IsNotExist(err) {
			return nil, fmt.Errorf("no file for timestamp %s", ts)
		} else {
			return nil, err
		}
	}
	var paths []string
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		candidate := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no file for timestamp %s", ts)
	}
	if len(paths) != 3 {
		sort.Strings(paths)
		return nil, fmt.Errorf("incomplete snapshot for timestamp %s", ts)
	}
	sort.Strings(paths)
	return paths, nil
}

func (bm *backupManager) Restore(ctx context.Context, ts string) ([]string, error) {
	_ = ctx
	files, err := bm.FilesForTimestamp(ctx, ts)
	if err != nil {
		return nil, err
	}
	if bm.packageMode() {
		live := bm.dataDir
		tmp := live + ".restoretmp"
		if err := os.RemoveAll(tmp); err != nil {
			return nil, err
		}
		if err := copyDir(filepath.Dir(files[0]), tmp); err != nil {
			return nil, err
		}
		if err := os.RemoveAll(live); err != nil {
			return nil, err
		}
		if err := os.Rename(tmp, live); err != nil {
			return nil, err
		}
		return files, nil
	}
	for _, src := range files {
		if err := bm.RestoreFile(ctx, src); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (bm *backupManager) RestoreFile(ctx context.Context, path string) error {
	_ = ctx
	if bm.packageMode() {
		if filepath.Base(path) != "main.sqlite" {
			return fmt.Errorf("nom de backup invalide: %s", filepath.Base(path))
		}
		dst := bm.livePrimaryDBPath()
		return bm.copyFn(path, dst)
	}
	base := filepath.Base(path)
	var baseTarget string
	if strings.HasPrefix(base, "main.sqlite.") {
		baseTarget = "main.sqlite"
	} else if strings.HasPrefix(base, "main.sqlite-shm.") {
		baseTarget = "main.sqlite-shm"
	} else if strings.HasPrefix(base, "main.sqlite-wal.") {
		baseTarget = "main.sqlite-wal"
	} else {
		return fmt.Errorf("nom de backup invalide: %s", base)
	}
	dst := filepath.Join(bm.dataDir, baseTarget)
	return bm.copyFn(path, dst)
}

func (bm *backupManager) prune(ctx context.Context, keep int) error {
	_ = ctx
	if keep <= 0 {
		return nil
	}
	timestamps, err := bm.allTimestamps()
	if err != nil {
		return err
	}
	if len(timestamps) <= keep {
		return nil
	}
	for _, ts := range timestamps[keep:] {
		if bm.packageMode() {
			if err := os.RemoveAll(bm.snapshotDir(ts)); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else {
			for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
				target := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
				if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
		if err := os.Remove(bm.semanticSnapshotPath(ts)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Remove(bm.backupMetadataPath(ts)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Remove(filepath.Join(bm.backupPath(), "state."+ts+".json")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (bm *backupManager) allTimestamps() ([]string, error) {
	dir, err := bm.ensureBackupDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	tsSet := map[string]struct{}{}
	for _, e := range entries {
		ts := extractTimestamp(e.Name())
		if ts != "" {
			tsSet[ts] = struct{}{}
		}
	}
	var ts []string
	for k := range tsSet {
		ts = append(ts, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ts)))
	return ts, nil
}

func (bm *backupManager) backupPath() string {
	root := bm.dataDir
	if bm.packageMode() {
		root = filepath.Dir(bm.dataDir)
	}
	return filepath.Join(root, backupDirName)
}

func (bm *backupManager) semanticSnapshotPath(ts string) string {
	return filepath.Join(bm.backupPath(), "manifest."+ts+".json")
}

func (bm *backupManager) backupMetadataPath(ts string) string {
	return filepath.Join(bm.backupPath(), "index."+ts+".json")
}

func (bm *backupManager) ensureBackupDir() (string, error) {
	path := bm.backupPath()
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func (bm *backupManager) nextTimestamp() (string, error) {
	current := bm.nowFn()
	timestamps, err := bm.allTimestamps()
	if err != nil {
		return "", err
	}
	if len(timestamps) > 0 {
		latest, err := time.ParseInLocation(backupTSFormat, timestamps[0], time.Local)
		if err == nil && !current.After(latest) {
			current = latest.Add(time.Second)
		}
	}
	for i := 0; i < 60; i++ {
		ts := current.Format(backupTSFormat)
		exists, err := bm.timestampExists(ts)
		if err != nil {
			return "", err
		}
		if !exists {
			return ts, nil
		}
		current = current.Add(time.Second)
	}
	return "", errors.New("could not allocate unique backup timestamp")
}

func (bm *backupManager) timestampExists(ts string) (bool, error) {
	if bm.packageMode() {
		if _, err := os.Stat(bm.snapshotDir(ts)); err == nil {
			return true, nil
		} else if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		target := filepath.Join(bm.backupPath(), base+"."+ts+".bak")
		if _, err := os.Stat(target); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

func (bm *backupManager) writeSemanticSnapshot(ts string, snapshot backupSemanticSnapshot) error {
	path := bm.semanticSnapshotPath(ts)
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (bm *backupManager) loadSemanticSnapshot(ts string) (backupSemanticSnapshot, error) {
	data, err := os.ReadFile(bm.semanticSnapshotPath(ts))
	if err != nil {
		return backupSemanticSnapshot{}, err
	}
	var snapshot backupSemanticSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return backupSemanticSnapshot{}, err
	}
	return snapshot, nil
}

func buildBackupSnapshot(ts string, files []string, meta backupCreateMetadata) backupSnapshot {
	meta = normalizeBackupCreateMetadata(meta)
	createdAt := ""
	if parsed, err := time.ParseInLocation(backupTSFormat, ts, time.Local); err == nil {
		createdAt = parsed.Format(time.RFC3339)
	}
	copiedFiles := append([]string(nil), files...)
	sort.Strings(copiedFiles)
	return backupSnapshot{
		Timestamp:     ts,
		Kind:          meta.Kind,
		CreatedAt:     createdAt,
		SourceCommand: meta.SourceCommand,
		Reason:        meta.Reason,
		Complete:      len(copiedFiles) > 0,
		Files:         copiedFiles,
	}
}

func normalizeBackupCreateMetadata(meta backupCreateMetadata) backupCreateMetadata {
	meta.Kind = backupKind(strings.TrimSpace(string(meta.Kind)))
	meta.SourceCommand = strings.TrimSpace(meta.SourceCommand)
	meta.Reason = strings.TrimSpace(meta.Reason)
	if meta.Kind == "" {
		meta.Kind = backupKindUnknown
	}
	switch meta.Kind {
	case backupKindSession:
		if meta.SourceCommand == "" {
			meta.SourceCommand = "session-start"
		}
		if meta.Reason == "" {
			meta.Reason = "session bootstrap checkpoint"
		}
	case backupKindSafety:
		if meta.SourceCommand == "" {
			meta.SourceCommand = "auto-safety"
		}
		if meta.Reason == "" {
			meta.Reason = "automatic rollback checkpoint"
		}
	case backupKindExplicit:
		if meta.SourceCommand == "" {
			meta.SourceCommand = "backup"
		}
		if meta.Reason == "" {
			meta.Reason = "manual checkpoint"
		}
	default:
		meta.Kind = backupKindUnknown
	}
	return meta
}

func (bm *backupManager) writeBackupMetadata(snapshot backupSnapshot) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return os.WriteFile(bm.backupMetadataPath(snapshot.Timestamp), data, 0o644)
}

func (bm *backupManager) loadBackupMetadata(ts string) (backupSnapshot, error) {
	data, err := os.ReadFile(bm.backupMetadataPath(ts))
	if err != nil {
		return backupSnapshot{}, err
	}
	var snapshot backupSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return backupSnapshot{}, err
	}
	return snapshot, nil
}

func hashSemanticLines(lines []string) string {
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func parseToAppleDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	t, err := parseDate(value)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

func parseDate(v string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", v)
}

func inferTimestamp(file string) string {
	base := filepath.Base(file)
	if base == "main.sqlite" {
		parent := filepath.Base(filepath.Dir(file))
		if strings.HasPrefix(parent, "snapshot.") && strings.HasSuffix(parent, ".thingsdatabase") {
			return strings.TrimSuffix(strings.TrimPrefix(parent, "snapshot."), ".thingsdatabase")
		}
	}
	if strings.HasPrefix(base, "snapshot.") && strings.HasSuffix(base, ".thingsdatabase") {
		return strings.TrimSuffix(strings.TrimPrefix(base, "snapshot."), ".thingsdatabase")
	}
	candidates := []string{
		"main.sqlite.",
		"main.sqlite-shm.",
		"main.sqlite-wal.",
	}
	for _, p := range candidates {
		if strings.HasPrefix(base, p) && strings.HasSuffix(base, ".bak") {
			return strings.TrimSuffix(strings.TrimPrefix(base, p), ".bak")
		}
	}
	return ""
}

func extractTimestamp(file string) string {
	base := filepath.Base(file)
	return inferTimestamp(base)
}

func (bm *backupManager) packageMode() bool {
	return strings.HasSuffix(strings.ToLower(filepath.Base(bm.dataDir)), ".thingsdatabase")
}

func (bm *backupManager) livePrimaryDBPath() string {
	return filepath.Join(bm.dataDir, "main.sqlite")
}

func (bm *backupManager) snapshotDir(ts string) string {
	return filepath.Join(bm.backupPath(), "snapshot."+ts+".thingsdatabase")
}

func clearRestoreSyncMetadata(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	stmts := []string{
		"DELETE FROM BSSyncronyMetadata",
		"DELETE FROM ThingsTouch_ExtensionCommandStore_Commands",
		"DELETE FROM ThingsTouch_ExtensionCommandStore_Meta",
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
