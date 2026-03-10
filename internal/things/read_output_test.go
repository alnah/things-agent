package things

import (
	"strings"
	"testing"
)

func TestParseStructuredRows(t *testing.T) {
	rows, err := ParseStructuredRows("id-1\tTask A\topen\nid-2\tTask B\tcompleted\n", 3)
	if err != nil {
		t.Fatalf("ParseStructuredRows failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two rows, got %d", len(rows))
	}
	if rows[0][0] != "id-1" || rows[1][1] != "Task B" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestParseStructuredRowsRejectsWrongFieldCount(t *testing.T) {
	_, err := ParseStructuredRows("id-1\tTask A\n", 3)
	if err == nil || !strings.Contains(err.Error(), "expected 3 fields") {
		t.Fatalf("expected field count error, got %v", err)
	}
}

func TestParseShowTaskOutput(t *testing.T) {
	raw := strings.Join([]string{
		"ID: task-1",
		"Name: Task A",
		"Type: project",
		"Statut: completed",
		"Due: 2026-03-06 00:00:00",
		"Completed on: 2026-03-07 00:00:00",
		"Created on: 2026-03-01 00:00:00",
		"Tags: alpha, beta",
		"Notes: line one",
		"line two",
		"Checklist Items: unsupported via AppleScript",
		"Child Tasks:",
		"1. Review [open] (id: child-1) | note-a",
		"2. Ship [completed] (id: child-2)",
	}, "\n")

	item, err := ParseShowTaskOutput(raw)
	if err != nil {
		t.Fatalf("ParseShowTaskOutput failed: %v", err)
	}
	if item.ID != "task-1" || item.Name != "Task A" || item.Type != "project" || item.Status != "completed" {
		t.Fatalf("unexpected show-task payload: %#v", item)
	}
	if item.Due != "2026-03-06 00:00:00" || item.Completed != "2026-03-07 00:00:00" || item.Created != "2026-03-01 00:00:00" {
		t.Fatalf("unexpected date parsing: %#v", item)
	}
	if item.Notes != "line one\nline two" {
		t.Fatalf("unexpected notes: %q", item.Notes)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "alpha" || item.Tags[1] != "beta" {
		t.Fatalf("unexpected tags: %#v", item.Tags)
	}
	if item.ChecklistItemsSupported {
		t.Fatalf("expected checklist read to be explicitly unsupported, got %#v", item)
	}
	if len(item.ChildTasks) != 2 || item.ChildTasks[0].Name != "Review" || item.ChildTasks[1].Status != "completed" {
		t.Fatalf("unexpected child_tasks: %#v", item.ChildTasks)
	}
	if item.ChildTasks[0].ID != "child-1" || item.ChildTasks[1].ID != "child-2" {
		t.Fatalf("unexpected child task ids: %#v", item.ChildTasks)
	}
}

func TestParseShowTaskOutputIgnoresChildTaskUnsupportedInNotes(t *testing.T) {
	raw := strings.Join([]string{
		"ID: task-2",
		"Name: Task B",
		"Type: selected to do",
		"Statut: open",
		"Due: ",
		"Completed on: ",
		"Created on: 2026-03-01 00:00:00",
		"Tags: solo",
		"Notes: line one",
		"Checklist Items: unsupported via AppleScript",
		"Child Tasks: not supported",
	}, "\n")

	item, err := ParseShowTaskOutput(raw)
	if err != nil {
		t.Fatalf("ParseShowTaskOutput failed: %v", err)
	}
	if item.Notes != "line one" {
		t.Fatalf("unexpected notes contamination: %q", item.Notes)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "solo" {
		t.Fatalf("unexpected single tag parsing: %#v", item.Tags)
	}
	if item.ChecklistItemsSupported {
		t.Fatalf("expected checklist read to stay unsupported, got %#v", item)
	}
}
