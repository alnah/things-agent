package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDateSupportsExpectedFormats(t *testing.T) {

	cases := []string{
		"2026-03-06",
		"2026-03-06 14:05",
		"2026-03-06 14:05:06",
		"2026-03-06T14:05:06Z",
		"06/03/2026",
		"06/03/2026 14:05",
		"06/03/2026 14:05:06",
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			if _, err := parseDate(tc); err != nil {
				t.Fatalf("parseDate(%q) returned error: %v", tc, err)
			}
		})
	}
}

func TestParseDateRejectsInvalidFormat(t *testing.T) {
	if _, err := parseDate("2026/03/06"); err == nil {
		t.Fatal("expected error for unsupported date format")
	}
}

func TestParseToAppleDateFormatsOutput(t *testing.T) {
	got, err := parseToAppleDate("2026-03-06 14:05:06")
	if err != nil {
		t.Fatalf("parseToAppleDate returned error: %v", err)
	}
	want := "2026-03-06 14:05:06"
	if got != want {
		t.Fatalf("parseToAppleDate output mismatch: got %q want %q", got, want)
	}
}

func TestInferTimestamp(t *testing.T) {
	got := inferTimestamp("main.sqlite.2026-03-06:14-05-06.bak")
	if got != "2026-03-06:14-05-06" {
		t.Fatalf("inferTimestamp mismatch: got %q", got)
	}
	if inferTimestamp("invalid.bak") != "" {
		t.Fatal("expected empty timestamp for invalid backup name")
	}
}

func TestBackupManagerCreateAndLatest(t *testing.T) {

	tmp := t.TempDir()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		if err := os.WriteFile(filepath.Join(tmp, base), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", base, err)
		}
	}

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 backup files, got %d", len(created))
	}
	for _, path := range created {
		if !strings.Contains(path, filepath.Join(tmp, backupDirName)) {
			t.Fatalf("backup path not in backup dir: %s", path)
		}
	}

	ts, err := bm.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	if ts == "" {
		t.Fatal("expected non-empty latest timestamp")
	}
}

func TestBackupManagerListAndVerify(t *testing.T) {
	tmp := t.TempDir()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		if err := os.WriteFile(filepath.Join(tmp, base), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", base, err)
		}
	}

	bm := newBackupManager(tmp)
	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	ts := inferTimestamp(created[0])

	snapshots, err := bm.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].Timestamp != ts || !snapshots[0].Complete {
		t.Fatalf("unexpected snapshots: %#v", snapshots)
	}

	snapshot, err := bm.Verify(context.Background(), ts)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if snapshot.Timestamp != ts || !snapshot.Complete || len(snapshot.Files) != 3 {
		t.Fatalf("unexpected verified snapshot: %#v", snapshot)
	}
}

func TestBackupManagerCreateWritesSemanticManifest(t *testing.T) {
	tmp := t.TempDir()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		if err := os.WriteFile(filepath.Join(tmp, base), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", base, err)
		}
	}

	bm := newBackupManager(tmp)
	expected := backupSemanticSnapshot{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	bm.semanticSnapshot = func(context.Context) (backupSemanticSnapshot, error) {
		return expected, nil
	}

	created, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	ts := inferTimestamp(created[0])

	got, err := bm.loadSemanticSnapshot(ts)
	if err != nil {
		t.Fatalf("loadSemanticSnapshot failed: %v", err)
	}
	if got.ListsCount != expected.ListsCount || got.ListsHash != expected.ListsHash || got.ProjectsCount != expected.ProjectsCount || got.ProjectsHash != expected.ProjectsHash || got.TasksCount != expected.TasksCount || got.TasksHash != expected.TasksHash || strings.Join(got.TaskRefs, ",") != strings.Join(expected.TaskRefs, ",") {
		t.Fatalf("unexpected semantic manifest: got %#v want %#v", got, expected)
	}
}

