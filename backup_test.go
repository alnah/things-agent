package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
