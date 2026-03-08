package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunResultBranches(t *testing.T) {
	cfg := &runtimeConfig{
		runner: &fakeRunner{output: ""},
	}
	if err := runResult(context.Background(), cfg, "script"); err != nil {
		t.Fatalf("runResult with empty output should succeed: %v", err)
	}

	cfgErr := &runtimeConfig{
		runner: &fakeRunner{err: errors.New("boom")},
	}
	if err := runResult(context.Background(), cfgErr, "script"); err == nil {
		t.Fatal("runResult should return runner error")
	}
}

func TestBackupIfNeededIsNoOp(t *testing.T) {
	ctx := context.Background()

	cfg := &runtimeConfig{dataDir: t.TempDir()}
	if err := backupIfNeeded(ctx, cfg); err != nil {
		t.Fatalf("backupIfNeeded should be a no-op: %v", err)
	}
}

func TestBackupIfDestructiveBranches(t *testing.T) {
	ctx := context.Background()

	cfgErr := &runtimeConfig{dataDir: t.TempDir()}
	if err := backupIfDestructive(ctx, cfgErr); err == nil {
		t.Fatal("backupIfDestructive should fail without backupable db files")
	}

	fr := &fakeRunner{}
	tmp := setupTestRuntimeWithDB(t, fr)
	cfgOK := &runtimeConfig{dataDir: tmp}
	if err := backupIfDestructive(ctx, cfgOK); err != nil {
		t.Fatalf("backupIfDestructive should succeed with db files: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(tmp, backupDirName))
	if err != nil {
		t.Fatalf("ReadDir backup dir failed: %v", err)
	}
	var foundIndex bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "index.") && filepath.Ext(entry.Name()) == ".json" {
			foundIndex = true
		}
		if strings.HasPrefix(entry.Name(), "manifest.") && filepath.Ext(entry.Name()) == ".json" {
			t.Fatalf("destructive auto backup should not write semantic manifests, got %s", entry.Name())
		}
	}
	if !foundIndex {
		t.Fatal("destructive auto backup should write an index manifest")
	}
	manager := newBackupManager(tmp)
	ts, err := manager.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest failed: %v", err)
	}
	metadata, err := manager.loadBackupMetadata(ts)
	if err != nil {
		t.Fatalf("loadBackupMetadata failed: %v", err)
	}
	if metadata.Kind != backupKindSafety || metadata.Reason != "automatic rollback checkpoint" {
		t.Fatalf("unexpected destructive backup metadata: %#v", metadata)
	}
}
