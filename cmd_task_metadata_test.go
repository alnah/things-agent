package main

import (
	"strings"
	"testing"
)

func TestTaskMetadataCommands(t *testing.T) {
	t.Run("tag commands success", func(t *testing.T) {
		fr := &fakeRunner{runFn: func(script string) (string, error) {
			if strings.Contains(script, `set out to "ID: "`) {
				return strings.Join([]string{
					"ID: task-1",
					"Name: task-a",
					"Type: task",
					"Statut: open",
					"Due: ",
					"Completed on: ",
					"Created on: ",
					"Tags: a",
					"Notes: ",
				}, "\n"), nil
			}
			return "ok", nil
		}}
		setupTestRuntimeWithDB(t, fr)

		setTags := newSetTagsCmd()
		setTags.SetArgs([]string{"--name", "task-a", "--tags", "a,b"})
		if err := setTags.Execute(); err != nil {
			t.Fatalf("set-tags failed: %v", err)
		}

		setTaskTags := newSetTaskTagsCmd()
		setTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "a,b"})
		if err := setTaskTags.Execute(); err != nil {
			t.Fatalf("set-task-tags failed: %v", err)
		}

		addTaskTags := newAddTaskTagsCmd()
		addTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "c"})
		if err := addTaskTags.Execute(); err != nil {
			t.Fatalf("add-task-tags failed: %v", err)
		}

		removeTaskTags := newRemoveTaskTagsCmd()
		removeTaskTags.SetArgs([]string{"--name", "task-a", "--tags", "a"})
		if err := removeTaskTags.Execute(); err != nil {
			t.Fatalf("remove-task-tags failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) < 6 {
			t.Fatalf("expected at least 6 scripts, got %d", len(scripts))
		}
		joined := strings.Join(scripts, "\n")
		if !strings.Contains(joined, `set tag names of t to "a, b"`) {
			t.Fatalf("expected exact tag set script, got %s", joined)
		}
		if !strings.Contains(joined, "things:///update?auth-token=token-test") || !strings.Contains(joined, "add-tags=c") {
			t.Fatalf("expected add-task-tags URL update, got %s", joined)
		}
		if !strings.Contains(joined, "things:///update?auth-token=token-test") || !strings.Contains(joined, "tags=") {
			t.Fatalf("expected remove-task-tags URL update, got %s", joined)
		}
	})

	t.Run("notes commands success", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--name", "task-a", "--notes", "hello"})
		if err := setNotes.Execute(); err != nil {
			t.Fatalf("set-task-notes failed: %v", err)
		}

		appendNotes := newAppendTaskNotesCmd()
		appendNotes.SetArgs([]string{"--name", "task-a", "--notes", "world", "--separator", " | "})
		if err := appendNotes.Execute(); err != nil {
			t.Fatalf("append-task-notes failed: %v", err)
		}
	})

	t.Run("metadata commands by id", func(t *testing.T) {
		fr := &fakeRunner{output: "ok"}
		setupTestRuntimeWithDB(t, fr)

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--id", "task-1", "--notes", "hello"})
		if err := setNotes.Execute(); err != nil {
			t.Fatalf("set-task-notes --id failed: %v", err)
		}

		scripts := fr.allScripts()
		if len(scripts) == 0 || !strings.Contains(scripts[0], `whose id is "task-1"`) {
			t.Fatalf("unexpected id-based metadata script: %#v", scripts)
		}
	})

	t.Run("set-task-date success and clear", func(t *testing.T) {
		fr := &fakeRunner{runFn: func(script string) (string, error) {
			if strings.Contains(script, `set out to "ID: "`) {
				return strings.Join([]string{
					"ID: task-1",
					"Name: task-a",
					"Type: task",
					"Statut: open",
					"Due: ",
					"Deadline: ",
					"Completed on: ",
					"Created on: ",
					"Tags: ",
					"Notes: ",
				}, "\n"), nil
			}
			return "ok", nil
		}}
		setupTestRuntimeWithDB(t, fr)

		setDate := newSetTaskDateCmd()
		setDate.SetArgs([]string{"--name", "task-a", "--due", "2026-03-06", "--deadline", "2026-03-07"})
		if err := setDate.Execute(); err != nil {
			t.Fatalf("set-task-date failed: %v", err)
		}

		clearDate := newSetTaskDateCmd()
		clearDate.SetArgs([]string{"--name", "task-a", "--clear-due", "--clear-deadline"})
		if err := clearDate.Execute(); err != nil {
			t.Fatalf("set-task-date --clear failed: %v", err)
		}

		scripts := strings.Join(fr.allScripts(), "\n")
		if !strings.Contains(scripts, `set month of dueDateValue to March`) || !strings.Contains(scripts, `schedule t for dueDateValue`) {
			t.Fatalf("expected due date AppleScript update, got %s", scripts)
		}
		if !strings.Contains(scripts, "things:///update?auth-token=token-test") || !strings.Contains(scripts, "&deadline=2026-03-07") {
			t.Fatalf("expected deadline URL update, got %s", scripts)
		}
		if !strings.Contains(scripts, "things:///update?auth-token=token-test") || !strings.Contains(scripts, "&when=") {
			t.Fatalf("expected due clear URL update, got %s", scripts)
		}
	})

	t.Run("validation branches", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntime(t, t.TempDir(), fr)

		setTaskTags := newSetTaskTagsCmd()
		setTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err := setTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		setDate := newSetTaskDateCmd()
		setDate.SetArgs([]string{"--name", "task-a"})
		err = setDate.Execute()
		if err == nil || !strings.Contains(err.Error(), "provide --due, --deadline, --clear-due, or --clear-deadline") {
			t.Fatalf("unexpected error: %v", err)
		}

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--name", "task-a", "--notes", "   "})
		err = setNotes.Execute()
		if err == nil || !strings.Contains(err.Error(), "--notes is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		appendNotes := newAppendTaskNotesCmd()
		appendNotes.SetArgs([]string{"--name", "task-a", "--notes", "   "})
		err = appendNotes.Execute()
		if err == nil || !strings.Contains(err.Error(), "--notes is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		setTags := newSetTagsCmd()
		setTags.SetArgs([]string{"--name", "task-a", "--tags", "   "})
		err = setTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "--tags is required") {
			t.Fatalf("unexpected error: %v", err)
		}

		addTaskTags := newAddTaskTagsCmd()
		addTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err = addTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		removeTaskTags := newRemoveTaskTagsCmd()
		removeTaskTags.SetArgs([]string{"--name", "task-a", "--tags", " , "})
		err = removeTaskTags.Execute()
		if err == nil || !strings.Contains(err.Error(), "specify at least one tag in --tags") {
			t.Fatalf("unexpected error: %v", err)
		}

		setDateInvalid := newSetTaskDateCmd()
		setDateInvalid.SetArgs([]string{"--name", "task-a", "--deadline", "invalid"})
		err = setDateInvalid.Execute()
		if err == nil {
			t.Fatal("expected invalid deadline format error")
		}
	})

	t.Run("clear deadline requires token", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)
		t.Setenv("THINGS_AUTH_TOKEN", "")
		config.authToken = ""
		cmd := newSetTaskDateCmd()
		cmd.SetArgs([]string{"--name", "task-a", "--clear-deadline"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "auth-token is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("metadata target validation requires exactly one selector", func(t *testing.T) {
		fr := &fakeRunner{}
		setupTestRuntimeWithDB(t, fr)

		setNotes := newSetTaskNotesCmd()
		setNotes.SetArgs([]string{"--notes", "hello"})
		err := setNotes.Execute()
		if err == nil || !strings.Contains(err.Error(), "exactly one of --name or --id") {
			t.Fatalf("unexpected selector error: %v", err)
		}
	})
}
