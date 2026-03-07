package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseStructuredRows(t *testing.T) {
	rows, err := parseStructuredRows("id-1\tTask A\topen\nid-2\tTask B\tcompleted\n", 3)
	if err != nil {
		t.Fatalf("parseStructuredRows failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two rows, got %d", len(rows))
	}
	if rows[0][0] != "id-1" || rows[1][1] != "Task B" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestParseStructuredRowsRejectsWrongFieldCount(t *testing.T) {
	_, err := parseStructuredRows("id-1\tTask A\n", 3)
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

	item, err := parseShowTaskOutput(raw)
	if err != nil {
		t.Fatalf("parseShowTaskOutput failed: %v", err)
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

	item, err := parseShowTaskOutput(raw)
	if err != nil {
		t.Fatalf("parseShowTaskOutput failed: %v", err)
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

func TestReadCommandsEmitJSON(t *testing.T) {
	t.Run("projects json", func(t *testing.T) {
		fr := &fakeRunner{output: "project-1\tProject A\topen\n"}
		setupTestRuntime(t, t.TempDir(), fr)

		stdout, err := captureStdout(t, func() error {
			root := newRootCmd()
			root.SetArgs([]string{"projects", "--json"})
			return root.Execute()
		})
		if err != nil {
			t.Fatalf("projects --json failed: %v", err)
		}

		var items []readItem
		if err := json.Unmarshal([]byte(stdout), &items); err != nil {
			t.Fatalf("decode projects json: %v", err)
		}
		if len(items) != 1 || items[0].ID != "project-1" || items[0].Type != "project" {
			t.Fatalf("unexpected projects payload: %#v", items)
		}
	})

	t.Run("search json", func(t *testing.T) {
		fr := &fakeRunner{output: "task-1\tTask A\topen\n"}
		setupTestRuntime(t, t.TempDir(), fr)

		stdout, err := captureStdout(t, func() error {
			root := newRootCmd()
			root.SetArgs([]string{"search", "--query", "Task", "--json"})
			return root.Execute()
		})
		if err != nil {
			t.Fatalf("search --json failed: %v", err)
		}

		var items []readItem
		if err := json.Unmarshal([]byte(stdout), &items); err != nil {
			t.Fatalf("decode search json: %v", err)
		}
		if len(items) != 1 || items[0].ID != "task-1" || items[0].Type != "task" {
			t.Fatalf("unexpected search payload: %#v", items)
		}
	})
}

func TestRunJSONResultPropagatesRunnerError(t *testing.T) {
	cfg := &runtimeConfig{runner: &fakeRunner{err: context.Canceled}}
	err := runJSONResult(context.Background(), cfg, "script", func(_ string) (any, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected runner error")
	}
}

func TestShowTaskJSONKeepsEmptyDateFields(t *testing.T) {
	fr := &fakeRunner{output: strings.Join([]string{
		"ID: task-3",
		"Name: Task C",
		"Type: to do",
		"Statut: open",
		"Due: ",
		"Deadline: ",
		"Completed on: ",
		"Created on: 2026-03-01 00:00:00",
		"Tags: ",
		"Notes: ",
		"Checklist Items: unsupported via AppleScript",
	}, "\n")}
	setupTestRuntime(t, t.TempDir(), fr)

	stdout, err := captureStdout(t, func() error {
		root := newRootCmd()
		root.SetArgs([]string{"show-task", "--name", "Task C", "--json"})
		return root.Execute()
	})
	if err != nil {
		t.Fatalf("show-task --json failed: %v", err)
	}
	if !strings.Contains(stdout, `"due":""`) || !strings.Contains(stdout, `"deadline":""`) || !strings.Contains(stdout, `"completed":""`) {
		t.Fatalf("expected empty date fields to be preserved, got %s", stdout)
	}
}
