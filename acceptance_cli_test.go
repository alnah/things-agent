package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	runErr := fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close stdout reader: %v", err)
	}
	return string(out), runErr
}

func executeAcceptanceRoot(t *testing.T, args ...string) error {
	t.Helper()

	root := newRootCmd()
	root.SetArgs(args)
	return root.Execute()
}

func TestAcceptanceCLIContracts(t *testing.T) {
	t.Run("create commands require explicit destination", func(t *testing.T) {
		t.Run("add-task rejects missing destination", func(t *testing.T) {
			fr := &fakeRunner{output: "task-id-1"}
			setupTestRuntimeWithDB(t, fr)
			t.Setenv("THINGS_DEFAULT_LIST", "")

			err := executeAcceptanceRoot(t, "add-task", "--name", "task-a")
			if err == nil || !strings.Contains(err.Error(), "destination is required") {
				t.Fatalf("expected explicit destination error, got %v", err)
			}
		})

		t.Run("add-project rejects missing destination", func(t *testing.T) {
			fr := &fakeRunner{output: "project-id-1"}
			setupTestRuntimeWithDB(t, fr)
			t.Setenv("THINGS_DEFAULT_LIST", "")

			err := executeAcceptanceRoot(t, "add-project", "--name", "project-a")
			if err == nil || !strings.Contains(err.Error(), "destination is required") {
				t.Fatalf("expected explicit destination error, got %v", err)
			}
		})
	})

	t.Run("add-task honors env destination and quoted checklist CSV", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_DEFAULT_LIST", "Inbox")

		stdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t,
				"add-task",
				"--name", "task-a",
				"--checklist-items", `"one, first","two"`,
			)
		})
		if err != nil {
			t.Fatalf("expected add-task to succeed with env destination: %v", err)
		}
		if !strings.Contains(stdout, "task-id-1") {
			t.Fatalf("expected created task id on stdout, got %q", stdout)
		}

		scripts := fr.allScripts()
		if len(scripts) < 2 {
			t.Fatalf("expected create and checklist scripts, got %d", len(scripts))
		}
		if !strings.Contains(scripts[0], `set targetList to first list whose name is "Inbox"`) {
			t.Fatalf("expected env destination in create script, got %s", scripts[0])
		}
		if !strings.Contains(scripts[1], "one%2C%20first%0Atwo") {
			t.Fatalf("expected quoted CSV checklist item to be preserved, got %s", scripts[1])
		}
	})

	t.Run("add-task accepts explicit project destination", func(t *testing.T) {
		fr := &fakeRunner{output: "task-id-1"}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_DEFAULT_LIST", "")

		err := executeAcceptanceRoot(t, "add-task", "--name", "task-a", "--project", "Project A")
		if err != nil {
			t.Fatalf("expected add-task --project to succeed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[0], `set targetProject to first project whose name is "Project A"`) {
			t.Fatalf("expected project destination script, got %#v", scripts)
		}
	})

	t.Run("url search allows missing query", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "url", "search"); err != nil {
			t.Fatalf("expected bare url search to succeed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `open location "things:///search"`) {
			t.Fatalf("expected bare search endpoint, got %s", scripts)
		}
	})

	t.Run("url json uses canonical endpoint and structural auth gate", func(t *testing.T) {
		t.Run("non-update payload uses json endpoint without token gate", func(t *testing.T) {
			fr := &fakeRunner{output: "ok"}
			setupTestRuntimeWithDB(t, fr)

			err := executeAcceptanceRoot(t, "url", "json", "--data", `{"items":[{"title":"operation:update"}]}`)
			if err != nil {
				t.Fatalf("expected non-update payload to succeed without structural token requirement: %v", err)
			}

			scripts := strings.Join(fr.allScripts(), "\n")
			if !strings.Contains(scripts, "things:///json?") {
				t.Fatalf("expected canonical json endpoint, got %s", scripts)
			}
		})

		t.Run("update payload requires auth token", func(t *testing.T) {
			fr := &fakeRunner{output: "ok"}
			setupTestRuntimeWithDB(t, fr)
			t.Setenv("THINGS_AUTH_TOKEN", "")
			config.authToken = ""

			err := executeAcceptanceRoot(t, "url", "json", "--data", `{"operation":"update","items":[]}`)
			if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
				t.Fatalf("expected auth-token gate for update payload, got %v", err)
			}
		})
	})

	t.Run("set-task-date splits due and deadline backends", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		err := executeAcceptanceRoot(t,
			"set-task-date",
			"--name", "task-a",
			"--due", "2026-03-06",
			"--deadline", "2026-03-07",
		)
		if err != nil {
			t.Fatalf("expected set-task-date to succeed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `set due date of t to date "2026-03-06 00:00:00"`) {
			t.Fatalf("expected due date AppleScript mutation, got %s", scripts)
		}
		if !strings.Contains(scripts, "things:///update?auth-token=token-test") || !strings.Contains(scripts, "&deadline=2026-03-07%2000%3A00%3A00") {
			t.Fatalf("expected deadline URL mutation, got %s", scripts)
		}
	})

	t.Run("list-child-tasks surfaces backend status markers", func(t *testing.T) {
		fr := &fakeRunner{output: "status:unsupported\ncode:-1708\nmessage:event not handled"}
		setupTestRuntime(t, t.TempDir(), fr)

		stdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "list-child-tasks", "--parent", "task-a")
		})
		if err != nil {
			t.Fatalf("expected list-child-tasks to surface backend marker instead of failing silently: %v", err)
		}
		if !strings.Contains(stdout, "status:unsupported") || !strings.Contains(stdout, "message:event not handled") {
			t.Fatalf("expected backend status markers on stdout, got %q", stdout)
		}
	})

	t.Run("read commands support stable json output", func(t *testing.T) {
		t.Run("tasks json returns machine fields", func(t *testing.T) {
			fr := &fakeRunner{output: "task-1\tTask A\topen\n"}
			setupTestRuntime(t, t.TempDir(), fr)

			stdout, err := captureStdout(t, func() error {
				return executeAcceptanceRoot(t, "tasks", "--json")
			})
			if err != nil {
				t.Fatalf("expected tasks --json to succeed: %v", err)
			}

			var items []map[string]any
			if err := json.Unmarshal([]byte(stdout), &items); err != nil {
				t.Fatalf("decode tasks json: %v\nstdout=%q", err, stdout)
			}
			if len(items) != 1 {
				t.Fatalf("expected one task, got %d", len(items))
			}
			if items[0]["id"] != "task-1" || items[0]["name"] != "Task A" || items[0]["type"] != "task" || items[0]["status"] != "open" {
				t.Fatalf("unexpected task json payload: %#v", items[0])
			}
		})

		t.Run("show-task json preserves notes and child_tasks", func(t *testing.T) {
			fr := &fakeRunner{output: strings.Join([]string{
				"ID: task-1",
				"Name: Task A",
				"Type: to do",
				"Statut: open",
				"Due: 2026-03-06 00:00:00",
				"Completed on: ",
				"Created on: 2026-03-01 00:00:00",
				"Tags: alpha, beta",
				"Notes: line one",
				"line two",
				"Child Tasks:",
				"1. Review [open] | note-a",
				"2. Ship [completed]",
			}, "\n")}
			setupTestRuntime(t, t.TempDir(), fr)

			stdout, err := captureStdout(t, func() error {
				return executeAcceptanceRoot(t, "show-task", "--name", "Task A", "--json")
			})
			if err != nil {
				t.Fatalf("expected show-task --json to succeed: %v", err)
			}

			var item map[string]any
			if err := json.Unmarshal([]byte(stdout), &item); err != nil {
				t.Fatalf("decode show-task json: %v\nstdout=%q", err, stdout)
			}
			if item["id"] != "task-1" || item["type"] != "task" || item["status"] != "open" {
				t.Fatalf("unexpected show-task json payload: %#v", item)
			}
			if item["notes"] != "line one\nline two" {
				t.Fatalf("expected multiline notes, got %#v", item["notes"])
			}
			childTasks, ok := item["child_tasks"].([]any)
			if !ok || len(childTasks) != 2 {
				t.Fatalf("expected two child_tasks, got %#v", item["child_tasks"])
			}
		})
	})

	t.Run("write commands support id selectors", func(t *testing.T) {
		fr := &fakeRunner{output: "task-1"}
		setupTestRuntimeWithDB(t, fr)

		err := executeAcceptanceRoot(t, "complete-task", "--id", "task-1")
		if err != nil {
			t.Fatalf("expected complete-task --id to succeed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[0], `set tid to "task-1"`) || !strings.Contains(scripts[0], `&completed=true`) {
			t.Fatalf("expected id-based task resolution, got %#v", scripts)
		}
	})

	t.Run("add-checklist-item supports task-id selector", func(t *testing.T) {
		fr := &fakeRunner{output: "task-1"}
		setupTestRuntimeWithDB(t, fr)

		err := executeAcceptanceRoot(t, "add-checklist-item", "--task-id", "task-1", "--name", "Checklist item")
		if err != nil {
			t.Fatalf("expected add-checklist-item --task-id to succeed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[len(scripts)-1], `every to do whose id is "task-1"`) {
			t.Fatalf("expected task-id based checklist mutation, got %#v", scripts)
		}
	})

	t.Run("child task commands support parent-id selector", func(t *testing.T) {
		fr := &fakeRunner{output: "task-1"}
		setupTestRuntimeWithDB(t, fr)

		err := executeAcceptanceRoot(t, "add-child-task", "--parent-id", "task-1", "--name", "Child Task")
		if err != nil {
			t.Fatalf("expected add-child-task --parent-id to succeed: %v", err)
		}
		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[len(scripts)-1], `set totalCount to projectCount + taskCount`) {
			t.Fatalf("expected parent-id based child task mutation, got %#v", scripts)
		}
	})

	t.Run("legacy ambiguous checklist and subtask surface is rejected", func(t *testing.T) {
		fr := &fakeRunner{output: "task-1"}
		setupTestRuntimeWithDB(t, fr)

		err := executeAcceptanceRoot(t, "add-subtask", "--task-id", "task-1", "--name", "Checklist item")
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("expected add-subtask legacy command to be rejected, got %v", err)
		}

		err = executeAcceptanceRoot(t, "edit-checklist-item", "--task-id", "task-1", "--name", "one", "--new-name", "two")
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("expected unsupported checklist item mutation command to be rejected, got %v", err)
		}

		err = executeAcceptanceRoot(t, "add-task", "--name", "task-a", "--area", "Inbox", "--subtasks", "one,two")
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --subtasks") {
			t.Fatalf("expected legacy --subtasks flag to be rejected, got %v", err)
		}

		err = executeAcceptanceRoot(t, "show-task", "--name", "task-a", "--with-checklist-items=false")
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --with-checklist-items") {
			t.Fatalf("expected old checklist view flag to be rejected, got %v", err)
		}
	})

	t.Run("restore is timestamp-only on the CLI surface", func(t *testing.T) {
		fr := &fakeRunner{}
		tmp := setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "backup"); err != nil {
			t.Fatalf("expected backup to succeed: %v", err)
		}
		if err := executeAcceptanceRoot(t, "restore"); err != nil {
			t.Fatalf("expected restore latest to succeed: %v", err)
		}

		err := executeAcceptanceRoot(t, "restore", "--file", tmp)
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --file") {
			t.Fatalf("expected unsupported --file flag, got %v", err)
		}
	})

	t.Run("restore discovery and verification expose deterministic state", func(t *testing.T) {
		fr := &fakeRunner{}
		tmp := setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "backup"); err != nil {
			t.Fatalf("expected backup to succeed: %v", err)
		}

		entries, err := os.ReadDir(tmp + "/" + backupDirName)
		if err != nil || len(entries) == 0 {
			t.Fatalf("expected at least one backup entry, err=%v count=%d", err, len(entries))
		}
		ts := inferTimestamp(entries[0].Name())

		listStdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "restore", "list", "--json")
		})
		if err != nil {
			t.Fatalf("expected restore list to succeed: %v", err)
		}

		var snapshots []map[string]any
		if err := json.Unmarshal([]byte(listStdout), &snapshots); err != nil {
			t.Fatalf("decode restore list json: %v\nstdout=%q", err, listStdout)
		}
		if len(snapshots) == 0 || snapshots[0]["timestamp"] == "" || snapshots[0]["complete"] != true {
			t.Fatalf("unexpected restore list payload: %#v", snapshots)
		}

		verifyStdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "restore", "verify", "--timestamp", ts, "--json")
		})
		if err != nil {
			t.Fatalf("expected restore verify to succeed: %v", err)
		}

		var verify map[string]any
		if err := json.Unmarshal([]byte(verifyStdout), &verify); err != nil {
			t.Fatalf("decode restore verify json: %v\nstdout=%q", err, verifyStdout)
		}
		if verify["timestamp"] != ts || verify["match"] != true || verify["complete"] != true {
			t.Fatalf("unexpected restore verify payload: %#v", verify)
		}
		files, ok := verify["files"].([]any)
		if !ok || len(files) != 3 {
			t.Fatalf("expected three per-file verification entries, got %#v", verify["files"])
		}
		first, ok := files[0].(map[string]any)
		if !ok || first["name"] == "" || first["match"] != true {
			t.Fatalf("unexpected per-file verification entry: %#v", files[0])
		}
	})

	t.Run("restore preflight reports readiness as structured json", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "backup"); err != nil {
			t.Fatalf("expected backup to succeed: %v", err)
		}

		stdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "restore", "preflight", "--json")
		})
		if err != nil {
			t.Fatalf("expected restore preflight to succeed: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode restore preflight json: %v\nstdout=%q", err, stdout)
		}
		if payload["ok"] != true || payload["complete"] != true || payload["live_files_present"] != true || payload["backup_writable"] != true {
			t.Fatalf("unexpected restore preflight payload: %#v", payload)
		}
	})

	t.Run("restore dry-run emits structured journal without mutating live files", func(t *testing.T) {
		fr := &fakeRunner{}
		tmp := setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "backup"); err != nil {
			t.Fatalf("expected backup to succeed: %v", err)
		}
		entries, err := os.ReadDir(tmp + "/" + backupDirName)
		if err != nil || len(entries) == 0 {
			t.Fatalf("expected at least one backup entry, err=%v count=%d", err, len(entries))
		}
		ts := inferTimestamp(entries[0].Name())
		before := readLiveDBFile(t, tmp, "main.sqlite")

		stdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "restore", "--timestamp", ts, "--dry-run", "--json")
		})
		if err != nil {
			t.Fatalf("expected restore --dry-run to succeed: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode restore dry-run json: %v\nstdout=%q", err, stdout)
		}
		if payload["dry_run"] != true || payload["outcome"] != "dry-run" || payload["timestamp"] != ts {
			t.Fatalf("unexpected restore dry-run payload: %#v", payload)
		}
		preflight, ok := payload["preflight"].(map[string]any)
		if !ok || preflight["ok"] != true {
			t.Fatalf("expected successful preflight in restore journal, got %#v", payload["preflight"])
		}
		steps, ok := payload["steps"].([]any)
		if !ok || len(steps) == 0 {
			t.Fatalf("expected restore journal steps, got %#v", payload["steps"])
		}
		if after := readLiveDBFile(t, tmp, "main.sqlite"); after != before {
			t.Fatalf("expected dry-run to keep live database untouched: before=%q after=%q", before, after)
		}
	})

	t.Run("restore json includes semantic verification details", func(t *testing.T) {
		fr := &fakeRunner{}
		tmp := setupTestRuntimeWithDB(t, fr)

		if err := executeAcceptanceRoot(t, "backup"); err != nil {
			t.Fatalf("expected backup to succeed: %v", err)
		}
		entries, err := os.ReadDir(tmp + "/" + backupDirName)
		if err != nil || len(entries) == 0 {
			t.Fatalf("expected at least one backup entry, err=%v count=%d", err, len(entries))
		}
		ts := inferTimestamp(entries[0].Name())

		stdout, err := captureStdout(t, func() error {
			return executeAcceptanceRoot(t, "restore", "--timestamp", ts, "--json")
		})
		if err != nil {
			t.Fatalf("expected restore --json to succeed: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode restore json: %v\nstdout=%q", err, stdout)
		}
		semantic, ok := payload["semantic_verification"].(map[string]any)
		if !ok || semantic["ok"] != true {
			t.Fatalf("expected semantic verification in restore journal, got %#v", payload["semantic_verification"])
		}
	})
}
