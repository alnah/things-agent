package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

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
