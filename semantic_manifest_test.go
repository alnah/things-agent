package main

import (
	"strings"
	"testing"
)

func TestParseSemanticManifest(t *testing.T) {
	raw := strings.Join([]string{
		"L\tlist-1\tInbox",
		"P\tproject-1\tProject A\topen",
		"T\ttask-1",
		"T\ttask-2",
	}, "\n")

	got, err := parseSemanticManifest(raw)
	if err != nil {
		t.Fatalf("parseSemanticManifest failed: %v", err)
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

func TestCompareSemanticManifests(t *testing.T) {
	base := backupSemanticManifest{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	if err := compareSemanticManifests(base, base); err != nil {
		t.Fatalf("expected identical semantic manifests to match: %v", err)
	}

	other := base
	other.TasksHash = "d"
	err := compareSemanticManifests(base, other)
	if err == nil || !strings.Contains(err.Error(), "task manifest mismatch") {
		t.Fatalf("expected task mismatch, got %v", err)
	}
}

func TestCompareSemanticManifestsAllowsCountOnlyActualProbe(t *testing.T) {
	expected := backupSemanticManifest{
		ListsCount:    1,
		ListsHash:     "a",
		ProjectsCount: 2,
		ProjectsHash:  "b",
		TasksCount:    3,
		TasksHash:     "c",
	}
	actual := backupSemanticManifest{
		ListsCount:    1,
		ProjectsCount: 2,
		TasksCount:    3,
	}
	if err := compareSemanticManifests(expected, actual); err != nil {
		t.Fatalf("expected count-only actual probe to pass, got %v", err)
	}
}

func TestCompareSemanticManifestsSummarizesTaskDiffs(t *testing.T) {
	base := backupSemanticManifest{
		TasksCount: 2,
		TasksHash:  "a",
		TaskRefs:   []string{"task-1", "task-2"},
	}
	other := backupSemanticManifest{
		TasksCount: 1,
		TasksHash:  "b",
		TaskRefs:   []string{"task-2"},
	}
	err := compareSemanticManifests(base, other)
	if err == nil || !strings.Contains(err.Error(), "missing=[task-1]") {
		t.Fatalf("expected task diff summary, got %v", err)
	}
}

func TestParseSemanticHealthManifest(t *testing.T) {
	raw := strings.Join([]string{
		"L\t4",
		"P\t2",
		"T\t9",
	}, "\n")
	got, err := parseSemanticHealthManifest(raw)
	if err != nil {
		t.Fatalf("parseSemanticHealthManifest failed: %v", err)
	}
	if got.ListsCount != 4 || got.ProjectsCount != 2 || got.TasksCount != 9 {
		t.Fatalf("unexpected semantic health manifest: %#v", got)
	}
	if got.ListsHash != "" || got.ProjectsHash != "" || got.TasksHash != "" {
		t.Fatalf("expected count-only semantic health manifest, got %#v", got)
	}
}
