package main

import (
	"context"
	"strings"
	"testing"
)

func TestParseStateSnapshot(t *testing.T) {
	raw := strings.Join([]string{
		`A	area-1	Area A`,
		`P	project-1	Project A	open	area-1	Area A	Project note	tag-a, tag-b`,
		`T	task-1	Task A	open	area-1	Area A	project-1	Project A	2026-03-07 00:00:00	2026-03-08 00:00:00	Task note\ntwo	tag-a`,
	}, "\n")

	got, err := parseStateSnapshot(raw)
	if err != nil {
		t.Fatalf("parseStateSnapshot failed: %v", err)
	}
	if got.SchemaVersion != 1 || len(got.Areas) != 1 || len(got.Projects) != 1 || len(got.Tasks) != 1 {
		t.Fatalf("unexpected snapshot cardinality: %#v", got)
	}
	if got.Projects[0].Notes != "Project note" || strings.Join(got.Projects[0].Tags, ",") != "tag-a,tag-b" {
		t.Fatalf("unexpected project payload: %#v", got.Projects[0])
	}
	if got.Tasks[0].Notes != "Task note\ntwo" || got.Tasks[0].Project != "Project A" || got.Tasks[0].Due != "2026-03-07 00:00:00" || got.Tasks[0].Deadline != "2026-03-08 00:00:00" {
		t.Fatalf("unexpected task payload: %#v", got.Tasks[0])
	}
}

func TestParseStateSnapshotRejectsInvalidRows(t *testing.T) {
	_, err := parseStateSnapshot("T\tonly-two-fields")
	if err == nil || !strings.Contains(err.Error(), "invalid task snapshot row") {
		t.Fatalf("expected invalid task row error, got %v", err)
	}
}

func TestScriptStateSnapshotterSnapshot(t *testing.T) {
	runner := runnerFunc(func(_ context.Context, script string) (string, error) {
		if !strings.Contains(script, "state snapshot capture") {
			t.Fatalf("expected state snapshot script marker, got %s", script)
		}
		return "A\tarea-1\tArea A", nil
	})

	got, err := newScriptStateSnapshotter(defaultBundleID, runner).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(got.Areas) != 1 || got.Areas[0].Name != "Area A" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}