func TestBackupManagerCreateAvoidsTimestampCollisions(t *testing.T) {
	tmp := t.TempDir()
	for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
		if err := os.WriteFile(filepath.Join(tmp, base), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", base, err)
		}
	}

	bm := newBackupManager(tmp)
	fixedNow := time.Date(2026, 3, 6, 21, 10, 0, 0, time.Local)
	bm.nowFn = func() time.Time { return fixedNow }

	first, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := bm.Create(context.Background())
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}

	firstTS := inferTimestamp(first[0])
	secondTS := inferTimestamp(second[0])
	if firstTS == secondTS {
		t.Fatalf("expected unique timestamps, got %q and %q", firstTS, secondTS)
	}
}

func TestBackupManagerPruneKeepsNewestTimestamps(t *testing.T) {

	tmp := t.TempDir()
	bm := newBackupManager(tmp)
	backupPath, err := bm.ensureBackupDir()
	if err != nil {
		t.Fatalf("ensureBackupDir failed: %v", err)
	}

	timestamps := []string{
		"2026-03-06:10-00-00",
		"2026-03-06:11-00-00",
		"2026-03-06:12-00-00",
	}
	for _, ts := range timestamps {
		for _, base := range []string{"main.sqlite", "main.sqlite-shm", "main.sqlite-wal"} {
			p := filepath.Join(backupPath, base+"."+ts+".bak")
			if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
				t.Fatalf("seed backup %s failed: %v", p, err)
			}
		}
	}

	if err := bm.prune(context.Background(), 2); err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupPath, "main.sqlite.2026-03-06:10-00-00.bak")); !os.IsNotExist(err) {
		t.Fatalf("oldest timestamp should be pruned, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(backupPath, "main.sqlite.2026-03-06:12-00-00.bak")); err != nil {
		t.Fatalf("newest timestamp should remain, stat err=%v", err)
	}
}

func TestBackupManagerErrorsAndRestoreFileBranches(t *testing.T) {
	tmp := t.TempDir()
	bm := newBackupManager(tmp)

	if _, err := bm.Create(context.Background()); err == nil || !strings.Contains(err.Error(), "no backupable database file found") {
		t.Fatalf("expected create no-file error, got: %v", err)
	}
	if _, err := bm.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "no backup available") {
		t.Fatalf("expected latest no-backup error, got: %v", err)
	}
	if _, err := bm.FilesForTimestamp(context.Background(), "2026-01-01:00-00-00"); err == nil {
		t.Fatal("expected files-for-timestamp error")
	}
	if err := bm.RestoreFile(context.Background(), filepath.Join(tmp, "invalid.bak")); err == nil {
		t.Fatal("expected invalid backup name error")
	}
}

func TestBackupManagerFilesForTimestampRequiresCompleteTrio(t *testing.T) {
	tmp := t.TempDir()
	bm := newBackupManager(tmp)
	backupPath, err := bm.ensureBackupDir()
	if err != nil {
		t.Fatalf("ensureBackupDir failed: %v", err)
	}

	ts := "2026-03-06:12-00-00"
	for _, base := range []string{"main.sqlite", "main.sqlite-shm"} {
		p := filepath.Join(backupPath, base+"."+ts+".bak")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("seed backup %s failed: %v", p, err)
		}
	}

	_, err = bm.FilesForTimestamp(context.Background(), ts)
	if err == nil || !strings.Contains(err.Error(), "incomplete snapshot") {
		t.Fatalf("expected incomplete snapshot error, got %v", err)
	}
}

func TestBackupManagerPruneKeepZeroNoop(t *testing.T) {
	tmp := t.TempDir()
	bm := newBackupManager(tmp)
	if err := bm.prune(context.Background(), 0); err != nil {
		t.Fatalf("prune keep=0 should be noop: %v", err)
	}
}

func TestCopyFileAndEnsureBackupDirErrorBranches(t *testing.T) {
	tmp := t.TempDir()

	if err := copyFile(filepath.Join(tmp, "missing"), filepath.Join(tmp, "out")); err == nil {
		t.Fatal("expected copyFile error when source is missing")
	}

	fileAsDir := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	bm := newBackupManager(fileAsDir)
	if _, err := bm.ensureBackupDir(); err == nil {
		t.Fatal("expected ensureBackupDir error when dataDir is a file")
	}
}
