package main

import (
	"strings"
	"testing"
)

func TestParseSemanticSnapshot(t *testing.T) {
	raw := strings.Join([]string{
		"L\tlist-1\tInbox",
		"P\tproject-1\tProject A\topen",
		"T\ttask-1",
		"T\ttask-2",
	}, "\n")

	got, err := parseSemanticSnapshot(raw)
	if err != nil {
		t.Fatalf("parseSemanticSnapshot failed: %v", err)
	}
	if got.ListsCount != 1 || got.ProjectsCount != 1 || got.TasksCount != 2 {
		t.Fatalf("unexpected semantic counts: %#v", got)
	}
	if got.ListsHash == "" || got.ProjectsHash == "" || got.TasksHash == "" {
		t.Fatalf("expected semantic hashes, got %#v", got)
	}
	if len(got.TaskRefs) != 2 || got.TaskRefs[0] != "task-1" || got.TaskRefs[1] != "task-2" {
		t.Fatalf("expected task refs, got %#v", got.TaskRefs)
	}
}

func TestCompareSemanticSnapshots(t *testing.T) {
	base := backupSemanticSnapshot{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	if err := compareSemanticSnapshots(base, base); err != nil {
		t.Fatalf("expected identical semantic snapshots to match: %v", err)
	}

	other := base
	other.TasksHash = "d"
	err := compareSemanticSnapshots(base, other)
	if err == nil || !strings.Contains(err.Error(), "task snapshot mismatch") {
		t.Fatalf("expected task mismatch, got %v", err)
	}
}

func TestCompareSemanticSnapshotsAllowsCountOnlyActualProbe(t *testing.T) {
	expected := backupSemanticSnapshot{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	actual := backupSemanticSnapshot{
		ListsCount:    1,
		ProjectsCount: 2,
		TasksCount:    3,
	}
	if err := compareSemanticSnapshots(expected, actual); err != nil {
		t.Fatalf("expected count-only actual probe to pass, got %v", err)
	}
}

func TestCompareSemanticSnapshotsSummarizesTaskDiffs(t *testing.T) {
	base := backupSemanticSnapshot{
		TasksCount: 2,
		TasksHash:  "a",
		TaskRefs:   []string{"task-1", "task-2"},
	}
	other := backupSemanticSnapshot{
		TasksCount: 1,
		TasksHash:  "b",
		TaskRefs:   []string{"task-2"},
	}
	err := compareSemanticSnapshots(base, other)
	if err == nil || !strings.Contains(err.Error(), "missing=[task-1]") {
		t.Fatalf("expected task diff summary, got %v", err)
	}
}

func TestParseSemanticHealthSnapshot(t *testing.T) {
	raw := strings.Join([]string{
		"L\t4",
		"P\t2",
		"T\t9",
	}, "\n")
	got, err := parseSemanticHealthSnapshot(raw)
	if err != nil {
		t.Fatalf("parseSemanticHealthSnapshot failed: %v", err)
	}
	if got.ListsCount != 4 || got.ProjectsCount != 2 || got.TasksCount != 9 {
		t.Fatalf("unexpected semantic health snapshot: %#v", got)
	}
	if got.ListsHash != "" || got.ProjectsHash != "" || got.TasksHash != "" {
		t.Fatalf("expected count-only semantic health snapshot, got %#v", got)
	}
}
